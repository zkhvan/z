package delete_test

import (
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
)

func TestDelete_missing_instance(t *testing.T) {
	cmd, harness := newCommandTest(t)
	harness.config(withWorkspaceRoot())

	err := cmd.run(cmd.withName("missing"))

	assertErrorContains(t, err, "has not been created")
}

func TestDelete_missing_name_arg(t *testing.T) {
	cmd, harness := newCommandTest(t)
	harness.config(withWorkspaceRoot())

	err := cmd.run()

	assertErrorContains(t, err, "accepts 1 arg(s)")
}

// The unmanaged-files guard fires before any git call, so it exercises the
// --force wiring end to end without a git fixture.
func TestDelete_unmanaged_files_are_refused(t *testing.T) {
	cmd, harness := newCommandTest(t)
	harness.config(withWorkspaceRoot())
	harness.seedInstance("login", "owner/repo", "feature/login")
	harness.seedFile(filepath.Join("login", "notes.md"))

	err := cmd.run(cmd.withName("login"))

	assertErrorContains(t, err, "files not managed by z")
	assertErrorContains(t, err, "notes.md")
	harness.fileExists(filepath.Join("login", ".z", "instance.yaml"))
	harness.fileExists(filepath.Join("login", "notes.md"))
}

func TestDelete_force_removes_instance(t *testing.T) {
	cmd, harness := newCommandTest(t)
	harness.config(withWorkspaceRoot())
	harness.seedInstance("login", "owner/repo", "feature/login")
	harness.seedFile(filepath.Join("login", "notes.md"))

	err := cmd.run(cmd.withName("login"), cmd.withForce())
	assert.NoError(t, err)

	harness.pathMissing("login")
	cmd.outputContains(`Deleted workspace "login"`)
}

func TestDelete_directory_without_manifest_is_refused_even_when_forced(t *testing.T) {
	cmd, harness := newCommandTest(t)
	harness.config(withWorkspaceRoot())
	harness.seedFile(filepath.Join("not-a-workspace", "important.txt"))

	err := cmd.run(cmd.withName("not-a-workspace"), cmd.withForce())

	assertErrorContains(t, err, "has not been created")
	harness.fileExists(filepath.Join("not-a-workspace", "important.txt"))
}
