package project_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/google/go-cmp/cmp"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/exec"
	testingexec "github.com/zkhvan/z/pkg/exec/testing"
	"github.com/zkhvan/z/pkg/project"
)

func TestList(t *testing.T) {
	type remoteRepo struct {
		owner string
		repo  string
	}

	tests := map[string]struct {
		cfg              string
		opts             *project.ListOptions
		err              error
		local            []string
		remote           map[string][]remoteRepo
		expectedProjects []project.Project
	}{
		"no config should list no projects": {
			cfg:              "",
			opts:             &project.ListOptions{},
			err:              nil,
			expectedProjects: []project.Project{},
		},
		"empty config should list no projects": {
			cfg:              "{}",
			opts:             &project.ListOptions{},
			err:              nil,
			expectedProjects: []project.Project{},
		},
		"no arguments should list no projects": {
			cfg: heredoc.Doc(`
				projects:
				  root: $PROJECTSDIR
			`),
			opts:             &project.ListOptions{},
			err:              nil,
			expectedProjects: []project.Project{},
		},
		"local projects should be listed": {
			cfg: heredoc.Doc(`
				projects:
				  root: $PROJECTSDIR
			`),
			opts: &project.ListOptions{Local: true},
			err:  nil,
			local: []string{
				"owner/repo",
			},
			expectedProjects: []project.Project{
				{
					LocalID:      "owner/repo",
					RemoteID:     "owner/repo",
					AbsolutePath: filepath.Join("$PROJECTSDIR", "owner", "repo"),
				},
			},
		},
		"remote projects should be listed": {
			cfg: heredoc.Doc(`
				projects:
				  root: $PROJECTSDIR
				  remote_patterns:
				    - owner/*
			`),
			opts: &project.ListOptions{Remote: true},
			err:  nil,
			remote: map[string][]remoteRepo{
				"owner": {
					{
						owner: "owner",
						repo:  "repo",
					},
				},
			},
			expectedProjects: []project.Project{
				{
					LocalID:      "owner/repo",
					RemoteID:     "owner/repo",
					AbsolutePath: filepath.Join("$PROJECTSDIR", "owner", "repo"),
				},
			},
		},
		"local and remote projects should be listed": {
			cfg: heredoc.Doc(`
				projects:
				  root: $PROJECTSDIR
				  remote_patterns:
				    - owner/*
			`),
			opts: &project.ListOptions{Local: true, Remote: true},
			local: []string{
				"owner/local",
			},
			remote: map[string][]remoteRepo{
				"owner": {
					{
						owner: "owner",
						repo:  "remote",
					},
				},
			},
			expectedProjects: []project.Project{
				{
					LocalID:      "owner/local",
					RemoteID:     "owner/local",
					AbsolutePath: filepath.Join("$PROJECTSDIR", "owner", "local"),
				},
				{
					LocalID:      "owner/remote",
					RemoteID:     "owner/remote",
					AbsolutePath: filepath.Join("$PROJECTSDIR", "owner", "remote"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			td := setupTestDir(t)
			cfg := setupConfig(t, td, test.cfg)

			// Setup local projects
			for _, dir := range test.local {
				err := os.MkdirAll(filepath.Join(td.projects, dir, ".git"), 0o700)
				assert.NoError(t, err)
			}

			// Setup remote projects
			fakeexec := &testingexec.FakeExec{}
			for owner, ownerRepos := range test.remote {
				fakeexec.CommandScript = append(fakeexec.CommandScript, func(_ string, _ ...string) exec.Cmd {
					fakeCmd := testingexec.NewFakeCmd("gh", "repo", "list", owner, "--json", "owner,name")
					fakeCmd.OutputScripts = []testingexec.FakeAction{
						func() ([]byte, []byte, error) {
							var responses []string
							for _, ownerRepo := range ownerRepos {
								response := fmt.Sprintf(`{"owner":{"login":"%s"},"name":"%s"}`, ownerRepo.owner, ownerRepo.repo)
								responses = append(responses, response)
							}

							return []byte("[" + strings.Join(responses, "\n") + "]"), nil, nil
						},
					}
					return fakeCmd
				})
			}

			service, err := project.NewService(
				cfg,
				project.WithCacheDir(td.cache),
				project.WithExecutor(fakeexec),
			)
			assert.NoError(t, err)

			projects, err := service.ListProjects(context.Background(), test.opts)
			assert.NoError(t, err)

			for i, p := range test.expectedProjects {
				test.expectedProjects[i].AbsolutePath = strings.ReplaceAll(p.AbsolutePath, "$PROJECTSDIR", td.projects)
			}

			if diff := cmp.Diff(projects, test.expectedProjects); diff != "" {
				t.Fatalf("projects mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// testDir represents the test directory structure:
//
//	$TESTDIR
//	├── projects
//	│   └── owner
//	│       └── repo
//	│           └── .git
//	├── config
//	│   └── config.yaml
//	└── cache
//	    └── project.remote-$TIMESTAMP
type testDir struct {
	root     string
	projects string
	config   string
	cache    string
}

func setupTestDir(t *testing.T) testDir {
	t.Helper()

	dirs := testDir{
		root: t.TempDir(),
	}

	dirs.projects = filepath.Join(dirs.root, "projects")
	dirs.config = filepath.Join(dirs.root, "config")
	dirs.cache = filepath.Join(dirs.root, "cache")

	var errs []error
	errs = append(errs, os.MkdirAll(dirs.projects, 0o700))
	errs = append(errs, os.MkdirAll(dirs.config, 0o700))
	errs = append(errs, os.MkdirAll(dirs.cache, 0o700))

	for _, err := range errs {
		if err != nil {
			t.Fatalf("failed to create test directory: %s", err)
		}
	}

	return dirs
}

func setupConfig(t *testing.T, td testDir, rawCfg string) cmdutil.Config {
	cfgPath := filepath.Join(td.config, "config.yaml")

	rawCfg = strings.ReplaceAll(rawCfg, "$PROJECTSDIR", td.projects)
	rawCfg = strings.ReplaceAll(rawCfg, "$CONFIGDIR", td.config)
	rawCfg = strings.ReplaceAll(rawCfg, "$CACHEDIR", td.cache)

	if len(rawCfg) > 0 {
		err := os.WriteFile(cfgPath, []byte(rawCfg), 0o600)
		assert.NoError(t, err)
	}

	cfg, err := config.NewWithDir(td.config)
	assert.NoError(t, err)

	return cfg
}
