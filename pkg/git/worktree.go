package git

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type Worktree struct {
	Path   string
	Branch string
	// Prunable means the gitdir points to a non-existent location right now —
	// a deleted worktree, but also one on an unmounted volume, so it must never
	// drive automatic pruning.
	Prunable bool
}

type WorktreeAddOptions struct {
	RepoPath     string
	WorktreePath string
	Branch       string
	BaseRef      string
	CreateBranch bool
}

func (c *Client) WorktreeAdd(ctx context.Context, opts WorktreeAddOptions) error {
	args := []string{"-C", opts.RepoPath, "worktree", "add"}
	if opts.CreateBranch {
		args = append(args, "-b", opts.Branch, "--", opts.WorktreePath, opts.BaseRef)
	} else {
		args = append(args, "--", opts.WorktreePath, opts.Branch)
	}

	return runCombined(c.executor.CommandContext(ctx, "git", args...))
}

type WorktreeRemoveOptions struct {
	RepoPath     string
	WorktreePath string
	Force        bool
}

// WorktreeRemove deregisters a worktree in the canonical repo. It succeeds even
// when the worktree directory has already been deleted, which is what makes
// teardown idempotent without pruning.
func (c *Client) WorktreeRemove(ctx context.Context, opts WorktreeRemoveOptions) error {
	args := []string{"-C", opts.RepoPath, "worktree", "remove"}
	if opts.Force {
		args = append(args, "--force")
	}
	args = append(args, "--", opts.WorktreePath)

	return runCombined(c.executor.CommandContext(ctx, "git", args...))
}

func (c *Client) WorktreeList(ctx context.Context, repoPath string) ([]Worktree, error) {
	cmd := c.executor.CommandContext(ctx, "git", "-C", repoPath, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}
	return parseWorktreeList(string(output)), nil
}

func parseWorktreeList(output string) []Worktree {
	var worktrees []Worktree
	var current *Worktree
	flush := func() {
		if current != nil && current.Path != "" {
			worktrees = append(worktrees, *current)
		}
		current = nil
	}

	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			flush()
			continue
		}
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			flush()
			current = &Worktree{Path: filepath.Clean(path)}
			continue
		}
		if branch, ok := strings.CutPrefix(line, "branch "); ok && current != nil {
			current.Branch = strings.TrimPrefix(branch, "refs/heads/")
			continue
		}
		if strings.HasPrefix(line, "prunable") && current != nil {
			current.Prunable = true
		}
	}
	flush()

	return worktrees
}
