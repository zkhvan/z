package wssync_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/workspace/wssync"
)

// syncOnce runs the whole pipeline the way Service.Sync will, returning the
// plan so tests can assert on both the plan and the resulting disk state.
func syncOnce(t *testing.T, definitionDir, instanceDir string, mode wssync.Mode) wssync.Result {
	t.Helper()

	ancestor, err := wssync.LoadState(instanceDir)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	// `.z/` is z's namespace on both sides, never user content, never synced.
	definition := scan(t, definitionDir, wssync.Ignores{Structural: []string{".z"}})
	instance := scan(t, instanceDir, wssync.Ignores{Structural: []string{".z"}})

	result := wssync.Reconcile(ancestor, definition, instance, mode)

	applied, err := wssync.Transition(instanceDir, definitionDir, result.Changes)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	next := wssync.Apply(ancestor, append(result.AncestorChanges, applied...))
	if err := wssync.SaveState(instanceDir, next); err != nil {
		t.Fatalf("save state: %v", err)
	}
	return result
}

func readFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func TestTransition_first_sync_copies_the_definition_tree(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "CLAUDE.md", "v1\n", 0o644)
	seed(t, def, ".claude/settings.json", "{}\n", 0o644)
	seed(t, def, "hooks/post-create", "#!/bin/sh\n", 0o755)

	syncOnce(t, def, inst, wssync.ModeSafe)

	if got := readFile(t, inst, "CLAUDE.md"); got != "v1\n" {
		t.Errorf("CLAUDE.md = %q", got)
	}
	if got := readFile(t, inst, ".claude/settings.json"); got != "{}\n" {
		t.Errorf("settings.json = %q", got)
	}

	info, err := os.Stat(filepath.Join(inst, "hooks", "post-create"))
	if err != nil {
		t.Fatalf("stat hook: %v", err)
	}
	if info.Mode()&0o100 == 0 {
		t.Errorf("hook mode = %v, want executable", info.Mode())
	}
}

func TestTransition_definition_metadata_is_never_copied(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, ".z/definition.yaml", "version: 1\n", 0o600)
	seed(t, def, "CLAUDE.md", "v1\n", 0o644)

	syncOnce(t, def, inst, wssync.ModeSafe)

	if _, err := os.Stat(filepath.Join(inst, ".z", "definition.yaml")); !os.IsNotExist(err) {
		t.Error("the definition manifest was copied into the instance")
	}
	if got := readFile(t, inst, "CLAUDE.md"); got != "v1\n" {
		t.Errorf("CLAUDE.md = %q", got)
	}
}

func TestTransition_second_sync_is_a_noop(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "CLAUDE.md", "v1\n", 0o644)

	syncOnce(t, def, inst, wssync.ModeSafe)
	result := syncOnce(t, def, inst, wssync.ModeSafe)

	if !result.Empty() {
		t.Errorf("second sync planned %+v", result)
	}
}

func TestTransition_updates_an_untouched_file_and_skips_a_modified_one(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "CLAUDE.md", "v1\n", 0o644)
	seed(t, def, "NOTES.md", "v1\n", 0o644)
	syncOnce(t, def, inst, wssync.ModeSafe)

	seed(t, def, "CLAUDE.md", "v2\n", 0o644)
	seed(t, def, "NOTES.md", "v2\n", 0o644)
	seed(t, inst, "NOTES.md", "mine\n", 0o644)

	result := syncOnce(t, def, inst, wssync.ModeSafe)

	if got := readFile(t, inst, "CLAUDE.md"); got != "v2\n" {
		t.Errorf("CLAUDE.md = %q, want v2", got)
	}
	if got := readFile(t, inst, "NOTES.md"); got != "mine\n" {
		t.Errorf("NOTES.md = %q, want the local edit preserved", got)
	}
	if len(result.Conflicts) != 1 || result.Conflicts[0].Path != "NOTES.md" {
		t.Errorf("conflicts = %+v, want one for NOTES.md", result.Conflicts)
	}
}

func TestTransition_conflict_is_reported_again_on_the_next_sync(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "NOTES.md", "v1\n", 0o644)
	syncOnce(t, def, inst, wssync.ModeSafe)

	seed(t, def, "NOTES.md", "v2\n", 0o644)
	seed(t, inst, "NOTES.md", "mine\n", 0o644)

	syncOnce(t, def, inst, wssync.ModeSafe)
	result := syncOnce(t, def, inst, wssync.ModeSafe)

	if len(result.Conflicts) != 1 {
		t.Fatalf("conflicts = %+v, want the conflict to persist", result.Conflicts)
	}
}

func TestTransition_deleting_an_instance_file_restores_it(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "CLAUDE.md", "v1\n", 0o644)
	syncOnce(t, def, inst, wssync.ModeSafe)

	if err := os.Remove(filepath.Join(inst, "CLAUDE.md")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	syncOnce(t, def, inst, wssync.ModeSafe)

	if got := readFile(t, inst, "CLAUDE.md"); got != "v1\n" {
		t.Errorf("CLAUDE.md = %q, want it restored", got)
	}
}

func TestTransition_deleting_a_conflicted_file_accepts_the_definition(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "NOTES.md", "v1\n", 0o644)
	syncOnce(t, def, inst, wssync.ModeSafe)

	seed(t, def, "NOTES.md", "v2\n", 0o644)
	seed(t, inst, "NOTES.md", "mine\n", 0o644)
	syncOnce(t, def, inst, wssync.ModeSafe)

	// The documented resolution: delete the side that should lose.
	if err := os.Remove(filepath.Join(inst, "NOTES.md")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	result := syncOnce(t, def, inst, wssync.ModeSafe)

	if len(result.Conflicts) != 0 {
		t.Errorf("conflicts = %+v, want none after deleting the losing side", result.Conflicts)
	}
	if got := readFile(t, inst, "NOTES.md"); got != "v2\n" {
		t.Errorf("NOTES.md = %q, want v2", got)
	}
}

func TestTransition_file_dropped_from_the_definition(t *testing.T) {
	t.Run("unmodified is removed", func(t *testing.T) {
		def, inst := t.TempDir(), t.TempDir()
		seed(t, def, "NOTES.md", "v1\n", 0o644)
		syncOnce(t, def, inst, wssync.ModeSafe)

		if err := os.Remove(filepath.Join(def, "NOTES.md")); err != nil {
			t.Fatalf("remove: %v", err)
		}
		syncOnce(t, def, inst, wssync.ModeSafe)

		if _, err := os.Stat(filepath.Join(inst, "NOTES.md")); !os.IsNotExist(err) {
			t.Error("unmodified file dropped by the definition was kept")
		}
	})

	t.Run("modified is untracked and left alone", func(t *testing.T) {
		def, inst := t.TempDir(), t.TempDir()
		seed(t, def, "NOTES.md", "v1\n", 0o644)
		syncOnce(t, def, inst, wssync.ModeSafe)

		seed(t, inst, "NOTES.md", "mine\n", 0o644)
		if err := os.Remove(filepath.Join(def, "NOTES.md")); err != nil {
			t.Fatalf("remove: %v", err)
		}
		syncOnce(t, def, inst, wssync.ModeSafe)

		if got := readFile(t, inst, "NOTES.md"); got != "mine\n" {
			t.Errorf("NOTES.md = %q, want the local edit kept", got)
		}

		// Untracked means untracked: no conflict, no plan, on every later run.
		result := syncOnce(t, def, inst, wssync.ModeSafe)
		if !result.Empty() || len(result.Conflicts) != 0 {
			t.Errorf("released file still generates work: %+v", result)
		}
	})
}

func TestTransition_replica_mode_overwrites_and_prunes(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "CLAUDE.md", "v1\n", 0o644)
	syncOnce(t, def, inst, wssync.ModeSafe)

	seed(t, def, "CLAUDE.md", "v2\n", 0o644)
	seed(t, inst, "CLAUDE.md", "mine\n", 0o644)
	seed(t, inst, "junk.md", "junk\n", 0o644)

	syncOnce(t, def, inst, wssync.ModeReplica)

	if got := readFile(t, inst, "CLAUDE.md"); got != "v2\n" {
		t.Errorf("CLAUDE.md = %q, want the definition to win", got)
	}
	if _, err := os.Stat(filepath.Join(inst, "junk.md")); !os.IsNotExist(err) {
		t.Error("replica mode kept extra content")
	}
}

func TestTransition_replica_mode_never_touches_ignored_content(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "CLAUDE.md", "v1\n", 0o644)
	seed(t, inst, "api/.git", "gitdir: elsewhere\n", 0o644)
	seed(t, inst, "api/main.go", "package main\n", 0o644)
	seed(t, inst, ".z/instance.yaml", "version: 1\n", 0o600)

	syncOnce(t, def, inst, wssync.ModeReplica)

	if got := readFile(t, inst, "api/main.go"); got != "package main\n" {
		t.Error("replica mode removed worktree content")
	}
	if got := readFile(t, inst, ".z/instance.yaml"); got != "version: 1\n" {
		t.Error("replica mode removed the instance manifest")
	}
}

func TestTransition_swaps_a_file_for_a_directory(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "hooks", "old\n", 0o644)
	syncOnce(t, def, inst, wssync.ModeSafe)

	if err := os.Remove(filepath.Join(def, "hooks")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	seed(t, def, "hooks/post-create", "#!/bin/sh\n", 0o755)
	syncOnce(t, def, inst, wssync.ModeSafe)

	if got := readFile(t, inst, "hooks/post-create"); got != "#!/bin/sh\n" {
		t.Errorf("hooks/post-create = %q", got)
	}
}

func TestTransition_interrupted_sync_recovers_without_conflicts(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "CLAUDE.md", "v1\n", 0o644)
	syncOnce(t, def, inst, wssync.ModeSafe)

	// Simulate a crash after the write but before the state was persisted.
	seed(t, def, "CLAUDE.md", "v2\n", 0o644)
	seed(t, inst, "CLAUDE.md", "v2\n", 0o644)

	result := syncOnce(t, def, inst, wssync.ModeSafe)

	if len(result.Conflicts) != 0 {
		t.Errorf("conflicts = %+v, want convergence to absorb the unrecorded write", result.Conflicts)
	}
	if next := syncOnce(t, def, inst, wssync.ModeSafe); !next.Empty() {
		t.Errorf("state was not repaired: %+v", next)
	}
}

func TestState_round_trips_paths_hashes_and_executability(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, def, "CLAUDE.md", "v1\n", 0o644)
	seed(t, def, "hooks/post-create", "#!/bin/sh\n", 0o755)
	syncOnce(t, def, inst, wssync.ModeSafe)

	loaded, err := wssync.LoadState(inst)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	got := wssync.StatePaths(loaded)
	want := []string{"CLAUDE.md", "hooks/post-create"}
	if len(got) != len(want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("paths = %v, want %v", got, want)
		}
	}
	if !loaded.Contents["hooks"].Contents["post-create"].Exec {
		t.Error("executability did not survive the round trip")
	}
}

func TestState_newer_version_is_refused(t *testing.T) {
	inst := t.TempDir()
	seed(t, inst, ".z/sync-state.yaml", "version: 99\nfiles: []\n", 0o600)

	if _, err := wssync.LoadState(inst); err == nil {
		t.Error("a newer sync state was accepted")
	}
}

func TestState_absent_file_is_not_an_error(t *testing.T) {
	root, err := wssync.LoadState(t.TempDir())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if root != nil {
		t.Errorf("root = %+v, want nil", root)
	}
}

// A definition that ships nothing must remove its files and leave the instance
// directory, and z's own metadata, standing.
func TestTransition_empty_definition_does_not_remove_the_instance(t *testing.T) {
	def, inst := t.TempDir(), t.TempDir()
	seed(t, inst, ".z/instance.yaml", "version: 1\n", 0o600)
	seed(t, def, "CLAUDE.md", "v1\n", 0o644)
	syncOnce(t, def, inst, wssync.ModeSafe)

	if err := os.Remove(filepath.Join(def, "CLAUDE.md")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	syncEmpty(t, def, inst, wssync.ModeSafe)

	if _, err := os.Stat(inst); err != nil {
		t.Fatalf("instance directory was removed: %v", err)
	}
	if got := readFile(t, inst, ".z/instance.yaml"); got != "version: 1\n" {
		t.Error("instance manifest was removed")
	}
	if _, err := os.Stat(filepath.Join(inst, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Error("the dropped file was kept")
	}
}

// syncEmpty mirrors Service.sync's pruning, which is what keeps an empty
// definition from resolving to "delete the instance".
func syncEmpty(t *testing.T, definitionDir, instanceDir string, mode wssync.Mode) {
	t.Helper()

	ancestor, err := wssync.LoadState(instanceDir)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	definition := wssync.PruneEmptyDirs(scan(t, definitionDir, wssync.Ignores{Structural: []string{".z"}}))
	instance := scan(t, instanceDir, wssync.Ignores{Structural: []string{".z"}})

	result := wssync.Reconcile(ancestor, definition, instance, mode)
	applied, err := wssync.Transition(instanceDir, definitionDir, result.Changes)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	next := wssync.Apply(ancestor, append(result.AncestorChanges, applied...))
	if err := wssync.SaveState(instanceDir, next); err != nil {
		t.Fatalf("save state: %v", err)
	}
}
