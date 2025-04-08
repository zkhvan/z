package gh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
)

type RepoViewOptions struct {
	Web bool
}

func (c *Client) RepoView(ctx context.Context, id string, opts *RepoViewOptions) (string, error) {
	if opts == nil {
		opts = &RepoViewOptions{}
	}

	if id == "" {
		return "", errors.New("id is required")
	}

	args := []string{"repo", "view", id}
	if opts.Web {
		args = append(args, "--web")
	}

	cmd := c.executor.CommandContext(
		ctx,
		"gh",
		args...,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}
	output = bytes.TrimSpace(output)

	return string(output), nil
}
