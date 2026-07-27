package wssync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const (
	fileMode     = 0o600
	execFileMode = 0o700
	dirMode      = 0o700
)

// Transition applies changes to the instance, taking content from the
// definition. It returns the changes that actually landed, so a caller that
// fails partway still advances the ancestor by exactly what is on disk — the
// bookkeeping that makes the next run resume rather than re-conflict.
func Transition(instanceDir, definitionDir string, changes []Change) ([]Change, error) {
	ordered := make([]Change, len(changes))
	copy(ordered, changes)

	// Deletions first, so a path can change from a file to a directory (or back)
	// within a single sync.
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Deletion() != ordered[j].Deletion() {
			return ordered[i].Deletion()
		}
		return ordered[i].Path < ordered[j].Path
	})

	applied := make([]Change, 0, len(ordered))
	for _, c := range ordered {
		target := filepath.Join(instanceDir, filepath.FromSlash(c.Path))

		if c.Deletion() {
			if err := os.RemoveAll(target); err != nil {
				return applied, fmt.Errorf("removing %s: %w", c.Path, err)
			}
			applied = append(applied, c)
			continue
		}

		source := filepath.Join(definitionDir, filepath.FromSlash(c.Path))
		if err := writeEntry(target, source, c.New); err != nil {
			return applied, fmt.Errorf("writing %s: %w", c.Path, err)
		}
		applied = append(applied, c)
	}
	return applied, nil
}

func writeEntry(target, source string, e *Entry) error {
	switch e.Kind {
	case KindDirectory:
		if info, err := os.Lstat(target); err == nil && !info.IsDir() {
			if err := os.Remove(target); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(target, dirMode); err != nil {
			return err
		}
		for _, name := range sortedNames(e.Contents) {
			child := filepath.Join(target, name)
			if err := writeEntry(child, filepath.Join(source, name), e.Contents[name]); err != nil {
				return err
			}
		}
		return nil
	case KindFile:
		return copyFile(target, source, e.Exec)
	default:
		return nil
	}
}

func copyFile(target, source string, exec bool) error {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, dirMode); err != nil {
		return err
	}
	if info, err := os.Lstat(target); err == nil && info.IsDir() {
		if err := os.RemoveAll(target); err != nil {
			return err
		}
	}

	in, err := os.Open(source) //nolint:gosec // path is derived from a scan of the definition
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	tmp, err := os.CreateTemp(parent, ".z-sync-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}

	mode := os.FileMode(fileMode)
	if exec {
		mode = execFileMode
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
