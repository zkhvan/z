package git

import (
	"context"
)

type SwitchOptions struct {
	WorktreePath string
	Branch       string
	// BaseRef only applies with CreateBranch. Empty omits the start point, so
	// git branches from the worktree's current HEAD.
	BaseRef      string
	CreateBranch bool
}

// Switch moves a worktree onto another branch in place, keeping untracked files
// (build artifacts) that removing and re-adding the worktree would destroy.
func (c *Client) Switch(ctx context.Context, opts SwitchOptions) error {
	args := []string{"-C", opts.WorktreePath, "switch"}
	if opts.CreateBranch {
		args = append(args, "-c", opts.Branch)
		if opts.BaseRef != "" {
			args = append(args, "--", opts.BaseRef)
		}
	} else {
		args = append(args, "--", opts.Branch)
	}

	return runCombined(c.executor.CommandContext(ctx, "git", args...))
}
