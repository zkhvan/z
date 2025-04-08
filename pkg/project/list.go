package project

import (
	"context"
	"fmt"
	"slices"

	"github.com/samber/lo"
)

type ListOptions struct {
	Local  bool
	Remote bool
}

// ListProjects will search for repositories using the given config and options.
//
// By default, it will only search for local repositories. To search for remote
// repositories, set opts.Remote to true.
func (s *Service) ListProjects(ctx context.Context, opts *ListOptions) ([]Project, error) {
	if opts == nil {
		opts = &ListOptions{}
	}

	remoteProjects, err := s.listRemoteProjects(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing remote projects: %w", err)
	}

	localProjects, err := s.listLocalProjects(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing local projects: %w", err)
	}

	projects, err := combineProjects(remoteProjects, localProjects)
	if err != nil {
		return nil, fmt.Errorf("error combining projects: %w", err)
	}

	return projects, nil
}

func combineProjects(remote, local []Project) ([]Project, error) {
	projects := make(map[string]Project, 0)

	var err error
	for _, p := range remote {
		if _, ok := projects[p.AbsolutePath]; ok {
			p, err = combineProject(projects[p.AbsolutePath], p)
			if err != nil {
				return nil, fmt.Errorf("error combining projects: %w", err)
			}
		}

		projects[p.AbsolutePath] = p
	}

	for _, p := range local {
		if _, ok := projects[p.AbsolutePath]; ok {
			p, err = combineProject(projects[p.AbsolutePath], p)
			if err != nil {
				return nil, fmt.Errorf("error combining projects: %w", err)
			}
		}

		projects[p.AbsolutePath] = p
	}

	result := lo.Values(projects)
	slices.SortFunc(result, func(a, b Project) int {
		return a.Compare(b)
	})
	return result, nil
}

func combineProject(a, b Project) (Project, error) {
	if a.LocalID != b.LocalID {
		return Project{}, fmt.Errorf("local id mismatch: %s != %s", a.LocalID, b.LocalID)
	}

	if a.RemoteID != b.RemoteID {
		return Project{}, fmt.Errorf("remote id mismatch: %s != %s", a.RemoteID, b.RemoteID)
	}

	if a.AbsolutePath != b.AbsolutePath {
		return Project{}, fmt.Errorf("absolute path mismatch: %s != %s", a.AbsolutePath, b.AbsolutePath)
	}

	if a.Source == SourceTypeUnknown || b.Source == SourceTypeUnknown {
		return Project{}, fmt.Errorf("source unknown")
	}

	if a.Source == b.Source {
		return a, nil
	}

	p := newProject(
		a.LocalID,
		a.RemoteID,
		a.AbsolutePath,
	)

	p.Source = SourceTypeSynced

	return p, nil
}
