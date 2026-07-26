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

func TestArchive_missing_name_arg(t *testing.T) {
	h := newCommandTest(t)

	err := h.run()

	wstest.AssertErrorContains(t, err, "accepts 1 arg(s)")
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
