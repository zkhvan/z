package project

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/zkhvan/z/pkg/fcache"
	"github.com/zkhvan/z/pkg/gh"
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
	RemoteID     string      `json:"remote_id"`
}

// URL returns the URL of the project.
//
// This is a quick way to determine the URL based on the fact that all there's
// an assumption that all projects are GitHub repositories.
//
// TODO: Detect the proper URL in a generic way, based on the project type.
func (p Project) URL() string {
	id := p.RemoteID
	if id == "" {
		id = p.ID
	}

	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return ""
	}

	owner := parts[0]
	repo := parts[1]

	return fmt.Sprintf("https://github.com/%s/%s", owner, repo)
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

// CloneProject clones a project.
func CloneProject(ctx context.Context, project Project) error {
	url := project.URL()
	if url == "" {
		return fmt.Errorf("error getting project URL")
	}

	// Check if absolute path exists
	if _, err := os.Stat(project.AbsolutePath); err == nil {
		// TODO: confirm with the user what to do in this scenario.
		return fmt.Errorf("project already exists: %s", project.AbsolutePath)
	}

	if _, err := gh.Clone(ctx, url, project.AbsolutePath); err != nil {
		return fmt.Errorf("error cloning project: %w", err)
	}

	return nil
}
