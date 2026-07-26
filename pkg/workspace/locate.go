package workspace

import (
	"path/filepath"
	"strings"
)

// Location is the workspace context of a directory.
type Location struct {
	Name string // instance directory name
	Dir  string // absolute path to the instance directory
	// Member is the worktree directory name of the member containing the
	// directory, empty when the directory is not inside one.
	Member string
}

// Root is the configured workspaces root.
func (s *Service) Root() string {
	return s.cfg.Root
}

// ResolveDir maps a directory to the workspace instance containing it,
// reporting false for the workspaces root itself and anything outside it.
//
// The name resolves structurally, whether or not the directory holds a
// manifest, so a caller reports a missing workspace with its own error. A
// member resolves only when the manifest declares it, which keeps `.z` and
// stray directories from posing as members.
func (s *Service) ResolveDir(dir string) (Location, bool, error) {
	root, err := realPath(s.cfg.Root)
	if err != nil {
		return Location{}, false, err
	}
	target, err := realPath(dir)
	if err != nil {
		return Location{}, false, err
	}

	rel, err := filepath.Rel(root, target)
	if err != nil {
		return Location{}, false, nil
	}

	segments := strings.Split(filepath.ToSlash(rel), "/")
	if ValidateName(segments[0]) != nil {
		return Location{}, false, nil // ".", "..", or an unusable directory name
	}

	loc := Location{
		Name: segments[0],
		Dir:  filepath.Join(s.cfg.Root, segments[0]),
	}
	if len(segments) > 1 && s.hasMemberDir(loc.Dir, segments[1]) {
		loc.Member = segments[1]
	}

	return loc, true, nil
}

func (s *Service) hasMemberDir(instanceDir, dirName string) bool {
	mf, err := readManifest(instanceDir)
	if err != nil {
		return false
	}
	for _, mm := range mf.Members {
		if memberFromManifest(mm).BaseName() == dirName {
			return true
		}
	}
	return false
}

// realPath compares paths the way the filesystem sees them, so a symlinked cwd
// still lands inside the root. A path that is not on disk yet keeps its
// cleaned form rather than failing the comparison outright.
func realPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil
	}
	return resolved, nil
}
