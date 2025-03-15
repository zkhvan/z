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
	MaxDepth         int      `json:"max_depth"`
	LocalDirectories []string `json:"local_directories"`
}

func (c Config) setDefaults() Config {
	c.MaxDepth = cmp.Or(c.MaxDepth, 4)

	if len(c.LocalDirectories) == 0 {
		c.LocalDirectories = append(c.LocalDirectories, "~/Projects")
	}

	return c
}

type ProjectType string

const (
	Local  ProjectType = "local"
)

type Project struct {
	Type         ProjectType
	Name         string
	AbsolutePath string
	Path         string
}

func NewProject(name string) Project {
	return Project{
		Type: Local,
		Name: name,
	}
}

func ListProjects(ctx context.Context, cfg Config) ([]Project, error) {
	cfg = cfg.setDefaults()

	projects, err := listLocalProjects(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func listLocalProjects(ctx context.Context, cfg Config) ([]Project, error) {
	var (
		glob        = true
		hidden      = true
		maxDepth    = cfg.MaxDepth
		noIgnoreVCS = true
		paths       = cfg.LocalDirectories
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

			abs := filepath.Dir(filepath.Clean(r))
			rel = filepath.Dir(rel)

			name := filepath.Base(rel)

			project := NewProject(name)
			project.AbsolutePath = abs
			project.Path = rel

			projects = append(projects, project)
		}
	}

	return projects, nil
}
