package wssync_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/workspace/wssync"
)

func seed(t *testing.T, dir, rel, content string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("seed %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("seed %s: %v", rel, err)
	}
}

func scan(t *testing.T, dir string, ignore wssync.Matcher) *wssync.Entry {
	t.Helper()
	root, err := wssync.Scan(dir, ignore)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	return root
}

func TestScan_records_hash_and_executability(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "CLAUDE.md", "hello\n", 0o644)
	seed(t, dir, "hooks/post-create", "#!/bin/sh\n", 0o755)

	root := scan(t, dir, wssync.NoIgnores())

	claude := root.Contents["CLAUDE.md"]
	if !strings.HasPrefix(claude.Hash, wssync.HashPrefix) {
		t.Errorf("hash = %q, want %s prefix", claude.Hash, wssync.HashPrefix)
	}
	if claude.Exec {
		t.Error("CLAUDE.md scanned as executable")
	}
	if hook := root.Contents["hooks"].Contents["post-create"]; !hook.Exec {
		t.Error("post-create scanned as non-executable")
	}
}

func TestScan_identical_content_hashes_identically(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	seed(t, a, "f.md", "same\n", 0o644)
	seed(t, b, "f.md", "same\n", 0o644)

	if scan(t, a, wssync.NoIgnores()).Contents["f.md"].Hash != scan(t, b, wssync.NoIgnores()).Contents["f.md"].Hash {
		t.Error("identical content produced different hashes")
	}
}

func TestScan_symlink_is_problematic_and_does_not_fail_the_scan(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "CLAUDE.md", "hello\n", 0o644)
	if err := os.Symlink("CLAUDE.md", filepath.Join(dir, "link.md")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	root := scan(t, dir, wssync.NoIgnores())

	if got := root.Contents["link.md"].Kind; got != wssync.KindProblematic {
		t.Errorf("link kind = %v, want problematic", got)
	}
	if root.Contents["CLAUDE.md"].Kind != wssync.KindFile {
		t.Error("sibling of a symlink was not scanned")
	}

	problems := root.Problems("")
	if len(problems) != 1 || problems[0].Path != "link.md" {
		t.Errorf("problems = %+v, want one for link.md", problems)
	}
}

func TestScan_directory_holding_a_git_repo_is_untracked(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "api/.git", "gitdir: elsewhere\n", 0o644)
	seed(t, dir, "api/main.go", "package main\n", 0o644)

	root := scan(t, dir, wssync.NoIgnores())

	api := root.Contents["api"]
	if api.Kind != wssync.KindUntracked {
		t.Fatalf("api kind = %v, want untracked", api.Kind)
	}
	if len(api.Contents) != 0 {
		t.Error("untracked worktree was descended into")
	}
}

func TestScan_ignored_entries_are_untracked(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, ".z/instance.yaml", "version: 1\n", 0o600)
	seed(t, dir, "node_modules/pkg/index.js", "\n", 0o644)
	seed(t, dir, "nested/.DS_Store", "\n", 0o644)
	seed(t, dir, "CLAUDE.md", "hello\n", 0o644)

	ignore := wssync.Ignores{
		Structural: []string{".z"},
		Patterns:   []string{"node_modules/", ".DS_Store"},
	}
	root := scan(t, dir, ignore)

	for _, path := range []string{".z", "node_modules"} {
		if got := root.Contents[path].Kind; got != wssync.KindUntracked {
			t.Errorf("%s kind = %v, want untracked", path, got)
		}
	}
	if got := root.Contents["nested"].Contents[".DS_Store"].Kind; got != wssync.KindUntracked {
		t.Errorf("nested/.DS_Store kind = %v, want untracked", got)
	}
	if root.Contents["CLAUDE.md"].Kind != wssync.KindFile {
		t.Error("CLAUDE.md was ignored")
	}
}

func TestScan_structural_names_match_only_at_the_top_level(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "docs/.z/notes.md", "kept\n", 0o644)

	root := scan(t, dir, wssync.Ignores{Structural: []string{".z"}})

	if got := root.Contents["docs"].Contents[".z"].Kind; got != wssync.KindDirectory {
		t.Errorf("docs/.z kind = %v, want directory", got)
	}
}

func TestScan_missing_root_is_an_absent_tree(t *testing.T) {
	root, err := wssync.Scan(filepath.Join(t.TempDir(), "nope"), wssync.NoIgnores())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if root != nil {
		t.Errorf("root = %+v, want nil", root)
	}
}

func TestPruneEmptyDirs_drops_dirs_with_nothing_to_propagate(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, "CLAUDE.md", "v1\n", 0o644)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	root := wssync.PruneEmptyDirs(scan(t, dir, wssync.NoIgnores()))

	if _, ok := root.Contents[".claude"]; ok {
		t.Error("empty directory survived pruning")
	}
	if _, ok := root.Contents["a"]; ok {
		t.Error("directory holding only empty directories survived pruning")
	}
	if _, ok := root.Contents["CLAUDE.md"]; !ok {
		t.Error("pruning dropped a file")
	}
}

func TestPruneEmptyDirs_keeps_a_dir_holding_a_problem(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "links"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink("elsewhere", filepath.Join(dir, "links", "l")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	root := wssync.PruneEmptyDirs(scan(t, dir, wssync.NoIgnores()))

	if _, ok := root.Contents["links"]; !ok {
		t.Fatal("pruning hid a directory holding a problem")
	}
	if len(root.Problems("")) != 1 {
		t.Errorf("problems = %+v, want the symlink reported", root.Problems(""))
	}
}

// An empty definition must remove its files, never the instance directory.
func TestPruneEmptyDirs_never_prunes_the_root(t *testing.T) {
	root := wssync.PruneEmptyDirs(scan(t, t.TempDir(), wssync.NoIgnores()))

	if root == nil {
		t.Fatal("the root was pruned; a nil definition tree makes the instance a deletion")
	}
	if root.Kind != wssync.KindDirectory {
		t.Errorf("root kind = %v, want directory", root.Kind)
	}
}
