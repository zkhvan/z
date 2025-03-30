package project

import (
	"context"
	"fmt"
	"os"
)

// CloneProject clones a project.
func (s *Service) CloneProject(ctx context.Context, project Project) (string, error) {
	url := project.URL()
	if url == "" {
		return "", fmt.Errorf("error getting project URL")
	}

	// Check if absolute path exists
	if _, err := os.Stat(project.AbsolutePath); err == nil {
		// TODO: confirm with the user what to do in this scenario.
		return "", fmt.Errorf("project already exists: %s", project.AbsolutePath)
	}

	output, err := s.gh.Clone(ctx, url, project.AbsolutePath)
	if err != nil {
		return "", fmt.Errorf("error cloning project: %w", err)
	}

	return output, nil
}
