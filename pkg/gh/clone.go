package gh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
)

func (c *Client) Clone(ctx context.Context, url, path string) (string, error) {
	if url == "" {
		return "", errors.New("url is required")
	}

	if path == "" {
		return "", errors.New("path is required")
	}

	cmd := c.executor.CommandContext(
		ctx,
		"gh", "repo", "clone",
		url, path,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}
	output = bytes.TrimSpace(output)

	return string(output), nil
}
