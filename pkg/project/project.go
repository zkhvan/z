package project

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zkhvan/z/pkg/fd"
	"github.com/zkhvan/z/pkg/gh"
	"github.com/zkhvan/z/pkg/oslib"
)

type Config struct {
	MaxDepth       int      `json:"max_depth"`
	Root           string   `json:"root"`
	RemotePatterns []string `json:"remote_patterns"`
}

func (c Config) setDefaults() Config {
	c.MaxDepth = cmp.Or(c.MaxDepth, 4)

	if c.Root == "" {
		c.Root = "~/Projects"
	}

	return c
}

type ProjectType string

const (
	Local  ProjectType = "local"
	Remote ProjectType = "remote"
)

type Project struct {
	Type         ProjectType
	Name         string
	Identifier   string
	AbsolutePath string
	Path         string
}

func NewProject(name string, identifier string) Project {
	return Project{
		Type:       Local,
		Name:       name,
		Identifier: identifier,
	}
}

type ListOptions struct {
	Remote bool
}

func ListProjects(ctx context.Context, cfg Config, opts *ListOptions) ([]Project, error) {
	cfg = cfg.setDefaults()

	if opts == nil {
		opts = &ListOptions{}
	}

	projects, err := listLocalProjects(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if opts.Remote {
		remoteProjects, err := listRemoteProjects(ctx, cfg)
		if err != nil {
			return nil, err
		}

		projects = append(projects, remoteProjects...)
	}

	return projects, nil
}

func listLocalProjects(ctx context.Context, cfg Config) ([]Project, error) {
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
		rel, err := filepath.Rel(root, r)
		if err != nil {
			return nil, fmt.Errorf("error convert absolute path to relative path %q: %w", r, err)
		}

		abs := filepath.Dir(filepath.Clean(r))
		rel = filepath.Dir(rel)

		name := filepath.Base(rel)

		project := NewProject(name, rel)
		project.AbsolutePath = abs
		project.Path = rel

		projects = append(projects, project)
	}

	return projects, nil
}

func listRemoteProjects(ctx context.Context, cfg Config) ([]Project, error) {
	var projects []Project

	for _, p := range cfg.RemotePatterns {
		pattern, err := parseRemotePattern(p)
		if err != nil {
			return nil, err
		}

		var repos []*gh.Repo
		if pattern.Repo == nil {
			repos, err = gh.ListRepos(ctx, &gh.RepoListOptions{Owner: pattern.Owner})
			if err != nil {
				return nil, err
			}
		} else {
			repos = append(repos, &gh.Repo{Owner: pattern.Owner, Name: *pattern.Repo})
		}

		for _, r := range repos {
			project := NewProject(r.Name, r.String())
			project.Type = Remote

			projects = append(projects, project)
		}
	}

	return projects, nil
}

type remotePattern struct {
	Owner string
	Repo  *string
}

func parseRemotePattern(pattern string) (remotePattern, error) {
	parts := strings.Split(pattern, "/")
	if len(parts) != 2 {
		return remotePattern{}, fmt.Errorf("invalid pattern: %q", pattern)
	}

	owner := parts[0]
	repo := parts[1]

	result := remotePattern{
		Owner: owner,
	}

	if repo != "*" {
		result.Repo = &repo
	}

	return result, nil
}
