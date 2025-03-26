package project

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/zkhvan/z/pkg/fd"
)

func newLocalProject(id, abs string) Project {
	return Project{
		Type:         Local,
		LocalID:      id,
		AbsolutePath: abs,
	}
}

func (s *Service) listLocalProjects(ctx context.Context, opts *ListOptions) ([]Project, error) {
	if !opts.Local {
		return nil, nil
	}

	// Local projects aren't cached, so we always load from the filesystem.
	return s.loadLocalProjects(ctx)
}

func (s *Service) loadLocalProjects(ctx context.Context) ([]Project, error) {
	var (
		glob        = true
		hidden      = true
		maxDepth    = s.cfg.MaxDepth
		noIgnoreVCS = true
		root        = s.cfg.Root
	)

	var projects []Project
	rr, err := fd.Run(
		ctx,
		".git",
		&fd.FdOptions{
			Glob:        &glob,
			Hidden:      &hidden,
			MaxDepth:    &maxDepth,
			NoIgnoreVCS: &noIgnoreVCS,
			Path:        &root,
		},
	)
	if err != nil {
		return nil, err
	}

	for _, r := range rr {
		abs := filepath.Dir(filepath.Clean(r))

		id, err := filepath.Rel(root, abs)
		if err != nil {
			return nil, fmt.Errorf("error convert absolute path to relative path %q: %w", r, err)
		}

		project := newLocalProject(id, abs)
		projects = append(projects, project)
	}

	return projects, nil

}
