package sync_test

import (
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

func member() workspace.Member {
	return workspace.Member{Repo: "owner/repo", Branch: "feature/login"}
}

func TestSync_copies_definition_files_and_records_them(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	h.SeedDefinitionFile("feature", ".claude/settings.json", "{}\n")
	h.SeedDefinitionExecFile("feature", "bin/setup", "#!/bin/sh\n")

	err := h.run("login")

	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	h.FileContains(filepath.Join("login", "CLAUDE.md"), "v1\n")
	h.FileContains(filepath.Join("login", ".claude", "settings.json"), "{}\n")
	h.FileIsExecutable(filepath.Join("login", "bin", "setup"))

	h.SyncState("login").
		HasVersion(1).
		FileCount(3).
		Records("CLAUDE.md").
		Records(".claude/settings.json").
		RecordsExecutable("bin/setup")
}

func TestSync_definition_metadata_is_not_copied(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")

	if err := h.run("login"); err != nil {
		t.Fatalf("sync: %v", err)
	}

	h.PathMissing(filepath.Join("login", ".z", "definition.yaml"))
	h.SyncState("login").DoesNotRecord(".z/definition.yaml")
}

func TestSync_second_run_reports_up_to_date(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")

	if err := h.run("login"); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if err := h.run("login"); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	h.OutputContains("already up to date")
}

func TestSync_updates_an_untouched_file(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	if err := h.run("login"); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	before := h.SyncState("login").HashOf("CLAUDE.md")

	h.SeedDefinitionFile("feature", "CLAUDE.md", "v2\n")
	if err := h.run("login"); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	h.FileContains(filepath.Join("login", "CLAUDE.md"), "v2\n")
	if after := h.SyncState("login").HashOf("CLAUDE.md"); after == before {
		t.Error("recorded hash did not advance after an applied update")
	}
}

func TestSync_locally_modified_file_is_skipped_with_a_warning(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	if err := h.run("login"); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	before := h.SyncState("login").HashOf("CLAUDE.md")

	h.SeedFile(filepath.Join("login", "CLAUDE.md"), "mine\n")
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v2\n")
	err := h.run("login")

	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	h.FileContains(filepath.Join("login", "CLAUDE.md"), "mine\n")
	h.ErrOutputContains("modified locally")
	h.ErrOutputContains("CLAUDE.md")
	h.OutputContains("1 skipped")

	if after := h.SyncState("login").HashOf("CLAUDE.md"); after != before {
		t.Error("a skipped conflict advanced the recorded hash")
	}
}

func TestSync_error_on_conflict_exits_non_zero(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	if err := h.run("login"); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	h.SeedFile(filepath.Join("login", "CLAUDE.md"), "mine\n")
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v2\n")

	err := h.run("login", "--error-on-conflict")

	wstest.AssertErrorContains(t, err, "skipped as locally modified")
}

func TestSync_deleting_a_conflicted_file_takes_the_definition(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	if err := h.run("login"); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	h.SeedFile(filepath.Join("login", "CLAUDE.md"), "mine\n")
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v2\n")
	if err := h.run("login"); err != nil {
		t.Fatalf("conflicting sync: %v", err)
	}

	h.RemoveFile(filepath.Join("login", "CLAUDE.md"))
	if err := h.run("login"); err != nil {
		t.Fatalf("resolving sync: %v", err)
	}

	h.FileContains(filepath.Join("login", "CLAUDE.md"), "v2\n")
}

func TestSync_file_dropped_from_the_definition_is_removed_when_untouched(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "NOTES.md", "v1\n")
	if err := h.run("login"); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	h.RemoveDefinitionFile("feature", "NOTES.md")
	if err := h.run("login"); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	h.PathMissing(filepath.Join("login", "NOTES.md"))
	h.SyncState("login").DoesNotRecord("NOTES.md")
}

func TestSync_file_dropped_from_the_definition_is_released_when_modified(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "NOTES.md", "v1\n")
	if err := h.run("login"); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	h.SeedFile(filepath.Join("login", "NOTES.md"), "mine\n")
	h.RemoveDefinitionFile("feature", "NOTES.md")
	if err := h.run("login"); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	h.FileContains(filepath.Join("login", "NOTES.md"), "mine\n")
	h.SyncState("login").DoesNotRecord("NOTES.md")
}

func TestSync_dry_run_writes_nothing(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")

	err := h.run("login", "--dry-run")

	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	h.OutputContains("Would sync")
	h.OutputContains("CLAUDE.md")
	h.PathMissing(filepath.Join("login", "CLAUDE.md"))
	h.NoSyncState("login")
}

func TestSync_force_overwrites_a_modified_file(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	if err := h.run("login"); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	h.SeedFile(filepath.Join("login", "CLAUDE.md"), "mine\n")
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v2\n")
	if err := h.run("login", "--force"); err != nil {
		t.Fatalf("forced sync: %v", err)
	}

	h.FileContains(filepath.Join("login", "CLAUDE.md"), "v2\n")
}

func TestSync_force_never_removes_the_instance_manifest(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")

	if err := h.run("login", "--force"); err != nil {
		t.Fatalf("forced sync: %v", err)
	}

	h.FileExists(filepath.Join("login", ".z", "instance.yaml"))
}

func TestSync_force_never_removes_a_member_worktree(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	h.SeedFile(filepath.Join("login", "repo", "main.go"), "package main\n")

	if err := h.run("login", "--force"); err != nil {
		t.Fatalf("forced sync: %v", err)
	}

	h.FileContains(filepath.Join("login", "repo", "main.go"), "package main\n")
}

func TestSync_definition_ignore_excludes_content(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinitionManifest("feature", `version: 1
branch_pattern: "{instance}"
members:
  - repo: owner/repo
sync:
  ignore:
    paths:
      - scratch/
`)
	h.SeedManifest("login", manifestFor("feature", []workspace.Member{member()}))
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	h.SeedDefinitionFile("feature", "scratch/notes.md", "ignored\n")

	if err := h.run("login"); err != nil {
		t.Fatalf("sync: %v", err)
	}

	h.FileExists(filepath.Join("login", "CLAUDE.md"))
	h.PathMissing(filepath.Join("login", "scratch"))
}

func TestSync_instance_without_a_definition_is_refused(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", member())

	err := h.run("login")

	wstest.AssertErrorContains(t, err, "was not created from a definition")
}

func TestSync_missing_definition_is_refused_and_changes_nothing(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	if err := h.run("login"); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	h.SeedManifest("login", manifestFor("gone", []workspace.Member{member()}))
	err := h.run("login")

	wstest.AssertErrorContains(t, err, "not found")
	// The instance keeps its configuration: an unresolvable definition must
	// never read as an empty one.
	h.FileContains(filepath.Join("login", "CLAUDE.md"), "v1\n")
}

func TestSync_broken_definition_is_refused(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinitionManifest("feature", "version: 1\nbranch_pattern: \"{nope}\"\nmembers:\n")
	h.SeedManifest("login", manifestFor("feature", []workspace.Member{member()}))

	err := h.run("login")

	wstest.AssertErrorContains(t, err, "not usable")
}

func TestSync_missing_workspace_is_refused(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("login")

	wstest.AssertErrorContains(t, err, "has not been created")
}

func TestSync_name_defaults_to_the_workspace_containing_the_cwd(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("attempt-a", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	h.InDir("attempt-a")

	if err := h.run(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	h.FileContains(filepath.Join("attempt-a", "CLAUDE.md"), "v1\n")
}

func TestSync_symlink_in_the_definition_is_reported_and_skipped(t *testing.T) {
	h := newCommandTest(t)
	h.seedInstanceFromDefinition("login", member())
	h.SeedDefinitionFile("feature", "CLAUDE.md", "v1\n")
	h.SeedDefinitionSymlink("feature", "link.md", "CLAUDE.md")

	if err := h.run("login"); err != nil {
		t.Fatalf("sync: %v", err)
	}

	h.ErrOutputContains("symbolic link")
	h.FileExists(filepath.Join("login", "CLAUDE.md"))
	h.PathMissing(filepath.Join("login", "link.md"))
}
