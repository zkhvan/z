package delete_test

import (
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestDelete_missing_instance(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("missing")

	wstest.AssertErrorContains(t, err, "has not been created")
}

func TestDelete_omitted_name_outside_the_workspaces_root(t *testing.T) {
	h := newCommandTest(t)

	err := h.run()

	wstest.AssertErrorContains(t, err, "is not inside a workspace under")
}

func TestDelete_name_from_the_instance_directory(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.InDir("login")

	err := h.run("--force")
	assert.NoError(t, err)

	h.PathMissing("login")
	h.OutputContains(`Deleted workspace "login"`)
}

// The unmanaged-files guard fires before any git call, so it exercises the
// --force wiring end to end without a git fixture.
func TestDelete_unmanaged_files_are_refused(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedFile(filepath.Join("login", "notes.md"), "content\n")

	err := h.run("login")

	wstest.AssertErrorContains(t, err, "files not managed by z")
	wstest.AssertErrorContains(t, err, "notes.md")
	h.FileExists(filepath.Join("login", ".z", "instance.yaml"))
	h.FileExists(filepath.Join("login", "notes.md"))
}

func TestDelete_force_removes_instance(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedFile(filepath.Join("login", "notes.md"), "content\n")

	err := h.run("login", "--force")
	assert.NoError(t, err)

	h.PathMissing("login")
	h.OutputContains(`Deleted workspace "login"`)
}

func TestDelete_directory_without_manifest_is_refused_even_when_forced(t *testing.T) {
	h := newCommandTest(t)
	h.SeedFile(filepath.Join("not-a-workspace", "important.txt"), "content\n")

	err := h.run("not-a-workspace", "--force")

	wstest.AssertErrorContains(t, err, "has not been created")
	h.FileExists(filepath.Join("not-a-workspace", "important.txt"))
}

func TestDelete_runs_post_delete_hook_from_the_workspaces_root(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstanceFromDefinition("login", "feature", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	// The instance directory is gone by post-delete, so the hook runs from the
	// workspaces root and records itself there, keyed on the instance name.
	h.SeedDefinitionHook("feature", workspace.HookPostDelete,
		"#!/bin/sh\nprintf '%s' \"$Z_HOOK_PHASE\" > \"$Z_INSTANCE_NAME.deleted\"\n")

	err := h.run("login", "--force")
	assert.NoError(t, err)

	h.PathMissing("login")
	h.FileContains("login.deleted", "post-delete")
	h.OutputContains(`Deleted workspace "login"`)
}

func TestDelete_pre_delete_hook_failure_aborts(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstanceFromDefinition("login", "feature", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedDefinitionHook("feature", workspace.HookPreDelete, "#!/bin/sh\nexit 1\n")

	err := h.run("login", "--force")

	wstest.AssertErrorContains(t, err, "pre-delete hook failed")
	h.FileExists(filepath.Join("login", ".z", "instance.yaml"))
}

func TestDelete_unresolvable_definition_warns_and_proceeds(t *testing.T) {
	h := newCommandTest(t)
	h.SeedManifest("login", "version: 1\ndefinition: ghost\nmembers:\n  - repo: owner/repo\n    branch: feature/login\n")

	err := h.run("login", "--force")
	assert.NoError(t, err)

	h.PathMissing("login")
	h.ErrOutputContains("skipping hooks")
}
