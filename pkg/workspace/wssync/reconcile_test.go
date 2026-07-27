package wssync_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/zkhvan/z/pkg/workspace/wssync"
)

func file(hash string) *wssync.Entry {
	return &wssync.Entry{Kind: wssync.KindFile, Hash: hash}
}

func execFile(hash string) *wssync.Entry {
	return &wssync.Entry{Kind: wssync.KindFile, Hash: hash, Exec: true}
}

func dir(children map[string]*wssync.Entry) *wssync.Entry {
	return &wssync.Entry{Kind: wssync.KindDirectory, Contents: children}
}

func untracked() *wssync.Entry {
	return &wssync.Entry{Kind: wssync.KindUntracked}
}

func problematic(reason string) *wssync.Entry {
	return &wssync.Entry{Kind: wssync.KindProblematic, Problem: reason}
}

type reconcileCase struct {
	name       string
	ancestor   *wssync.Entry
	definition *wssync.Entry
	instance   *wssync.Entry
	mode       wssync.Mode

	changes   []string
	ancestors []string
	conflicts []string
}

func TestReconcile(t *testing.T) {
	cases := []reconcileCase{
		{
			name:       "first sync writes every definition file",
			definition: dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			instance:   dir(map[string]*wssync.Entry{}),
			changes:    []string{"write CLAUDE.md=v1"},
			ancestors:  []string{"record =<dir>"},
		},
		{
			name:       "unmodified instance file is updated",
			ancestor:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"CLAUDE.md": file("v2")}),
			instance:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			changes:    []string{"write CLAUDE.md=v2"},
		},
		{
			name:       "unchanged definition and instance plan nothing",
			ancestor:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			instance:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
		},
		{
			name:       "locally modified file conflicts and is left alone",
			ancestor:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"CLAUDE.md": file("v2")}),
			instance:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("mine")}),
			conflicts:  []string{"CLAUDE.md"},
		},
		{
			name:       "conflict does not advance the ancestor",
			ancestor:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"CLAUDE.md": file("v2")}),
			instance:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("mine")}),
			conflicts:  []string{"CLAUDE.md"},
			ancestors:  nil,
		},
		{
			name:       "local edit matching the new definition converges without conflict",
			ancestor:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"CLAUDE.md": file("v2")}),
			instance:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("v2")}),
			ancestors:  []string{"record CLAUDE.md=v2"},
		},
		{
			name:       "interrupted sync recovers because both sides already agree",
			ancestor:   dir(map[string]*wssync.Entry{"a.md": file("v1"), "b.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"a.md": file("v2"), "b.md": file("v2")}),
			instance:   dir(map[string]*wssync.Entry{"a.md": file("v2"), "b.md": file("v1")}),
			changes:    []string{"write b.md=v2"},
			ancestors:  []string{"record a.md=v2"},
		},
		{
			name:       "deleted instance file comes back",
			ancestor:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			instance:   dir(map[string]*wssync.Entry{}),
			changes:    []string{"write CLAUDE.md=v1"},
		},
		{
			name:       "file removed from the definition is deleted when unmodified",
			ancestor:   dir(map[string]*wssync.Entry{"NOTES.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{}),
			instance:   dir(map[string]*wssync.Entry{"NOTES.md": file("v1")}),
			changes:    []string{"delete NOTES.md"},
		},
		{
			name:       "file removed from the definition is untracked when modified",
			ancestor:   dir(map[string]*wssync.Entry{"NOTES.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{}),
			instance:   dir(map[string]*wssync.Entry{"NOTES.md": file("mine")}),
			ancestors:  []string{"forget NOTES.md"},
		},
		{
			name:       "untracked file the definition never shipped is left alone",
			ancestor:   dir(map[string]*wssync.Entry{}),
			definition: dir(map[string]*wssync.Entry{}),
			instance:   dir(map[string]*wssync.Entry{"scratch.md": file("mine")}),
		},
		{
			name:       "executability change propagates",
			ancestor:   dir(map[string]*wssync.Entry{"hook": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"hook": execFile("v1")}),
			instance:   dir(map[string]*wssync.Entry{"hook": file("v1")}),
			changes:    []string{"write hook=v1+x"},
		},
		{
			name:       "locally chmod-ed file conflicts",
			ancestor:   dir(map[string]*wssync.Entry{"hook": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"hook": file("v2")}),
			instance:   dir(map[string]*wssync.Entry{"hook": execFile("v1")}),
			conflicts:  []string{"hook"},
		},
		{
			name:     "nested definition files reconcile individually",
			ancestor: dir(map[string]*wssync.Entry{".claude": dir(map[string]*wssync.Entry{"settings.json": file("v1")})}),
			definition: dir(map[string]*wssync.Entry{".claude": dir(map[string]*wssync.Entry{
				"settings.json": file("v2"),
				"agents.md":     file("new"),
			})}),
			instance: dir(map[string]*wssync.Entry{".claude": dir(map[string]*wssync.Entry{"settings.json": file("v1")})}),
			changes:  []string{"write .claude/agents.md=new", "write .claude/settings.json=v2"},
		},
		{
			name:       "ignored worktree is never touched",
			ancestor:   dir(map[string]*wssync.Entry{}),
			definition: dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			instance:   dir(map[string]*wssync.Entry{"api": untracked()}),
			changes:    []string{"write CLAUDE.md=v1"},
		},
		{
			name:       "ignored worktree survives replica mode",
			mode:       wssync.ModeReplica,
			ancestor:   dir(map[string]*wssync.Entry{}),
			definition: dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			instance:   dir(map[string]*wssync.Entry{"api": untracked(), "junk.md": file("mine")}),
			changes:    []string{"write CLAUDE.md=v1", "delete junk.md"},
		},
		{
			name:       "replica mode overwrites local edits",
			mode:       wssync.ModeReplica,
			ancestor:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"CLAUDE.md": file("v2")}),
			instance:   dir(map[string]*wssync.Entry{"CLAUDE.md": file("mine")}),
			changes:    []string{"write CLAUDE.md=v2"},
		},
		{
			name:       "replica mode removes a file the definition dropped even when modified",
			mode:       wssync.ModeReplica,
			ancestor:   dir(map[string]*wssync.Entry{"NOTES.md": file("v1")}),
			definition: dir(map[string]*wssync.Entry{}),
			instance:   dir(map[string]*wssync.Entry{"NOTES.md": file("mine")}),
			changes:    []string{"delete NOTES.md"},
		},
		{
			name:       "problematic instance entry is skipped without conflict",
			ancestor:   dir(map[string]*wssync.Entry{}),
			definition: dir(map[string]*wssync.Entry{"link": file("v1")}),
			instance:   dir(map[string]*wssync.Entry{"link": problematic("symbolic link")}),
		},
		{
			name:       "problematic definition entry does not block its siblings",
			ancestor:   dir(map[string]*wssync.Entry{}),
			definition: dir(map[string]*wssync.Entry{"link": problematic("symbolic link"), "CLAUDE.md": file("v1")}),
			instance:   dir(map[string]*wssync.Entry{}),
			changes:    []string{"write CLAUDE.md=v1"},
		},
		{
			name:       "file replaced by a directory in the definition",
			ancestor:   dir(map[string]*wssync.Entry{"hooks": file("v1")}),
			definition: dir(map[string]*wssync.Entry{"hooks": dir(map[string]*wssync.Entry{"post-create": file("v1")})}),
			instance:   dir(map[string]*wssync.Entry{"hooks": file("v1")}),
			changes:    []string{"write hooks=<dir:post-create>"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := wssync.Reconcile(tc.ancestor, tc.definition, tc.instance, tc.mode)

			if diff := cmp.Diff(tc.changes, describeChanges(got.Changes, "write", "delete")); diff != "" {
				t.Errorf("changes mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.ancestors, describeChanges(got.AncestorChanges, "record", "forget")); diff != "" {
				t.Errorf("ancestor changes mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.conflicts, describeConflicts(got.Conflicts)); diff != "" {
				t.Errorf("conflicts mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcile_ancestor_advances_only_for_applied_changes(t *testing.T) {
	ancestor := dir(map[string]*wssync.Entry{"a.md": file("v1"), "b.md": file("v1")})
	definition := dir(map[string]*wssync.Entry{"a.md": file("v2"), "b.md": file("v2")})
	instance := dir(map[string]*wssync.Entry{"a.md": file("v1"), "b.md": file("mine")})

	got := wssync.Reconcile(ancestor, definition, instance, wssync.ModeSafe)

	// Only a.md was applicable; b.md conflicted and must keep its old record so
	// the conflict is reported again next run.
	next := wssync.Apply(ancestor, append(got.AncestorChanges, got.Changes...))

	if want := "v2"; next.Contents["a.md"].Hash != want {
		t.Errorf("a.md recorded hash = %q, want %q", next.Contents["a.md"].Hash, want)
	}
	if want := "v1"; next.Contents["b.md"].Hash != want {
		t.Errorf("b.md recorded hash = %q, want %q", next.Contents["b.md"].Hash, want)
	}
}

func TestApply_records_nested_paths_and_forgets_removed_ones(t *testing.T) {
	root := wssync.Apply(nil, []wssync.Change{
		{Path: ".claude", New: dir(nil)},
		{Path: ".claude/settings.json", New: file("v1")},
		{Path: "CLAUDE.md", New: file("v1")},
	})

	if got := root.Contents[".claude"].Contents["settings.json"].Hash; got != "v1" {
		t.Fatalf("nested hash = %q, want %q", got, "v1")
	}

	root = wssync.Apply(root, []wssync.Change{{Path: ".claude/settings.json"}})
	if _, ok := root.Contents[".claude"].Contents["settings.json"]; ok {
		t.Error("settings.json still recorded after removal")
	}
	if _, ok := root.Contents["CLAUDE.md"]; !ok {
		t.Error("CLAUDE.md was dropped by an unrelated removal")
	}
}

func describeChanges(changes []wssync.Change, write, remove string) []string {
	var out []string
	for _, c := range changes {
		if c.Deletion() {
			out = append(out, fmt.Sprintf("%s %s", remove, c.Path))
			continue
		}
		out = append(out, fmt.Sprintf("%s %s=%s", write, c.Path, describeEntry(c.New)))
	}
	return out
}

func describeEntry(e *wssync.Entry) string {
	if e.Kind == wssync.KindDirectory {
		if len(e.Contents) == 0 {
			return "<dir>"
		}
		names := make([]string, 0, len(e.Contents))
		for name := range e.Contents {
			names = append(names, name)
		}
		sort.Strings(names)
		return "<dir:" + strings.Join(names, ",") + ">"
	}
	if e.Exec {
		return e.Hash + "+x"
	}
	return e.Hash
}

func describeConflicts(conflicts []wssync.Conflict) []string {
	var out []string
	for _, c := range conflicts {
		out = append(out, c.Path)
	}
	return out
}
