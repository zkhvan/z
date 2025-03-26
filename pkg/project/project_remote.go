package project

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/zkhvan/z/pkg/fcache"
	"github.com/zkhvan/z/pkg/gh"
)

func newRemoteProject(root string, pattern remotePattern, repo *gh.Repo) Project {
	id := repo.String()
	if pattern.AlternatePath != nil {
		id = filepath.Join(*pattern.AlternatePath, id)
	}

	abs := filepath.Join(root, id)

	return Project{
		Type:         Remote,
		LocalID:      id,
		AbsolutePath: abs,
		RemoteID:     repo.String(),
	}
}

func (s *Service) GetRemoteProject(ctx context.Context, remoteID string) (Project, error) {
	root := s.cfg.Root

	localID := s.toLocalID(remoteID)

	return newProject(
		localID,
		remoteID,
		filepath.Join(root, localID),
	), nil
}

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
			if err := fcache.SaveMany(s.cacheDir, "projects.remote", projects, ttl); err != nil {
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
			project := newRemoteProject(root, pattern, r)
			projects = append(projects, project)
		}
	}

	return projects, nil
}

func (s *Service) loadRemoteRepos(ctx context.Context, pattern remotePattern) ([]*gh.Repo, error) {
	if pattern.Repo != nil {
		// If the repo is specified, return a single repo.
		return []*gh.Repo{
			{
				Owner: pattern.Owner,
				Name:  *pattern.Repo,
			},
		}, nil
	}

	repos, err := gh.ListRepos(ctx, &gh.RepoListOptions{Owner: pattern.Owner})
	if err != nil {
		return nil, err
	}

	return repos, nil
}

type remotePattern struct {
	original string

	Owner         string
	Repo          *string
	AlternatePath *string
}

// parseRemotePattern will parse the pattern with the following format:
//
//	owner/repo -> ./alternate-path
//
// If the pattern is in the format above, the AlternatePath will be set.
func parseRemotePattern(pattern string) (remotePattern, error) {
	out := remotePattern{
		original: pattern,
	}

	// Check and parse if the pattern contains an alternate path.
	parts := strings.Split(pattern, "->")
	if len(parts) == 2 {
		alternatePath := strings.TrimSpace(parts[1])
		out.AlternatePath = &alternatePath
	}

	// Parse the owner/repo
	parts = strings.Split(strings.TrimSpace(parts[0]), "/")
	if len(parts) != 2 {
		return out, fmt.Errorf("invalid pattern: %q", pattern)
	}

	out.Owner = parts[0]
	out.Repo = &parts[1]

	if *out.Repo == "*" {
		out.Repo = nil
	}

	return out, nil
}
