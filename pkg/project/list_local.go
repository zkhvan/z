package project

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/zkhvan/z/pkg/fd"
)

func (s *Service) listLocalProjects(ctx context.Context, opts *ListOptions) ([]Project, error) {
	if !opts.Local {
		return nil, nil
	}

	// Local projects aren't cached, so we always load from the filesystem.
	return s.loadLocalProjects(ctx)
}

func (s *Service) loadLocalProjects(ctx context.Context) ([]Project, error) {
	var (
		glob   = true
		hidden = true
		// maxDepth should be increased by 1, so that it finds the `.git`
		// directory inside a project directory.
		maxDepth    = s.cfg.MaxDepth + 1
		noIgnoreVCS = true
		follow      = true
		root        = s.cfg.Root
	)

	rr, err := fd.Run(
		ctx,
		".git",
		&fd.Options{
			Glob:        &glob,
			Hidden:      &hidden,
			MaxDepth:    &maxDepth,
			NoIgnoreVCS: &noIgnoreVCS,
			Path:        &root,
			Follow:      &follow,
		},
	)
	if err != nil {
		return nil, err
	}

	var projects []Project
	for _, r := range rr {
		abs := filepath.Dir(filepath.Clean(r))

		id, err := filepath.Rel(root, abs)
		if err != nil {
			return nil, fmt.Errorf("error convert absolute path to relative path %q: %w", r, err)
		}

		project := newProject(
			id,
			s.toRemoteID(id),
			abs,
		)
		project.Source = SourceTypeLocal
		projects = append(projects, project)
	}

	return projects, nil
}
