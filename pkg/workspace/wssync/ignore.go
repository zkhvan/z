package wssync

import (
	"path"
	"strings"
)

// Matcher decides whether a scanned path is off-limits. Ignored content becomes
// KindUntracked: never propagated, never removed — which is what stops replica
// mode from deleting a member worktree.
type Matcher interface {
	Ignore(relPath string, isDir bool) bool
}

// Ignores is the composed rule set. Structural entries are supplied by z and
// cannot be overridden; Patterns come from the definition manifest and global
// config.
type Ignores struct {
	// Structural names are matched against top-level entries only, because
	// that is where `.z` and the member worktrees live.
	Structural []string
	Patterns   []string
}

// Ignore applies gitignore-lite semantics, deliberately narrower than
// gitignore: a pattern without a separator matches a base name at any depth, a
// trailing slash restricts a pattern to directories, and a pattern containing a
// separator is anchored to the sync root. There is no `**` and no negation, so
// no dependency and nothing to explain beyond three sentences.
func (i Ignores) Ignore(relPath string, isDir bool) bool {
	if relPath == "" {
		return false
	}

	if !strings.Contains(relPath, "/") {
		for _, name := range i.Structural {
			if name == relPath {
				return true
			}
		}
	}

	base := path.Base(relPath)
	for _, pattern := range i.Patterns {
		if pattern == "" {
			continue
		}

		dirOnly := strings.HasSuffix(pattern, "/")
		trimmed := strings.TrimSuffix(pattern, "/")
		if dirOnly && !isDir {
			continue
		}

		target := base
		if strings.Contains(trimmed, "/") {
			target = relPath
		}
		if ok, err := path.Match(trimmed, target); err == nil && ok {
			return true
		}
	}
	return false
}

// noIgnores is used when a caller has nothing to exclude.
type noIgnores struct{}

func (noIgnores) Ignore(string, bool) bool { return false }

// NoIgnores matches nothing.
func NoIgnores() Matcher { return noIgnores{} }
