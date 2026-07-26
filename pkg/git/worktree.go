package git

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type Worktree struct {
	Path   string
	Branch string
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

	cmd := c.executor.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		output = bytes.TrimSpace(output)
		if len(output) > 0 {
			return fmt.Errorf("error running command %q: %w: %s", cmd.String(), err, output)
		}
		return fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}
	return nil
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
		}
	}
	flush()

	return worktrees
}
