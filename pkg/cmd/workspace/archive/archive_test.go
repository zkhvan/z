package archive_test

import (
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestArchive_missing_instance(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("missing")

	wstest.AssertErrorContains(t, err, "has not been created")
}

func TestArchive_omitted_name_outside_the_workspaces_root(t *testing.T) {
	h := newCommandTest(t)

	err := h.run()

	wstest.AssertErrorContains(t, err, "is not inside a workspace under")
}

func TestArchive_omitted_name_at_the_workspaces_root(t *testing.T) {
	h := newCommandTest(t)
	h.InDir()

	err := h.run()

	wstest.AssertErrorContains(t, err, "is not inside a workspace under")
}

func TestArchive_name_from_the_instance_directory(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedFile(filepath.Join("login", "repo", "work.txt"), "content\n")
	h.InDir("login")

	err := h.run("--force")
	assert.NoError(t, err)

	h.OutputContains(`Archived workspace "login"`)
	h.FileExists(filepath.Join("login", ".z", "instance.yaml"))
}

func TestArchive_name_from_a_member_worktree(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedFile(filepath.Join("login", "repo", "work.txt"), "content\n")
	h.InDir("login", "repo")

	err := h.run("--force")
	assert.NoError(t, err)

	h.OutputContains(`Archived workspace "login"`)
}

func TestArchive_explicit_name_overrides_the_current_directory(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedInstance("other", workspace.Member{Repo: "owner/other", Branch: "feature/other"})
	h.SeedFile(filepath.Join("other", "other", "work.txt"), "content\n")
	h.SeedFile(filepath.Join("login", "repo", "work.txt"), "content\n")
	h.InDir("other", "other")

	err := h.run("login", "--force")
	assert.NoError(t, err)

	h.OutputContains(`Archived workspace "login"`)
	h.PathMissing(filepath.Join("login", "repo"))
	h.FileExists(filepath.Join("other", "other", "work.txt"))
}

// The orphaned-member guard fires before any git call, so it exercises the
// --force wiring end to end without a git fixture.
func TestArchive_orphaned_member_is_refused(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedFile(filepath.Join("login", "repo", "work.txt"), "content\n")

	err := h.run("login")

	wstest.AssertErrorContains(t, err, "canonical clone is missing")
	h.FileExists(filepath.Join("login", "repo", "work.txt"))
}

func TestArchive_force_removes_orphaned_member(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedFile(filepath.Join("login", "repo", "work.txt"), "content\n")

	err := h.run("login", "--force")
	assert.NoError(t, err)

	h.PathMissing(filepath.Join("login", "repo"))
	h.FileExists(filepath.Join("login", ".z", "instance.yaml"))
	h.OutputContains(`Archived workspace "login"`)
}

func TestArchive_runs_post_archive_hook(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstanceFromDefinition("login", "feature", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedDefinitionHook("feature", workspace.HookPostArchive,
		"#!/bin/sh\nprintf '%s' \"$Z_HOOK_PHASE\" > archived.out\n")

	err := h.run("login", "--force")
	assert.NoError(t, err)

	h.FileContains(filepath.Join("login", "archived.out"), "post-archive")
	h.OutputContains(`Archived workspace "login"`)
}

func TestArchive_pre_archive_hook_failure_aborts(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstanceFromDefinition("login", "feature", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedDefinitionHook("feature", workspace.HookPreArchive, "#!/bin/sh\nexit 1\n")

	err := h.run("login", "--force")

	wstest.AssertErrorContains(t, err, "pre-archive hook failed")
	h.FileExists(filepath.Join("login", ".z", "instance.yaml"))
}
