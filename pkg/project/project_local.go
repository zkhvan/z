package project

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/zkhvan/z/pkg/fd"
	"github.com/zkhvan/z/pkg/oslib"
)

func newLocalProject(id, abs string) Project {
	return Project{
		Type:         Local,
		ID:           id,
		AbsolutePath: abs,
	}
}

func listLocalProjects(ctx context.Context, cfg Config, opts *ListOptions) ([]Project, error) {
	if !opts.Local {
		return nil, nil
	}

	// Local projects aren't cached, so we always load from the filesystem.
	return loadLocalProjects(ctx, cfg)
}

func loadLocalProjects(ctx context.Context, cfg Config) ([]Project, error) {
	var (
		glob        = true
		hidden      = true
		maxDepth    = cfg.MaxDepth
		noIgnoreVCS = true
		root        = oslib.Expand(cfg.Root)
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
