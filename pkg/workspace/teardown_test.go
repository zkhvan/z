package workspace_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestArchive_removes_worktree_and_preserves_instance(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	canonical := filepath.Join(td.projects, "owner", "repo")
	initGitRepo(t, canonical)
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})

	err := svc.Archive(context.Background(), "login", workspace.TeardownOptions{})
	assert.NoError(t, err)

	assertPathMissing(t, filepath.Join(td.workspaces, "login", "repo"))
	assertFileExists(t, filepath.Join(td.workspaces, "login", ".z", "instance.yaml"))
	assertNoWorktreeRegistrations(t, canonical)
}

func TestArchive_reports_archived_status(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})

	assert.NoError(t, svc.Archive(context.Background(), "login", workspace.TeardownOptions{}))

	instances, err := svc.List(context.Background())
	assert.NoError(t, err)
	assertInstanceState(t, instances, "login", workspace.InstanceStatusArchived, workspace.MemberStateUnknown)
}

func TestArchive_archived_instance_rematerializes(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	assert.NoError(t, svc.Archive(context.Background(), "login", workspace.TeardownOptions{}))

	err := svc.Materialize(context.Background(), "login")
	assert.NoError(t, err)

	worktree := filepath.Join(td.workspaces, "login", "repo")
	assertGitBranch(t, worktree, "feature/login")
	assertFileExists(t, filepath.Join(worktree, "README.md"))
}

func TestArchive_is_idempotent(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	assert.NoError(t, svc.Archive(context.Background(), "login", workspace.TeardownOptions{}))

	err := svc.Archive(context.Background(), "login", workspace.TeardownOptions{})
	assert.NoError(t, err)
}

func TestArchive_never_materialized_instance_is_a_noop(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))
	assert.NoError(t, svc.Create(context.Background(), "login", []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login", BaseRef: "main"},
	}))

	err := svc.Archive(context.Background(), "login", workspace.TeardownOptions{})
	assert.NoError(t, err)

	assertFileExists(t, filepath.Join(td.workspaces, "login", ".z", "instance.yaml"))
}

func TestArchive_dirty_member_aborts_before_removing_anything(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	repo1 := filepath.Join(td.projects, "owner", "repo1")
	repo2 := filepath.Join(td.projects, "owner", "repo2")
	initGitRepo(t, repo1)
	initGitRepo(t, repo2)
	createAndMaterialize(t, svc, "login",
		workspace.Member{Repo: "owner/repo1", Branch: "feature/login", BaseRef: "main"},
		workspace.Member{Repo: "owner/repo2", Branch: "feature/login", BaseRef: "main"},
	)
	dirty(t, filepath.Join(td.workspaces, "login", "repo2"))

	err := svc.Archive(context.Background(), "login", workspace.TeardownOptions{})

	assertErrorContains(t, err, "cannot archive workspace")
	assertErrorContains(t, err, "1 member with uncommitted or untracked changes")
	assertErrorContains(t, err, "owner/repo2")
	assertErrorContains(t, err, "use --force")
	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo1"), "feature/login")
	assertWorktreeRegistered(t, repo1, filepath.Join(td.workspaces, "login", "repo1"))
}

func TestArchive_reports_every_dirty_member(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo1"))
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo2"))
	createAndMaterialize(t, svc, "login",
		workspace.Member{Repo: "owner/repo1", Branch: "feature/login", BaseRef: "main"},
		workspace.Member{Repo: "owner/repo2", Branch: "feature/login", BaseRef: "main"},
	)
	dirty(t, filepath.Join(td.workspaces, "login", "repo1"))
	dirty(t, filepath.Join(td.workspaces, "login", "repo2"))

	err := svc.Archive(context.Background(), "login", workspace.TeardownOptions{})

	assertErrorContains(t, err, "2 members with uncommitted or untracked changes")
	assertErrorContains(t, err, "owner/repo1")
	assertErrorContains(t, err, "owner/repo2")
}

func TestArchive_ignores_gitignored_files(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	canonical := filepath.Join(td.projects, "owner", "repo")
	initGitRepo(t, canonical)
	writeFile(t, filepath.Join(canonical, ".gitignore"), "node_modules/\n")
	runGit(t, "-C", canonical, "add", ".gitignore")
	runGit(t, "-C", canonical, "commit", "-m", "ignore")
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	worktree := filepath.Join(td.workspaces, "login", "repo")
	writeFile(t, filepath.Join(worktree, "node_modules", "index.js"), "artifact\n")

	err := svc.Archive(context.Background(), "login", workspace.TeardownOptions{})
	assert.NoError(t, err)

	assertPathMissing(t, worktree)
}

func TestArchive_force_removes_dirty_member(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	canonical := filepath.Join(td.projects, "owner", "repo")
	initGitRepo(t, canonical)
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	dirty(t, filepath.Join(td.workspaces, "login", "repo"))

	err := svc.Archive(context.Background(), "login", workspace.TeardownOptions{Force: true})
	assert.NoError(t, err)

	assertPathMissing(t, filepath.Join(td.workspaces, "login", "repo"))
	assertNoWorktreeRegistrations(t, canonical)
}

func TestArchive_deregisters_prunable_registration(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	canonical := filepath.Join(td.projects, "owner", "repo")
	initGitRepo(t, canonical)
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	if err := os.RemoveAll(filepath.Join(td.workspaces, "login", "repo")); err != nil {
		t.Fatalf("remove worktree directory: %v", err)
	}

	err := svc.Archive(context.Background(), "login", workspace.TeardownOptions{})
	assert.NoError(t, err)

	assertNoWorktreeRegistrations(t, canonical)
}

func TestArchive_missing_canonical_clone_errors(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	assert.NoError(t, svc.Create(context.Background(), "login", []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login", BaseRef: "main"},
	}))
	orphan := filepath.Join(td.workspaces, "login", "repo")
	writeFile(t, filepath.Join(orphan, "work.txt"), "unrecoverable\n")

	err := svc.Archive(context.Background(), "login", workspace.TeardownOptions{})

	assertErrorContains(t, err, "canonical clone is missing")
	assertErrorContains(t, err, "owner/repo")
	assertFileExists(t, filepath.Join(orphan, "work.txt"))
}

func TestArchive_force_removes_orphaned_member_directory(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	assert.NoError(t, svc.Create(context.Background(), "login", []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login", BaseRef: "main"},
	}))
	orphan := filepath.Join(td.workspaces, "login", "repo")
	writeFile(t, filepath.Join(orphan, "work.txt"), "unrecoverable\n")

	err := svc.Archive(context.Background(), "login", workspace.TeardownOptions{Force: true})
	assert.NoError(t, err)

	assertPathMissing(t, orphan)
	assertFileExists(t, filepath.Join(td.workspaces, "login", ".z", "instance.yaml"))
}

func TestArchive_missing_instance_errors(t *testing.T) {
	_, svc := setupTeardownTest(t)

	err := svc.Archive(context.Background(), "missing", workspace.TeardownOptions{})

	assertErrorContains(t, err, "has not been created")
}

func TestDelete_removes_worktrees_then_instance_directory(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	canonical := filepath.Join(td.projects, "owner", "repo")
	initGitRepo(t, canonical)
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})

	err := svc.Delete(context.Background(), "login", workspace.TeardownOptions{})
	assert.NoError(t, err)

	assertPathMissing(t, filepath.Join(td.workspaces, "login"))
	assertNoWorktreeRegistrations(t, canonical)
}

func TestDelete_dirty_member_aborts_before_removing_anything(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	canonical := filepath.Join(td.projects, "owner", "repo")
	initGitRepo(t, canonical)
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	dirty(t, filepath.Join(td.workspaces, "login", "repo"))

	err := svc.Delete(context.Background(), "login", workspace.TeardownOptions{})

	assertErrorContains(t, err, "cannot delete workspace")
	assertFileExists(t, filepath.Join(td.workspaces, "login", ".z", "instance.yaml"))
	assertWorktreeRegistered(t, canonical, filepath.Join(td.workspaces, "login", "repo"))
}

func TestDelete_unmanaged_files_abort_the_delete(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	writeFile(t, filepath.Join(td.workspaces, "login", "notes.md"), "hand-written\n")
	writeFile(t, filepath.Join(td.workspaces, "login", "scratch", "todo.txt"), "later\n")

	err := svc.Delete(context.Background(), "login", workspace.TeardownOptions{})

	assertErrorContains(t, err, "files not managed by z")
	assertErrorContains(t, err, "notes.md")
	assertErrorContains(t, err, "scratch/")
	assertFileExists(t, filepath.Join(td.workspaces, "login", "notes.md"))
	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo"), "feature/login")
}

func TestDelete_force_removes_unmanaged_files(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	writeFile(t, filepath.Join(td.workspaces, "login", "notes.md"), "hand-written\n")

	err := svc.Delete(context.Background(), "login", workspace.TeardownOptions{Force: true})
	assert.NoError(t, err)

	assertPathMissing(t, filepath.Join(td.workspaces, "login"))
}

func TestDelete_reports_dirty_members_and_unmanaged_files_together(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	dirty(t, filepath.Join(td.workspaces, "login", "repo"))
	writeFile(t, filepath.Join(td.workspaces, "login", "notes.md"), "hand-written\n")

	err := svc.Delete(context.Background(), "login", workspace.TeardownOptions{})

	assertErrorContains(t, err, "uncommitted or untracked changes")
	assertErrorContains(t, err, "files not managed by z")
}

func TestDelete_missing_manifest_is_refused_even_when_forced(t *testing.T) {
	td, svc := setupTeardownTest(t)
	stray := filepath.Join(td.workspaces, "not-a-workspace")
	writeFile(t, filepath.Join(stray, "important.txt"), "keep me\n")

	err := svc.Delete(context.Background(), "not-a-workspace", workspace.TeardownOptions{Force: true})

	assertErrorContains(t, err, "has not been created")
	assertFileExists(t, filepath.Join(stray, "important.txt"))
}

func TestDelete_invalid_manifest_is_refused_even_when_forced(t *testing.T) {
	td, svc := setupTeardownTest(t)
	manifest := "version: 1\nmembers:\n  - repo: not-an-owner-repo\n    branch: main\n"
	writeFile(t, filepath.Join(td.workspaces, "broken", ".z", "instance.yaml"), manifest)

	err := svc.Delete(context.Background(), "broken", workspace.TeardownOptions{Force: true})

	assertErrorContains(t, err, "manifest is invalid")
	assertFileExists(t, filepath.Join(td.workspaces, "broken", ".z", "instance.yaml"))
}

func TestDelete_failed_worktree_removal_keeps_instance_directory(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	canonical := filepath.Join(td.projects, "owner", "repo")
	initGitRepo(t, canonical)
	createAndMaterialize(t, svc, "login", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	worktree := filepath.Join(td.workspaces, "login", "repo")
	runGit(t, "-C", canonical, "worktree", "lock", worktree)

	err := svc.Delete(context.Background(), "login", workspace.TeardownOptions{})

	assertErrorContains(t, err, "removing worktree for owner/repo")
	assertFileExists(t, filepath.Join(td.workspaces, "login", ".z", "instance.yaml"))
	assertGitBranch(t, worktree, "feature/login")
}

func TestMaterialize_prunable_conflict_names_the_remedy(t *testing.T) {
	requireGit(t)
	td, svc := setupTeardownTest(t)
	canonical := filepath.Join(td.projects, "owner", "repo")
	initGitRepo(t, canonical)
	createAndMaterialize(t, svc, "abandoned", workspace.Member{
		Repo: "owner/repo", Branch: "feature/login", BaseRef: "main",
	})
	// The instance directory goes with the manifest, so archive and delete can no
	// longer reach the registration it leaves behind.
	if err := os.RemoveAll(filepath.Join(td.workspaces, "abandoned")); err != nil {
		t.Fatalf("remove instance directory: %v", err)
	}
	assert.NoError(t, svc.Create(context.Background(), "retry", []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/login", BaseRef: "main"},
	}))

	err := svc.Materialize(context.Background(), "retry")

	assertErrorContains(t, err, "registered to a missing worktree")
	assertErrorContains(t, err, "worktree prune")
}

func setupTeardownTest(t *testing.T) (serviceTestDir, *workspace.Service) {
	t.Helper()
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	return td, svc
}

func createAndMaterialize(t *testing.T, svc *workspace.Service, name string, members ...workspace.Member) {
	t.Helper()
	assert.NoError(t, svc.Create(context.Background(), name, members))
	assert.NoError(t, svc.Materialize(context.Background(), name))
}

func dirty(t *testing.T, worktree string) {
	t.Helper()
	writeFile(t, filepath.Join(worktree, "scratch.txt"), "untracked\n")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create parent of %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertNoWorktreeRegistrations(t *testing.T, canonical string) {
	t.Helper()
	output := runGit(t, "-C", canonical, "worktree", "list", "--porcelain")
	for _, line := range strings.Split(output, "\n") {
		path, ok := strings.CutPrefix(line, "worktree ")
		if ok && filepath.Clean(path) != filepath.Clean(canonical) {
			t.Fatalf("canonical %s still registers a worktree:\n%s", canonical, output)
		}
	}
}

func assertWorktreeRegistered(t *testing.T, canonical, worktree string) {
	t.Helper()
	output := runGit(t, "-C", canonical, "worktree", "list", "--porcelain")
	if !strings.Contains(output, filepath.Clean(worktree)) {
		t.Fatalf("canonical %s does not register %s:\n%s", canonical, worktree, output)
	}
}
