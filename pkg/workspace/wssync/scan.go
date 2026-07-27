package wssync

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// HashPrefix names the digest algorithm in recorded hashes, so a future change
// is detectable rather than silent.
const HashPrefix = "sha256:"

// Scan reads a tree into memory. A missing root is an absent tree, not an
// error: an instance may be scanned before anything has been written into it.
func Scan(root string, ignore Matcher) (*Entry, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scanning %s: not a directory", root)
	}
	return scanDir(root, "", ignore)
}

func scanDir(dir, relPath string, ignore Matcher) (*Entry, error) {
	names, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	// A directory holding a git repository is never definition-authored
	// content. This is structural and non-overridable: it is the backstop that
	// keeps replica mode away from a worktree whose member was hand-edited out
	// of the manifest.
	if relPath != "" && holdsGitRepo(names) {
		return &Entry{Kind: KindUntracked}, nil
	}

	entry := &Entry{Kind: KindDirectory}
	for _, e := range names {
		childRel := join(relPath, e.Name())
		childPath := filepath.Join(dir, e.Name())

		child, err := scanEntry(childPath, childRel, e, ignore)
		if err != nil {
			return nil, err
		}
		if child == nil {
			continue
		}
		if entry.Contents == nil {
			entry.Contents = make(map[string]*Entry)
		}
		entry.Contents[e.Name()] = child
	}
	return entry, nil
}

func scanEntry(path, relPath string, e fs.DirEntry, ignore Matcher) (*Entry, error) {
	if ignore.Ignore(relPath, e.IsDir()) {
		return &Entry{Kind: KindUntracked}, nil
	}

	switch {
	case e.Type()&fs.ModeSymlink != 0:
		// Copying a link's target is what silently produces the
		// container-breaking outcome sync exists to avoid.
		return &Entry{Kind: KindProblematic, Problem: "symbolic link"}, nil
	case e.IsDir():
		return scanDir(path, relPath, ignore)
	case !e.Type().IsRegular():
		return &Entry{Kind: KindProblematic, Problem: "not a regular file"}, nil
	}

	info, err := e.Info()
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspecting %s: %w", path, err)
	}

	hash, err := hashFile(path)
	if err != nil {
		return &Entry{Kind: KindProblematic, Problem: "unreadable"}, nil
	}

	return &Entry{Kind: KindFile, Hash: hash, Exec: info.Mode()&0o111 != 0}, nil
}

// PruneEmptyDirs drops directories whose subtree holds nothing worth
// propagating. Git cannot represent an empty directory, so a definition cannot
// reliably ship one; propagating them anyway would create instance directories
// that no recorded file explains — which `delete`'s guard would then correctly
// but uselessly report as unmanaged.
//
// Problematic entries count as content, so pruning never hides a problem the
// user should see.
func PruneEmptyDirs(e *Entry) *Entry {
	if e == nil || e.Kind != KindDirectory {
		return e
	}
	// The root is never a deletion candidate. A definition that legitimately
	// ships nothing must remove its files, not the instance directory.
	if pruned := pruneEmptyDirs(e); pruned != nil {
		return pruned
	}
	return &Entry{Kind: KindDirectory}
}

func pruneEmptyDirs(e *Entry) *Entry {
	if e == nil || e.Kind != KindDirectory {
		return e
	}

	pruned := e.slim()
	for name, child := range e.Contents {
		kept := pruneEmptyDirs(child)
		if kept == nil {
			continue
		}
		if pruned.Contents == nil {
			pruned.Contents = make(map[string]*Entry)
		}
		pruned.Contents[name] = kept
	}

	if len(pruned.Contents) == 0 {
		return nil
	}
	return pruned
}

func holdsGitRepo(entries []fs.DirEntry) bool {
	for _, e := range entries {
		if e.Name() == ".git" {
			return true
		}
	}
	return false
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // path comes from a directory walk
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return HashPrefix + hex.EncodeToString(h.Sum(nil)), nil
}
