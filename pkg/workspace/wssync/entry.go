// Package wssync propagates a workspace definition tree into an instance
// directory without silently clobbering local edits.
//
// The model is mutagen's three-way synchronization, reduced to one direction:
// the definition is alpha, the instance is beta, and the recorded sync state is
// the ancestor. The ancestor means "z wrote these exact bytes here, and unless
// someone else touched them they are still there" — a falsifiable claim about
// the instance directory, never a record of what the definition said. It
// advances only when a change is applied or when both sides independently
// converge, never on a conflict.
package wssync

import "sort"

type Kind uint8

const (
	KindFile Kind = iota
	KindDirectory
	// KindUntracked marks content excluded by an ignore rule. It is never
	// propagated and never removed, which is what keeps replica mode away from
	// member worktrees.
	KindUntracked
	// KindProblematic marks content that cannot be synchronized: a symlink, a
	// device node, an unreadable file. Reported and skipped, never fatal.
	KindProblematic
)

// Entry is one node of a scanned tree. Contents is populated for directories
// only; Hash and Exec for files only.
type Entry struct {
	Kind     Kind
	Hash     string
	Exec     bool
	Contents map[string]*Entry
	// Problem explains a KindProblematic entry, for reporting.
	Problem string
}

// Equal reports agreement at this node. A deep comparison also walks directory
// contents. Executability participates, so chmod +x is a change even though the
// content hash is untouched.
func (e *Entry) Equal(other *Entry, deep bool) bool {
	if e == nil || other == nil {
		return e == nil && other == nil
	}
	if e.Kind != other.Kind || e.Hash != other.Hash || e.Exec != other.Exec || e.Problem != other.Problem {
		return false
	}
	if !deep {
		return true
	}
	if len(e.Contents) != len(other.Contents) {
		return false
	}
	for name, child := range e.Contents {
		if !child.Equal(other.Contents[name], true) {
			return false
		}
	}
	return true
}

func (e *Entry) kindIs(k Kind) bool {
	return e != nil && e.Kind == k
}

func (e *Entry) nilOrUntracked() bool {
	return e == nil || e.Kind == KindUntracked
}

func (e *Entry) contents() map[string]*Entry {
	if e == nil {
		return nil
	}
	return e.Contents
}

// slim copies the node without its contents. Children are reconciled
// individually, so carrying them here would double-record every subtree.
func (e *Entry) slim() *Entry {
	if e == nil {
		return nil
	}
	return &Entry{Kind: e.Kind, Hash: e.Hash, Exec: e.Exec, Problem: e.Problem}
}

func (e *Entry) copy() *Entry {
	if e == nil {
		return nil
	}
	c := e.slim()
	if len(e.Contents) > 0 {
		c.Contents = make(map[string]*Entry, len(e.Contents))
		for name, child := range e.Contents {
			c.Contents[name] = child.copy()
		}
	}
	return c
}

// synchronizable strips content z is not allowed to touch. Reconciliation runs
// against the result, so ignored and problematic content can never be
// propagated or removed; the caller separately checks whether stripping changed
// anything, which is how such content blocks an otherwise-valid change.
func (e *Entry) synchronizable() *Entry {
	if e == nil || e.Kind == KindUntracked || e.Kind == KindProblematic {
		return nil
	}
	if e.Kind != KindDirectory {
		return e.slim()
	}
	stripped := e.slim()
	for name, child := range e.Contents {
		if s := child.synchronizable(); s != nil {
			if stripped.Contents == nil {
				stripped.Contents = make(map[string]*Entry)
			}
			stripped.Contents[name] = s
		}
	}
	return stripped
}

// Problems collects every problematic node under e, keyed by path.
func (e *Entry) Problems(path string) []Problem {
	if e == nil {
		return nil
	}
	if e.Kind == KindProblematic {
		return []Problem{{Path: path, Reason: e.Problem}}
	}
	var problems []Problem
	for _, name := range sortedNames(e.Contents) {
		problems = append(problems, e.Contents[name].Problems(join(path, name))...)
	}
	return problems
}

// Lookup returns the entry at a slash-separated path, or nil when the tree does
// not hold one.
func Lookup(root *Entry, path string) *Entry {
	current := root
	for _, segment := range splitPath(path) {
		if !current.kindIs(KindDirectory) {
			return nil
		}
		current = current.Contents[segment]
	}
	return current
}

// Problem is a path the scan could not represent as synchronizable content.
type Problem struct {
	Path   string
	Reason string
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "/" + name
}

func sortedNames(contents map[string]*Entry) []string {
	names := make([]string, 0, len(contents))
	for name := range contents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// nameUnion returns every child name across the given content maps, sorted so
// reconciliation output is deterministic.
func nameUnion(maps ...map[string]*Entry) []string {
	seen := make(map[string]struct{})
	for _, m := range maps {
		for name := range m {
			seen[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
