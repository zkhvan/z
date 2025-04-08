package gh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
)

type RepoViewOptions struct {
	WorkingDirectory string
	RepositoryID     string
	Web              bool
}

func (c *Client) RepoView(ctx context.Context, opts *RepoViewOptions) (string, error) {
	if opts == nil {
		opts = &RepoViewOptions{}
	}

	if opts.RepositoryID == "" && opts.WorkingDirectory == "" {
		// At least one is required in order to determine the repository
		return "", errors.New("repository ID or working directory is required")
	}

	args := []string{"repo", "view", opts.RepositoryID}
	if opts.Web {
		args = append(args, "--web")
	}

	cmd := c.executor.CommandContext(
		ctx,
		"gh",
		args...,
	)

	if opts.WorkingDirectory != "" {
		cmd.SetDir(opts.WorkingDirectory)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}
	output = bytes.TrimSpace(output)

	return string(output), nil
}
