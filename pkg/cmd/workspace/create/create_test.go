package create_test

import (
	"errors"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestCreate_single_member(t *testing.T) {
	cmd, harness, cleanup := newCommandTest(t)
	defer cleanup()

	harness.config(withWorkspaceRoot())

	err := cmd.run(
		cmd.withName("login-feature"),
		cmd.withMember("org/repo@feature/login"),
	)
	assert.NoError(t, err)

	harness.workspace("login-feature").
		hasVersion(1).
		memberCount(1).
		hasMember("org/repo", "feature/login", "")

	cmd.outputContains(`Created workspace "login-feature"`)
}

func TestCreate_multiple_members_with_base_ref(t *testing.T) {
	cmd, harness, cleanup := newCommandTest(t)
	defer cleanup()

	harness.config(withWorkspaceRoot())

	err := cmd.run(
		cmd.withName("auth"),
		cmd.withMember("org/repo1@feature/auth"),
		cmd.withMember("org/repo2@feature/auth:main"),
	)
	assert.NoError(t, err)

	harness.workspace("auth").
		memberCount(2).
		hasMember("org/repo1", "feature/auth", "").
		hasMember("org/repo2", "feature/auth", "main")
}

func TestCreate_missing_name_arg(t *testing.T) {
	cmd, harness, cleanup := newCommandTest(t)
	defer cleanup()

	harness.config(withWorkspaceRoot())

	err := cmd.run(
		cmd.withMember("org/repo@main"),
	)
	assertErrorContains(t, err, "accepts 1 arg(s)")
}

func TestCreate_missing_member_flag(t *testing.T) {
	cmd, harness, cleanup := newCommandTest(t)
	defer cleanup()

	harness.config(withWorkspaceRoot())

	err := cmd.run(
		cmd.withName("empty"),
	)
	assertErrorContains(t, err, `required flag(s) "member" not set`)
	harness.noWorkspace("empty")
}

func TestCreate_malformed_member(t *testing.T) {
	cmd, harness, cleanup := newCommandTest(t)
	defer cleanup()

	harness.config(withWorkspaceRoot())

	err := cmd.run(
		cmd.withName("bad"),
		cmd.withMember("org/repo"), // missing @branch
	)
	assertErrorContains(t, err, "invalid member")
	harness.noWorkspace("bad")
}

func TestCreate_invalid_name(t *testing.T) {
	cmd, harness, cleanup := newCommandTest(t)
	defer cleanup()

	harness.config(withWorkspaceRoot())

	err := cmd.run(
		cmd.withName(".hidden"),
		cmd.withMember("org/repo@main"),
	)
	if !errors.Is(err, workspace.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
	harness.noWorkspace(".hidden")
}

func TestCreate_duplicate_member_base_name(t *testing.T) {
	cmd, harness, cleanup := newCommandTest(t)
	defer cleanup()

	harness.config(withWorkspaceRoot())

	err := cmd.run(
		cmd.withName("conflict"),
		cmd.withMember("org1/repo@main"),
		cmd.withMember("org2/repo@main"),
	)
	assertErrorContains(t, err, "worktree directory name")
	harness.noWorkspace("conflict")
}

func TestCreate_existing_empty_dir_is_rejected(t *testing.T) {
	cmd, harness, cleanup := newCommandTest(t)
	defer cleanup()

	harness.config(withWorkspaceRoot())
	harness.seedWorkspaceDir(wsDir.withName("dupe"))

	err := cmd.run(
		cmd.withName("dupe"),
		cmd.withMember("org/repo@main"),
	)
	assertErrorContains(t, err, "already exists")

	// Nothing written into the pre-existing directory.
	harness.noWorkspace("dupe")
}

func TestCreate_existing_instance_is_rejected(t *testing.T) {
	cmd, harness, cleanup := newCommandTest(t)
	defer cleanup()

	harness.config(withWorkspaceRoot())
	harness.seedWorkspaceDir(
		wsDir.withName("dupe"),
		wsDir.withEmptyInstanceManifest(),
	)

	err := cmd.run(
		cmd.withName("dupe"),
		cmd.withMember("org/repo@main"),
	)
	assertErrorContains(t, err, "already exists")

	// The seeded manifest must be untouched (still empty, no members).
	harness.workspace("dupe").memberCount(0)
}
