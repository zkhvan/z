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
	"github.com/zkhvan/z/pkg/oslib"
)

func newRemoteProject(id, abs, remoteID string) Project {
	return Project{
		Type:         Remote,
		ID:           id,
		AbsolutePath: abs,
		RemoteID:     remoteID,
	}
}

func (s *Service) listRemoteProjects(ctx context.Context, opts *ListOptions) ([]Project, error) {
	var err error
	var projects []Project

	if !opts.Remote {
		return nil, nil
	}

	if !opts.RefreshCache {
		projects, err = fcache.LoadMany[Project](opts.CacheDir, "projects.remote")
		if errors.Is(err, fcache.ErrNotFound) {
			projects, err = s.loadRemoteProjects(ctx)
			if err != nil {
				return nil, fmt.Errorf("error loading remote projects: %w", err)
			}

			ttl := time.Now().Add(time.Duration(s.cfg.TTL) * time.Second)
			if err := fcache.SaveMany(opts.CacheDir, "projects.remote", projects, ttl); err != nil {
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
	if err := fcache.SaveMany(opts.CacheDir, "projects.remote", projects, ttl); err != nil {
		return nil, fmt.Errorf("error saving remote projects to cache: %w", err)
	}

	return projects, nil
}

func (s *Service) loadRemoteProjects(ctx context.Context) ([]Project, error) {
	var (
		projects = make([]Project, 0)
		root     = oslib.Expand(s.cfg.Root)
	)

	for _, rp := range s.cfg.RemotePatterns {
		pattern, err := parseRemotePattern(rp)
		if err != nil {
			return nil, fmt.Errorf("error parsing remote pattern %q: %w", rp, err)
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
			abs := filepath.Join(root, r.String())
			if pattern.AlternatePath != nil {
				alt := *pattern.AlternatePath
				dir := r.String()

				if strings.HasSuffix(alt, "/") {
					dir = r.Name
				}

				abs = filepath.Join(root, alt, dir)
			}

			id, err := filepath.Rel(root, abs)
			if err != nil {
				return nil, fmt.Errorf("error convert absolute path to relative path %q: %w", abs, err)
			}

			project := newRemoteProject(id, abs, r.String())
			projects = append(projects, project)
		}
	}

	return projects, nil
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
