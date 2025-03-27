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

	projects := combineProjects(remoteProjects, localProjects)
	return projects, nil
}

func combineProjects(remote, local []Project) []Project {
	projects := make([]Project, 0, len(remote)+len(local))

	projects = append(projects, remote...)
	projects = append(projects, local...)

	projects = lo.UniqBy(projects, func(p Project) string {
		return p.AbsolutePath
	})

	slices.SortFunc(projects, func(a, b Project) int {
		return a.Compare(b)
	})

	return projects
}
