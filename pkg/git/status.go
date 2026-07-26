package git

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) StatusPorcelain(ctx context.Context, worktreePath string) (string, error) {
	cmd := c.executor.CommandContext(ctx, "git", "-C", worktreePath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}
	return strings.TrimRight(string(output), "\r\n"), nil
}
