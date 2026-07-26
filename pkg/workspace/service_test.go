package workspace_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	testingexec "github.com/zkhvan/z/pkg/exec/testing"
	"github.com/zkhvan/z/pkg/workspace"
)

// serviceTestDir holds paths for a test run.
type serviceTestDir struct {
	root       string
	workspaces string
	projects   string
	configDir  string
}

func setupServiceTestDir(t *testing.T) serviceTestDir {
	t.Helper()
	root := t.TempDir()
	td := serviceTestDir{
		root:       root,
		workspaces: filepath.Join(root, "workspaces"),
		projects:   filepath.Join(root, "projects"),
		configDir:  filepath.Join(root, "config"),
	}
	assert.NoError(t, os.MkdirAll(td.workspaces, 0o700))
	assert.NoError(t, os.MkdirAll(td.projects, 0o700))
	assert.NoError(t, os.MkdirAll(td.configDir, 0o700))
	return td
}

func setupServiceConfig(t *testing.T, td serviceTestDir, rawCfg string) cmdutil.Config {
	t.Helper()
	rawCfg = strings.ReplaceAll(rawCfg, "$WORKSPACESDIR", td.workspaces)
	rawCfg = strings.ReplaceAll(rawCfg, "$PROJECTSDIR", td.projects)
	cfgPath := filepath.Join(td.configDir, "config.yaml")
	if rawCfg != "" {
		assert.NoError(t, os.WriteFile(cfgPath, []byte(rawCfg), 0o600))
	}
	cfg, err := config.NewWithDir(td.configDir)
	assert.NoError(t, err)
	return cfg
}

func TestCreate_WritesManifest(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	members := []workspace.Member{
		{Repo: "zkhvan/api", Branch: "feature/login", BaseRef: ""},
	}

	err = svc.Create(context.Background(), "login-feature", members)
	assert.NoError(t, err)

	manifestPath := filepath.Join(td.workspaces, "login-feature", ".z", "instance.yaml")
	data, err := os.ReadFile(manifestPath)
	assert.NoError(t, err)

	content := string(data)
	if !strings.Contains(content, "version: 1") {
		t.Fatalf("manifest missing 'version: 1':\n%s", content)
	}
	if !strings.Contains(content, "repo: zkhvan/api") {
		t.Fatalf("manifest missing member repo:\n%s", content)
	}
	if !strings.Contains(content, "branch: feature/login") {
		t.Fatalf("manifest missing member branch:\n%s", content)
	}
	// name and status must NOT be stored
	if strings.Contains(content, "login-feature") {
		t.Fatalf("manifest must not contain the instance name:\n%s", content)
	}
	if strings.Contains(content, "status") {
		t.Fatalf("manifest must not contain status:\n%s", content)
	}
}

func TestCreate_MultipleMembers_OrderPreserved(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	members := []workspace.Member{
		{Repo: "zkhvan/api", Branch: "feature/auth"},
		{Repo: "zkhvan/web", Branch: "feature/auth"},
		{Repo: "zkhvan/docs", Branch: "feature/auth"},
	}

	assert.NoError(t, svc.Create(context.Background(), "auth", members))

	data, err := os.ReadFile(filepath.Join(td.workspaces, "auth", ".z", "instance.yaml"))
	assert.NoError(t, err)

	content := string(data)
	apiIdx := strings.Index(content, "zkhvan/api")
	webIdx := strings.Index(content, "zkhvan/web")
	docsIdx := strings.Index(content, "zkhvan/docs")

	if apiIdx < 0 || webIdx < 0 || docsIdx < 0 {
		t.Fatalf("one or more members missing from manifest:\n%s", content)
	}
	if apiIdx >= webIdx || webIdx >= docsIdx {
		t.Fatalf("members not written in declaration order:\n%s", content)
	}
}

func TestCreate_ZeroMembers_Error(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, "")
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	err = svc.Create(context.Background(), "empty", nil)
	if err == nil {
		t.Fatal("expected error for zero members, got nil")
	}
}

func TestCreate_InvalidName_Error(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, "")
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	err = svc.Create(context.Background(), ".hidden", []workspace.Member{
		{Repo: "owner/repo", Branch: "main"},
	})
	if err == nil {
		t.Fatal("expected error for invalid name, got nil")
	}
}

func TestCreate_ExistingDir_Error(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	members := []workspace.Member{{Repo: "owner/repo", Branch: "main"}}
	assert.NoError(t, svc.Create(context.Background(), "dupe", members))

	// Second create with the same name must fail.
	err = svc.Create(context.Background(), "dupe", members)
	if err == nil {
		t.Fatal("expected error for existing workspace directory, got nil")
	}
}

func TestCreate_UnsafeRepoID_Error(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	err = svc.Create(context.Background(), "unsafe", []workspace.Member{
		{Repo: "owner/..", Branch: "main"},
	})
	if err == nil {
		t.Fatal("expected error for unsafe repo ID, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(td.workspaces, "unsafe")); !os.IsNotExist(statErr) {
		t.Fatalf("unsafe workspace should not be created: %v", statErr)
	}
}

func TestCreate_UnsafeBranch_Error(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	err = svc.Create(context.Background(), "unsafe", []workspace.Member{
		{Repo: "owner/repo", Branch: "-feature"},
	})
	if err == nil {
		t.Fatal("expected error for unsafe branch, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(td.workspaces, "unsafe")); !os.IsNotExist(statErr) {
		t.Fatalf("unsafe workspace should not be created: %v", statErr)
	}
}

func TestCreate_UnsafeBaseRef_Error(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	err = svc.Create(context.Background(), "unsafe", []workspace.Member{
		{Repo: "owner/repo", Branch: "feature", BaseRef: "-base"},
	})
	if err == nil {
		t.Fatal("expected error for unsafe base ref, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(td.workspaces, "unsafe")); !os.IsNotExist(statErr) {
		t.Fatalf("unsafe workspace should not be created: %v", statErr)
	}
}

func TestCreate_DuplicateBaseName_Error(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	// Both zkhvan/api and other/api resolve to base name "api".
	err = svc.Create(context.Background(), "conflict", []workspace.Member{
		{Repo: "zkhvan/api", Branch: "main"},
		{Repo: "other/api", Branch: "main"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate base name, got nil")
	}
}

func TestCreate_MalformedMember_Error(t *testing.T) {
	// ParseMember is called by the cmd layer before Create; verify it errors.
	_, parseErr := workspace.ParseMember("no-at-sign")
	if parseErr == nil {
		t.Fatal("expected ParseMember to error on malformed string")
	}
}

func TestList_DiscoversSortedInstances(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	// Create two instances out of alphabetical order.
	assert.NoError(t, svc.Create(context.Background(), "zebra", []workspace.Member{
		{Repo: "owner/repo", Branch: "main"},
	}))
	assert.NoError(t, svc.Create(context.Background(), "alpha", []workspace.Member{
		{Repo: "owner/other", Branch: "feat"},
	}))

	instances, err := svc.List(context.Background())
	assert.NoError(t, err)

	if len(instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(instances))
	}
	if instances[0].Name != "alpha" {
		t.Fatalf("expected first instance %q, got %q", "alpha", instances[0].Name)
	}
	if instances[1].Name != "zebra" {
		t.Fatalf("expected second instance %q, got %q", "zebra", instances[1].Name)
	}

	// Members should be populated.
	if len(instances[0].Members) != 1 || instances[0].Members[0].Repo != "owner/other" {
		t.Fatalf("unexpected members for alpha: %+v", instances[0].Members)
	}
}

func TestList_SkipsUnsafeManifestWithoutRunningCommands(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	manifestDir := filepath.Join(td.workspaces, "unsafe", ".z")
	assert.NoError(t, os.MkdirAll(manifestDir, 0o700))
	manifest := []byte("version: 1\nmembers:\n  - repo: ../../outside\n    branch: main\n")
	assert.NoError(t, os.WriteFile(filepath.Join(manifestDir, "instance.yaml"), manifest, 0o600))

	fake := &testingexec.FakeExec{}
	svc, err := workspace.NewService(cfg, workspace.WithExecutor(fake))
	assert.NoError(t, err)

	instances, err := svc.List(context.Background())
	assert.NoError(t, err)
	if len(instances) != 0 {
		t.Fatalf("expected unsafe manifest to be skipped, got %+v", instances)
	}
	if fake.CommandCalls != 0 {
		t.Fatalf("expected no commands for unsafe manifest, got %d", fake.CommandCalls)
	}
}

func TestList_SkipsNonInstanceDirs(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	// Plant a directory that has no .z/instance.yaml.
	assert.NoError(t, os.MkdirAll(filepath.Join(td.workspaces, "stray"), 0o700))

	// Create one real instance.
	assert.NoError(t, svc.Create(context.Background(), "real", []workspace.Member{
		{Repo: "owner/repo", Branch: "main"},
	}))

	instances, err := svc.List(context.Background())
	assert.NoError(t, err)

	if len(instances) != 1 {
		t.Fatalf("expected 1 instance (stray skipped), got %d", len(instances))
	}
	if instances[0].Name != "real" {
		t.Fatalf("expected instance %q, got %q", "real", instances[0].Name)
	}
}

func TestList_NewStatus(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	assert.NoError(t, svc.Create(context.Background(), "ws", []workspace.Member{
		{Repo: "owner/repo", Branch: "main"},
	}))

	instances, err := svc.List(context.Background())
	assert.NoError(t, err)

	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(instances))
	}
	if instances[0].Status != workspace.InstanceStatusNew {
		t.Fatalf("expected status %q, got %q", workspace.InstanceStatusNew, instances[0].Status)
	}
	if instances[0].Members[0].State != workspace.MemberStateUnknown {
		t.Fatalf("expected member state %q, got %q", workspace.MemberStateUnknown, instances[0].Members[0].State)
	}
}

func TestList_ArchivedStatus(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "ws", []workspace.Member{
		{Repo: "owner/repo", Branch: "main"},
	}))
	assert.NoError(t, os.WriteFile(filepath.Join(td.workspaces, "ws", ".z", "materialized"), nil, 0o600))

	instances, err := svc.List(context.Background())
	assert.NoError(t, err)

	if instances[0].Status != workspace.InstanceStatusArchived {
		t.Fatalf("expected status %q, got %q", workspace.InstanceStatusArchived, instances[0].Status)
	}
}
