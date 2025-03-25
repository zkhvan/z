package project

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/zkhvan/z/pkg/fcache"
)

type ProjectType string

const (
	Local  ProjectType = "local"
	Remote ProjectType = "remote"
)

type Project struct {
	Type         ProjectType `json:"type"`
	ID           string      `json:"id"`
	AbsolutePath string      `json:"absolute_path"`
}

func (p Project) Compare(other Project) int {
	return strings.Compare(p.AbsolutePath, other.AbsolutePath)
}

type ListOptions struct {
	Local  bool
	Remote bool

	RefreshCache bool
	CacheDir     string
}

// ListProjects will search for repositories using the given config and options.
//
// By default, it will only search for local repositories. To search for remote
// repositories, set opts.Remote to true.
func ListProjects(ctx context.Context, cfg Config, opts *ListOptions) ([]Project, error) {
	cfg = cfg.setDefaults()

	if opts == nil {
		opts = &ListOptions{}
	}

	opts.CacheDir = fcache.NormalizeCacheDir(opts.CacheDir)

	remoteProjects, err := listRemoteProjects(ctx, cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing remote projects: %w", err)
	}

	localProjects, err := listLocalProjects(ctx, cfg, opts)
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
