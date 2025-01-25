package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/samber/lo"

	"github.com/zkhvan/z/pkg/fd"
)

type Project struct {
	Name string
	Path string
}

func ListProjects(ctx context.Context) ([]Project, error) {
	var (
		glob        = true
		hidden      = true
		maxDepth    = 3
		noIgnoreVCS = true
		path        = os.ExpandEnv("$HOME/Projects")
	)

	results, err := fd.Run(
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

	ossPath := filepath.Join(path, "oss")
	ossResults, err := fd.Run(
		ctx,
		".git",
		&fd.FdOptions{
			Glob:        &glob,
			Hidden:      &hidden,
			MaxDepth:    &maxDepth,
			NoIgnoreVCS: &noIgnoreVCS,
			Path:        &ossPath,
		},
	)
	if err != nil {
		return nil, err
	}

	workPath := filepath.Join(path, "work")
	workResults, err := fd.Run(
		ctx,
		".git",
		&fd.FdOptions{
			Glob:        &glob,
			Hidden:      &hidden,
			MaxDepth:    &maxDepth,
			NoIgnoreVCS: &noIgnoreVCS,
			Path:        &workPath,
		},
	)
	if err != nil {
		return nil, err
	}

	results = append(results, ossResults...)
	results = append(results, workResults...)
	results = lo.Uniq(results)

	projects := make([]Project, 0, len(results))
	for _, result := range results {
		// Convert full paths to relative paths
		rel, err := filepath.Rel(path, result)
		if err != nil {
			return nil, fmt.Errorf("error convert absolute path to relative path %q: %w", result, err)
		}

		rel = filepath.Dir(rel)
		name := filepath.Base(rel)

		projects = append(projects, Project{Name: name, Path: rel})
	}

	return projects, nil
}
