package create_test

import (
	"errors"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestCreate_single_member(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("login-feature", "--member", "org/repo@feature/login")
	assert.NoError(t, err)

	h.Workspace("login-feature").
		HasVersion(1).
		MemberCount(1).
		HasMember("org/repo", "feature/login", "")

	h.OutputContains(`Created workspace "login-feature"`)
}

func TestCreate_multiple_members_with_base_ref(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("auth",
		"--member", "org/repo1@feature/auth",
		"--member", "org/repo2@feature/auth:main",
	)
	assert.NoError(t, err)

	h.Workspace("auth").
		MemberCount(2).
		HasMember("org/repo1", "feature/auth", "").
		HasMember("org/repo2", "feature/auth", "main")
}

func TestCreate_missing_name_arg(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("--member", "org/repo@main")

	wstest.AssertErrorContains(t, err, "accepts 1 arg(s)")
}

func TestCreate_no_members_outside_a_terminal_fails_loudly(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("empty")

	wstest.AssertErrorContains(t, err, "pass --member, or run interactively")
	h.NoWorkspace("empty")
}

func TestCreate_malformed_member(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("bad", "--member", "org/repo") // missing @branch

	wstest.AssertErrorContains(t, err, "invalid member")
	h.NoWorkspace("bad")
}

func TestCreate_invalid_name(t *testing.T) {
	h := newCommandTest(t)

	err := h.run(".hidden", "--member", "org/repo@main")

	if !errors.Is(err, workspace.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
	h.NoWorkspace(".hidden")
}

func TestCreate_duplicate_member_base_name(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("conflict", "--member", "org1/repo@main", "--member", "org2/repo@main")

	wstest.AssertErrorContains(t, err, "worktree directory name")
	h.NoWorkspace("conflict")
}

func TestCreate_existing_empty_dir_is_rejected(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDir("dupe")

	err := h.run("dupe", "--member", "org/repo@main")

	wstest.AssertErrorContains(t, err, "already exists")

	// Nothing written into the pre-existing directory.
	h.NoWorkspace("dupe")
}

func TestCreate_existing_instance_is_rejected(t *testing.T) {
	h := newCommandTest(t)
	h.SeedManifest("dupe", "")

	err := h.run("dupe", "--member", "org/repo@main")

	wstest.AssertErrorContains(t, err, "already exists")

	// The seeded manifest must be untouched (still empty, no members).
	h.Workspace("dupe").MemberCount(0)
}
