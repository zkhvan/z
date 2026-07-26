package workspace_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestCheckout_switches_to_existing_branch(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td, workspace.Member{
		Repo: "owner/repo", Branch: "feature/one", BaseRef: "main",
	})
	runGit(t, "-C", filepath.Join(td.projects, "owner", "repo"), "branch", "feature/two", "main")

	result, err := svc.Checkout(context.Background(), "login", "repo", "feature/two", workspace.CheckoutOptions{})
	assert.NoError(t, err)

	if result.Previous != "feature/one" || result.Created {
		t.Fatalf("unexpected result %+v", result)
	}
	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo"), "feature/two")
	assertManifestMember(t, td, 0, "owner/repo", "feature/two", "main")
}

func TestCheckout_create_stacks_on_current_head_not_base_ref(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td, workspace.Member{
		Repo: "owner/repo", Branch: "feature/one", BaseRef: "main",
	})

	worktree := filepath.Join(td.workspaces, "login", "repo")
	commitFile(t, worktree, "one.txt")
	stackTip := runGit(t, "-C", worktree, "rev-parse", "HEAD")

	result, err := svc.Checkout(context.Background(), "login", "repo", "feature/two", workspace.CheckoutOptions{
		Create: true,
	})
	assert.NoError(t, err)

	if !result.Created || result.Previous != "feature/one" {
		t.Fatalf("unexpected result %+v", result)
	}
	assertGitBranch(t, worktree, "feature/two")
	if got := runGit(t, "-C", worktree, "rev-parse", "HEAD"); got != stackTip {
		t.Fatalf("feature/two branched from %q, want the stack tip %q", got, stackTip)
	}
	assertManifestMember(t, td, 0, "owner/repo", "feature/two", "main")
}

func TestCheckout_create_from_explicit_base(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td, workspace.Member{
		Repo: "owner/repo", Branch: "feature/one", BaseRef: "main",
	})

	worktree := filepath.Join(td.workspaces, "login", "repo")
	commitFile(t, worktree, "one.txt")
	mainTip := runGit(t, "-C", worktree, "rev-parse", "main")

	_, err := svc.Checkout(context.Background(), "login", "repo", "feature/two", workspace.CheckoutOptions{
		Create: true,
		Base:   "main",
	})
	assert.NoError(t, err)

	if got := runGit(t, "-C", worktree, "rev-parse", "HEAD"); got != mainTip {
		t.Fatalf("feature/two branched from %q, want main %q", got, mainTip)
	}
}

func TestCheckout_base_without_create_is_rejected(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td, workspace.Member{
		Repo: "owner/repo", Branch: "feature/one", BaseRef: "main",
	})

	_, err := svc.Checkout(context.Background(), "login", "repo", "feature/two", workspace.CheckoutOptions{
		Base: "main",
	})
	assertErrorContains(t, err, "--base requires --create")
	assertManifestMember(t, td, 0, "owner/repo", "feature/one", "main")
}

func TestCheckout_missing_branch_without_create_is_rejected(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td, workspace.Member{
		Repo: "owner/repo", Branch: "feature/one", BaseRef: "main",
	})

	_, err := svc.Checkout(context.Background(), "login", "repo", "feature/typo", workspace.CheckoutOptions{})
	assertErrorContains(t, err, `branch "feature/typo" does not exist`)
	assertErrorContains(t, err, "pass --create")
	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo"), "feature/one")
	assertManifestMember(t, td, 0, "owner/repo", "feature/one", "main")
}

func TestCheckout_dirty_worktree_leaves_disk_and_manifest_unchanged(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td, workspace.Member{
		Repo: "owner/repo", Branch: "feature/one", BaseRef: "main",
	})

	worktree := filepath.Join(td.workspaces, "login", "repo")
	commitFile(t, worktree, "conflict.txt")
	runGit(t, "-C", filepath.Join(td.projects, "owner", "repo"), "branch", "feature/two", "main")
	writeFile(t, filepath.Join(worktree, "conflict.txt"), "uncommitted\n")

	_, err := svc.Checkout(context.Background(), "login", "repo", "feature/two", workspace.CheckoutOptions{})
	assertErrorContains(t, err, "switching owner/repo")
	assertGitBranch(t, worktree, "feature/one")
	assertManifestMember(t, td, 0, "owner/repo", "feature/one", "main")
}

func TestCheckout_branch_checked_out_in_another_worktree_is_refused(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td, workspace.Member{
		Repo: "owner/repo", Branch: "feature/one", BaseRef: "main",
	})

	canonical := filepath.Join(td.projects, "owner", "repo")
	runGit(t, "-C", canonical, "worktree", "add", "-b", "feature/two", filepath.Join(td.root, "elsewhere"), "main")

	_, err := svc.Checkout(context.Background(), "login", "repo", "feature/two", workspace.CheckoutOptions{})
	assertErrorContains(t, err, "already used by worktree")
	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo"), "feature/one")
	assertManifestMember(t, td, 0, "owner/repo", "feature/one", "main")
}

func TestCheckout_unmaterialized_member_is_rejected(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, serviceConfigWithRoots)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/one", BaseRef: "main"},
	}}))

	_, err = svc.Checkout(context.Background(), "login", "repo", "feature/two", workspace.CheckoutOptions{})
	assertErrorContains(t, err, "is not materialized")
	assertManifestMember(t, td, 0, "owner/repo", "feature/one", "main")
}

func TestCheckout_unknown_member_lists_members(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td,
		workspace.Member{Repo: "owner/api", Branch: "feature/one", BaseRef: "main"},
		workspace.Member{Repo: "owner/web", Branch: "feature/one", BaseRef: "main"},
	)

	_, err := svc.Checkout(context.Background(), "login", "docs", "feature/two", workspace.CheckoutOptions{})
	assertErrorContains(t, err, `no member "docs"`)
	assertErrorContains(t, err, "api (owner/api), web (owner/web)")
}

func TestCheckout_matches_member_by_full_remote_id(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td, workspace.Member{
		Repo: "owner/repo", Branch: "feature/one", BaseRef: "main",
	})

	_, err := svc.Checkout(context.Background(), "login", "owner/repo", "feature/two", workspace.CheckoutOptions{
		Create: true,
	})
	assert.NoError(t, err)
	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo"), "feature/two")
}

func TestCheckout_updates_only_the_target_member(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td,
		workspace.Member{Repo: "owner/api", Branch: "feature/one", BaseRef: "main"},
		workspace.Member{Repo: "owner/web", Branch: "feature/one", BaseRef: "main"},
	)

	_, err := svc.Checkout(context.Background(), "login", "web", "feature/two", workspace.CheckoutOptions{
		Create: true,
	})
	assert.NoError(t, err)

	assertManifestMember(t, td, 0, "owner/api", "feature/one", "main")
	assertManifestMember(t, td, 1, "owner/web", "feature/two", "main")
}

func TestCheckout_leaves_an_empty_base_ref_empty(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, serviceConfigWithRoots)
	initGitRepo(t, filepath.Join(td.projects, "owner", "repo"))
	// Pre-created so materialize never needs to resolve a default branch.
	runGit(t, "-C", filepath.Join(td.projects, "owner", "repo"), "branch", "feature/one", "main")

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/one"},
	}}))
	assert.NoError(t, svc.Materialize(context.Background(), "login"))

	_, err = svc.Checkout(context.Background(), "login", "repo", "feature/two", workspace.CheckoutOptions{
		Create: true,
	})
	assert.NoError(t, err)

	assertManifestMember(t, td, 0, "owner/repo", "feature/two", "")
}

func TestCheckout_heals_manifest_drift_without_switching(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td, workspace.Member{
		Repo: "owner/repo", Branch: "feature/one", BaseRef: "main",
	})

	worktree := filepath.Join(td.workspaces, "login", "repo")
	runGit(t, "-C", worktree, "switch", "-c", "feature/manual")

	result, err := svc.Checkout(context.Background(), "login", "repo", "feature/manual", workspace.CheckoutOptions{})
	assert.NoError(t, err)

	if result.Previous != "feature/manual" || result.Created {
		t.Fatalf("unexpected result %+v", result)
	}
	assertManifestMember(t, td, 0, "owner/repo", "feature/manual", "main")
}

func TestCheckout_refuses_unknown_manifest_version(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	svc := setupMaterialized(t, td, workspace.Member{
		Repo: "owner/repo", Branch: "feature/one", BaseRef: "main",
	})

	manifestPath := filepath.Join(td.workspaces, "login", ".z", "instance.yaml")
	writeFile(t, manifestPath,
		"version: 2\nmembers:\n  - repo: owner/repo\n    branch: feature/one\n    base_ref: main\n")

	_, err := svc.Checkout(context.Background(), "login", "repo", "feature/two", workspace.CheckoutOptions{Create: true})
	assertErrorContains(t, err, "manifest version 2")
	assertGitBranch(t, filepath.Join(td.workspaces, "login", "repo"), "feature/one")
}

func TestCheckout_tracks_remote_branch_without_create(t *testing.T) {
	requireGit(t)
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, serviceConfigWithRoots)
	cloneCanonicalFromBareRemote(t, td, "owner", "repo")

	canonical := filepath.Join(td.projects, "owner", "repo")
	runGit(t, "-C", canonical, "push", "origin", "main:feature/remote")
	runGit(t, "-C", canonical, "fetch", "origin")

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: []workspace.Member{
		{Repo: "owner/repo", Branch: "feature/one", BaseRef: "main"},
	}}))
	assert.NoError(t, svc.Materialize(context.Background(), "login"))

	result, err := svc.Checkout(context.Background(), "login", "repo", "feature/remote", workspace.CheckoutOptions{})
	assert.NoError(t, err)

	if !result.Created {
		t.Fatalf("expected a local branch to be created by DWIM, got %+v", result)
	}
	worktree := filepath.Join(td.workspaces, "login", "repo")
	assertGitBranch(t, worktree, "feature/remote")
	upstream := runGit(t, "-C", worktree, "rev-parse", "--abbrev-ref", "feature/remote@{upstream}")
	if upstream != "origin/feature/remote" {
		t.Fatalf("upstream = %q, want origin/feature/remote", upstream)
	}
	assertManifestMember(t, td, 0, "owner/repo", "feature/remote", "main")
}

func TestCheckout_rejects_invalid_branch_before_touching_the_workspace(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, serviceConfigWithRoots)

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	_, err = svc.Checkout(context.Background(), "login", "repo", "--force", workspace.CheckoutOptions{})
	assertErrorContains(t, err, "invalid branch")
}

const serviceConfigWithRoots = `
projects:
  root: $PROJECTSDIR
workspaces:
  root: $WORKSPACESDIR
`

// setupMaterialized creates a canonical clone per member and materializes an
// instance named "login".
func setupMaterialized(t *testing.T, td serviceTestDir, members ...workspace.Member) *workspace.Service {
	t.Helper()

	cfg := setupServiceConfig(t, td, serviceConfigWithRoots)
	for _, m := range members {
		initGitRepo(t, filepath.Join(td.projects, m.Repo))
	}

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	assert.NoError(t, svc.Create(context.Background(), "login", workspace.CreateOptions{Members: members}))
	assert.NoError(t, svc.Materialize(context.Background(), "login"))

	return svc
}

func commitFile(t *testing.T, worktree, name string) {
	t.Helper()
	writeFile(t, filepath.Join(worktree, name), "content\n")
	runGit(t, "-C", worktree, "add", name)
	runGit(t, "-C", worktree, "commit", "-m", "add "+name)
}

// manifestFile is declared independently of the domain type so a schema change
// breaks these assertions instead of silently following along.
type manifestFile struct {
	Version int `yaml:"version"`
	Members []struct {
		Repo    string `yaml:"repo"`
		Branch  string `yaml:"branch"`
		BaseRef string `yaml:"base_ref"`
	} `yaml:"members"`
}

func assertManifestMember(t *testing.T, td serviceTestDir, idx int, repo, branch, baseRef string) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(td.workspaces, "login", ".z", "instance.yaml"))
	assert.NoError(t, err)

	var mf manifestFile
	if err := yaml.Unmarshal(data, &mf); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if idx >= len(mf.Members) {
		t.Fatalf("manifest has %d members, want index %d:\n%s", len(mf.Members), idx, data)
	}

	got := mf.Members[idx]
	if got.Repo != repo || got.Branch != branch || got.BaseRef != baseRef {
		t.Fatalf("member %d = %+v, want {%s %s %s}", idx, got, repo, branch, baseRef)
	}
}
