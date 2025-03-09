package project

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"

	"github.com/zkhvan/z/pkg/fd"
	"github.com/zkhvan/z/pkg/oslib"
)

type Config struct {
	MaxDepth          int      `json:"max_depth"`
	SearchDirectories []string `json:"search_directories"`
}

func (c Config) setDefaults() Config {
	c.MaxDepth = cmp.Or(c.MaxDepth, 3)

	if len(c.SearchDirectories) == 0 {
		c.SearchDirectories = append(c.SearchDirectories, "~/Projects")
	}

	return c
}

type Project struct {
	Name string
	Path string
}

func ListProjects(ctx context.Context, cfg Config) ([]Project, error) {
	cfg = cfg.setDefaults()

	var (
		glob        = true
		hidden      = true
		maxDepth    = cfg.MaxDepth
		noIgnoreVCS = true
		paths       = cfg.SearchDirectories
	)

	var projects []Project
	for _, path := range paths {
		path = oslib.Expand(path)

		rr, err := fd.Run(
			ctx,
			".git",
			&fd.FdOptions{
				Glob:        &glob,
				Hidden:      &hidden,
				MaxDepth:    &maxDepth,
				NoIgnoreVCS: &noIgnoreVCS,
				Path:        &path,
			},
		)
		if err != nil {
			return nil, err
		}

		for _, r := range rr {
			rel, err := filepath.Rel(path, r)
			if err != nil {
				return nil, fmt.Errorf("error convert absolute path to relative path %q: %w", r, err)
			}

			rel = filepath.Dir(rel)
			name := filepath.Base(rel)

			projects = append(projects, Project{Name: name, Path: rel})
		}
	}

	return projects, nil
}
