package checkout_test

import (
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestCheckout_missing_args(t *testing.T) {
	h := newCommandTest(t)

	err := h.run()

	wstest.AssertErrorContains(t, err, "accepts between 1 and 3 arg(s)")
}

func TestCheckout_name_from_the_instance_directory(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/one"})
	h.InDir("login")

	err := h.run("repo", "feature/two")

	wstest.AssertErrorContains(t, err, "run `z workspace materialize login` first")
}

func TestCheckout_name_and_repo_from_a_member_worktree(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/one"})
	h.SeedDir(filepath.Join("login", "repo"))
	h.InDir("login", "repo")

	err := h.run("feature/two")

	wstest.AssertErrorContains(t, err, "owner/repo is not materialized")
}

func TestCheckout_omitted_repo_at_the_instance_directory(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/one"})
	h.InDir("login")

	err := h.run("feature/two")

	wstest.AssertErrorContains(t, err, `is not inside a member worktree of workspace "login"`)
}

func TestCheckout_explicit_args_override_the_current_directory(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/api", Branch: "feature/one"})
	h.SeedInstance("other", workspace.Member{Repo: "owner/web", Branch: "feature/one"})
	h.SeedDir(filepath.Join("other", "web"))
	h.InDir("other", "web")

	err := h.run("login", "api", "feature/two")

	wstest.AssertErrorContains(t, err, "run `z workspace materialize login` first")
}

func TestCheckout_missing_instance(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("missing", "repo", "feature/two")

	wstest.AssertErrorContains(t, err, "has not been created")
}

func TestCheckout_unknown_member_lists_members(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login",
		workspace.Member{Repo: "owner/api", Branch: "feature/one"},
		workspace.Member{Repo: "owner/web", Branch: "feature/one"},
	)

	err := h.run("login", "docs", "feature/two")

	wstest.AssertErrorContains(t, err, `no member "docs"`)
	wstest.AssertErrorContains(t, err, "api (owner/api), web (owner/web)")
}

func TestCheckout_base_without_create(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/one"})

	err := h.run("login", "repo", "feature/two", "--base", "main")

	wstest.AssertErrorContains(t, err, "--base requires --create")
}

// The missing canonical clone stops checkout before it shells out, so the
// --create/--base wiring is provable without a git fixture: --base no longer
// trips the guard above, and the run reaches the materialization check.
func TestCheckout_create_with_base_reaches_the_service(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/one"})

	err := h.run("login", "repo", "feature/two", "--create", "--base", "main")

	wstest.AssertErrorContains(t, err, "is not materialized")
}

func TestCheckout_unmaterialized_member(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/one"})

	err := h.run("login", "repo", "feature/two")

	wstest.AssertErrorContains(t, err, "run `z workspace materialize login` first")
}

func TestCheckout_invalid_branch(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/one"})

	err := h.run("login", "repo", "HEAD")

	wstest.AssertErrorContains(t, err, "invalid branch")
}
