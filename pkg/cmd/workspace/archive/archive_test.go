package archive_test

import (
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
)

func TestArchive_missing_instance(t *testing.T) {
	cmd, harness := newCommandTest(t)
	harness.config(withWorkspaceRoot())

	err := cmd.run(cmd.withName("missing"))

	assertErrorContains(t, err, "has not been created")
}

func TestArchive_missing_name_arg(t *testing.T) {
	cmd, harness := newCommandTest(t)
	harness.config(withWorkspaceRoot())

	err := cmd.run()

	assertErrorContains(t, err, "accepts 1 arg(s)")
}

// The orphaned-member guard fires before any git call, so it exercises the
// --force wiring end to end without a git fixture.
func TestArchive_orphaned_member_is_refused(t *testing.T) {
	cmd, harness := newCommandTest(t)
	harness.config(withWorkspaceRoot())
	harness.seedInstance("login", "owner/repo", "feature/login")
	harness.seedMemberFile("login", "repo", "work.txt")

	err := cmd.run(cmd.withName("login"))

	assertErrorContains(t, err, "canonical clone is missing")
	harness.fileExists(filepath.Join("login", "repo", "work.txt"))
}

func TestArchive_force_removes_orphaned_member(t *testing.T) {
	cmd, harness := newCommandTest(t)
	harness.config(withWorkspaceRoot())
	harness.seedInstance("login", "owner/repo", "feature/login")
	harness.seedMemberFile("login", "repo", "work.txt")

	err := cmd.run(cmd.withName("login"), cmd.withForce())
	assert.NoError(t, err)

	harness.pathMissing(filepath.Join("login", "repo"))
	harness.fileExists(filepath.Join("login", ".z", "instance.yaml"))
	cmd.outputContains(`Archived workspace "login"`)
}
