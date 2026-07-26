package workspace_test

import (
	"context"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/exec"
	testingexec "github.com/zkhvan/z/pkg/exec/testing"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestMaterialize_creates_worktree_for_single_member(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login", BaseRef: "main"},
	}}))

	err = svc.Materialize(context.Background(), "login")
	assert.NoError(t, err)

	worktree := filepath.Join(td.workspaces, "login", "repo")
	assertGitBranch(t, worktree, "feature/login")
	assertFileExists(t, filepath.Join(worktree, "README.md"))
}

func TestMaterialize_rejects_repo_path_escape_before_running_commands(t *testing.T) {
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

	err = svc.Materialize(context.Background(), "unsafe")
	assertErrorContains(t, err, "manifest is invalid")
	if fake.CommandCalls != 0 {
		t.Fatalf("expected no commands for unsafe manifest, got %d", fake.CommandCalls)
	}
}

func TestMaterialize_resolves_empty_base_ref_from_origin_head(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	cloneCanonicalFromBareRemote(t, td, "owner", "repo")

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login"},
	}}))

	err = svc.Materialize(context.Background(), "login")
	assert.NoError(t, err)

	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo"), "feature/login")
}

func TestList_MaterializedStatusAndDirtyState(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login", BaseRef: "main"},
	}}))
	assert.NoError(t, svc.Materialize(context.Background(), "login"))

	instances, err := svc.List(context.Background())
	assert.NoError(t, err)
	assertInstanceState(t, instances, "login", workspace.InstanceStatusMaterialized, workspace.MemberStateClean)

	worktree := filepath.Join(td.workspaces, "login", "repo")
	writeErr := os.WriteFile(filepath.Join(worktree, "README.md"), []byte("changed\n"), 0o600)
	if writeErr != nil {
		t.Fatalf("dirty worktree: %v", writeErr)
	}

	instances, err = svc.List(context.Background())
	assert.NoError(t, err)
	assertInstanceState(t, instances, "login", workspace.InstanceStatusMaterialized, workspace.MemberStateDirty)
}

func TestList_standalone_repository_at_member_path_is_not_materialized(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "standalone", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/list", BaseRef: "main"},
	}}))

	memberPath := filepath.Join(td.workspaces, "standalone", "repo")
	initGitRepo(t, memberPath)
	runGit(t, "-C", memberPath, "branch", "-m", "feature/list")

	instances, err := svc.List(context.Background())
	assert.NoError(t, err)
	assertInstanceState(t, instances, "standalone", workspace.InstanceStatusNew, workspace.MemberStateUnknown)
}

func TestMaterialize_existing_correct_worktree_is_skipped(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login", BaseRef: "main"},
	}}))
	assert.NoError(t, svc.Materialize(context.Background(), "login"))

	err = svc.Materialize(context.Background(), "login")
	assert.NoError(t, err)

	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo"), "feature/login")
}

func TestMaterialize_existing_non_worktree_directory_errors(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login", BaseRef: "main"},
	}}))
	mkdirErr := os.MkdirAll(filepath.Join(td.workspaces, "login", "repo"), 0o700)
	if mkdirErr != nil {
		t.Fatalf("seed non-worktree dir: %v", mkdirErr)
	}

	err = svc.Materialize(context.Background(), "login")
	assertErrorContains(t, err, "exists but is not a worktree")
}

func TestMaterialize_existing_wrong_branch_worktree_errors(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	canonical := filepath.Join(td.projects, "owner", "repo")
	initGitRepo(t, canonical)

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/other", BaseRef: "main"},
	}}))
	worktree := filepath.Join(td.workspaces, "login", "repo")
	runGit(t, "-C", canonical, "worktree", "add", "-b", "feature/login", worktree, "main")

	err = svc.Materialize(context.Background(), "login")
	assertErrorContains(t, err, "is on branch")
	assertErrorContains(t, err, "feature/login")
	assertErrorContains(t, err, "feature/other")
}

func TestMaterialize_branch_checked_out_elsewhere_errors_with_conflict_path(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	canonical := filepath.Join(td.projects, "owner", "repo")
	initGitRepo(t, canonical)

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "main", BaseRef: "main"},
	}}))

	err = svc.Materialize(context.Background(), "login")
	assertErrorContains(t, err, "already checked out")
	assertErrorContains(t, err, canonical)
}

func TestMaterialize_auto_clones_missing_canonical_clone(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	canonical := filepath.Join(td.projects, "owner", "repo")
	worktree := filepath.Join(td.workspaces, "login", "repo")
	fake := &testingexec.FakeExec{CommandScript: []testingexec.FakeCommandAction{
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd("gh", "repo", "clone", "https://github.com/owner/repo", canonical)
			cmd.CombinedOutputScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return []byte("cloned"), nil, nil },
			}
			return cmd
		},
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd("git", "-C", canonical, "worktree", "list", "--porcelain")
			cmd.OutputScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) {
					return []byte("worktree " + canonical + "\nbranch refs/heads/main\n"), nil, nil
				},
			}
			return cmd
		},
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd(
				"git", "-C", canonical, "show-ref", "--verify", "--quiet", "refs/heads/feature/login",
			)
			cmd.RunScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return nil, nil, fakeExitError(1) },
			}
			return cmd
		},
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd(
				"git", "-C", canonical, "worktree", "add", "-b", "feature/login", "--", worktree, "main",
			)
			cmd.CombinedOutputScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return nil, nil, nil },
			}
			return cmd
		},
	}}

	svc, err := workspace.NewService(cfg, workspace.WithExecutor(fake))
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login", BaseRef: "main"},
	}}))

	err = svc.Materialize(context.Background(), "login")
	assert.NoError(t, err)
	if fake.CommandCalls != 4 {
		t.Fatalf("expected 4 commands, got %d", fake.CommandCalls)
	}
}

func TestMaterialize_reuses_existing_canonical_clone(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	canonical := filepath.Join(td.projects, "owner", "repo")
	worktree := filepath.Join(td.workspaces, "login", "repo")
	if err := os.MkdirAll(canonical, 0o700); err != nil {
		t.Fatalf("seed canonical clone: %v", err)
	}
	fake := &testingexec.FakeExec{CommandScript: []testingexec.FakeCommandAction{
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd("git", "-C", canonical, "worktree", "list", "--porcelain")
			cmd.OutputScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) {
					return []byte("worktree " + canonical + "\nbranch refs/heads/main\n"), nil, nil
				},
			}
			return cmd
		},
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd(
				"git", "-C", canonical, "show-ref", "--verify", "--quiet", "refs/heads/feature/login",
			)
			cmd.RunScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return nil, nil, fakeExitError(1) },
			}
			return cmd
		},
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd(
				"git", "-C", canonical, "worktree", "add", "-b", "feature/login", "--", worktree, "main",
			)
			cmd.CombinedOutputScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return nil, nil, nil },
			}
			return cmd
		},
	}}

	svc, err := workspace.NewService(cfg, workspace.WithExecutor(fake))
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login", BaseRef: "main"},
	}}))

	err = svc.Materialize(context.Background(), "login")
	assert.NoError(t, err)
	if fake.CommandCalls != 3 {
		t.Fatalf("expected 3 commands, got %d", fake.CommandCalls)
	}
}

func TestMaterialize_fail_fast_leaves_completed_worktrees_and_rerun_completes(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	repo1 := filepath.Join(td.projects, "owner", "repo1")
	repo2 := filepath.Join(td.projects, "owner", "repo2")
	initGitRepo(t, repo1)
	initGitRepo(t, repo2)

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo1", Branch: "feature/login", BaseRef: "main"},
		{Repo: "owner/repo2", Branch: "main", BaseRef: "main"},
	}}))

	err = svc.Materialize(context.Background(), "login")
	assertErrorContains(t, err, "already checked out")
	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo1"), "feature/login")
	assertPathMissing(t, filepath.Join(td.workspaces, "login", "repo2"))
	assertPathMissing(t, filepath.Join(td.workspaces, "login", ".z", "materialized"))
	instances, err := svc.List(context.Background())
	assert.NoError(t, err)
	assertPartialInstance(t, instances, "login")

	runGit(t, "-C", repo2, "checkout", "-b", "spare")

	err = svc.Materialize(context.Background(), "login")
	assert.NoError(t, err)
	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo1"), "feature/login")
	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo2"), "main")
	assertFileExists(t, filepath.Join(td.workspaces, "login", ".z", "materialized"))
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := osexec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, "init", "-b", "main", dir)
	runGit(t, "-C", dir, "config", "user.name", "z test")
	runGit(t, "-C", dir, "config", "user.email", "z@example.test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o600); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGit(t, "-C", dir, "add", "README.md")
	runGit(t, "-C", dir, "commit", "-m", "initial")
}

func cloneCanonicalFromBareRemote(t *testing.T, td serviceTestDir, owner, repo string) {
	t.Helper()
	source := filepath.Join(td.root, "source", owner, repo)
	initGitRepo(t, source)
	bare := filepath.Join(td.root, "remotes", owner, repo+".git")
	runGit(t, "clone", "--bare", source, bare)
	canonical := filepath.Join(td.projects, owner, repo)
	if err := os.MkdirAll(filepath.Dir(canonical), 0o700); err != nil {
		t.Fatalf("create canonical parent: %v", err)
	}
	runGit(t, "clone", bare, canonical)
}

type fakeExitError int

func (e fakeExitError) Error() string {
	return "command exited"
}

func (e fakeExitError) ExitCode() int {
	return int(e)
}

func runGit(t *testing.T, args ...string) string {
	t.Helper()
	cmd := osexec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func assertGitBranch(t *testing.T, worktree, want string) {
	t.Helper()
	got := runGit(t, "-C", worktree, "branch", "--show-current")
	if got != want {
		t.Fatalf("worktree %s branch = %q, want %q", worktree, got, want)
	}
}

func assertInstanceState(
	t *testing.T,
	instances []workspace.Instance,
	name string,
	status workspace.InstanceStatus,
	memberState workspace.MemberState,
) {
	t.Helper()
	for _, inst := range instances {
		if inst.Name != name {
			continue
		}
		if inst.Status != status {
			t.Fatalf("instance %q status = %q, want %q", name, inst.Status, status)
		}
		if len(inst.Members) != 1 {
			t.Fatalf("instance %q member count = %d, want 1", name, len(inst.Members))
		}
		if inst.Members[0].State != memberState {
			t.Fatalf("instance %q member state = %q, want %q", name, inst.Members[0].State, memberState)
		}
		return
	}
	t.Fatalf("instance %q not found in %+v", name, instances)
}

func assertPartialInstance(t *testing.T, instances []workspace.Instance, name string) {
	t.Helper()
	for _, inst := range instances {
		if inst.Name != name {
			continue
		}
		if inst.Status != workspace.InstanceStatusPartial {
			t.Fatalf("instance %q status = %q, want %q", name, inst.Status, workspace.InstanceStatusPartial)
		}
		if len(inst.Members) != 2 {
			t.Fatalf("instance %q member count = %d, want 2", name, len(inst.Members))
		}
		if inst.Members[0].State != workspace.MemberStateClean {
			t.Fatalf("first member state = %q, want clean", inst.Members[0].State)
		}
		if inst.Members[1].State != workspace.MemberStateUnknown {
			t.Fatalf("second member state = %q, want unknown", inst.Members[1].State)
		}
		return
	}
	t.Fatalf("instance %q not found in %+v", name, instances)
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, stat error: %v", path, err)
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}
