package project

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/zkhvan/z/pkg/gh"
)

type ProjectType string

const (
	Local  ProjectType = "local"
	Remote ProjectType = "remote"
)

type Project struct {
	// Type represents how the project was discovered.
	Type ProjectType `json:"type"`

	// ID is the identifier of the project. It's the relative path to the
	// project from the root directory.
	ID string `json:"id"`

	// AbsolutePath is the absolute path to the project.
	AbsolutePath string `json:"absolute_path"`

	// RemoteID is the identifier of the project on the remote service. This is
	// usually owner/repo for GitHub.
	RemoteID string `json:"remote_id"`
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

// CloneProject clones a project.
func (s *Service) CloneProject(ctx context.Context, project Project) error {
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
