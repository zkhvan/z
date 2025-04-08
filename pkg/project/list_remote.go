package project

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/zkhvan/z/pkg/fcache"
	"github.com/zkhvan/z/pkg/gh"
)

func (s *Service) listRemoteProjects(ctx context.Context, opts *ListOptions) ([]Project, error) {
	var err error
	var projects []Project

	if !opts.Remote {
		return nil, nil
	}

	if !s.refreshCache {
		projects, err = fcache.LoadMany[Project](s.cacheDir, "projects.remote")
		if errors.Is(err, fcache.ErrNotFound) {
			projects, err = s.loadRemoteProjects(ctx)
			if err != nil {
				return nil, fmt.Errorf("error loading remote projects: %w", err)
			}

			ttl := time.Now().Add(time.Duration(s.cfg.TTL) * time.Second)
			if err = fcache.SaveMany(s.cacheDir, "projects.remote", projects, ttl); err != nil {
				return nil, fmt.Errorf("error saving remote projects to cache: %w", err)
			}

			return projects, nil
		}
		if err != nil {
			return nil, fmt.Errorf("error loading cached remote projects: %w", err)
		}

		return projects, nil
	}

	projects, err = s.loadRemoteProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading remote projects: %w", err)
	}

	ttl := time.Now().Add(time.Duration(s.cfg.TTL) * time.Second)
	if err := fcache.SaveMany(s.cacheDir, "projects.remote", projects, ttl); err != nil {
		return nil, fmt.Errorf("error saving remote projects to cache: %w", err)
	}

	return projects, nil
}

func (s *Service) loadRemoteProjects(ctx context.Context) ([]Project, error) {
	var (
		projects = make([]Project, 0)
		root     = s.cfg.Root
	)

	for _, pattern := range s.cfg.remotePatterns {
		repos, err := s.loadRemoteRepos(ctx, pattern)
		if err != nil {
			return nil, fmt.Errorf("error loading remote repos: %w", err)
		}

		for _, r := range repos {
			localID := s.toLocalID(r.String())

			project := newProject(
				localID,
				r.String(),
				filepath.Join(root, localID),
			)
			project.Source = SourceTypeRemote

			projects = append(projects, project)
		}
	}

	return projects, nil
}

func (s *Service) loadRemoteRepos(ctx context.Context, pattern remotePattern) ([]*gh.Repo, error) {
	if pattern.Repo != "*" {
		// If the repo is specified, return a single repo.
		return []*gh.Repo{
			{
				Owner: pattern.Owner,
				Name:  pattern.Repo,
			},
		}, nil
	}

	repos, err := s.gh.ListRepos(ctx, &gh.RepoListOptions{Owner: pattern.Owner})
	if err != nil {
		return nil, err
	}

	return repos, nil
}
