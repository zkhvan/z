package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (c *Client) BranchExists(ctx context.Context, repoPath, branch string) (bool, error) {
	cmd := c.executor.CommandContext(ctx, "git", "-C", repoPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	if err := cmd.Run(); err != nil {
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("checking branch %q in %q: %w", branch, repoPath, err)
	}
	return true, nil
}

// RefExists reports whether any ref has name as its tail: a local branch, a
// remote-tracking branch, or a tag. That is the set `git switch` can resolve,
// including the remote-tracking DWIM.
func (c *Client) RefExists(ctx context.Context, repoPath, name string) (bool, error) {
	cmd := c.executor.CommandContext(ctx, "git", "-C", repoPath, "show-ref", "--quiet", name)
	if err := cmd.Run(); err != nil {
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("checking ref %q in %q: %w", name, repoPath, err)
	}
	return true, nil
}

func (c *Client) DefaultBranch(ctx context.Context, repoPath string) (string, error) {
	ref, err := c.remoteHead(ctx, repoPath)
	if err != nil {
		cmd := c.executor.CommandContext(ctx, "git", "-C", repoPath, "remote", "set-head", "origin", "--auto")
		if setHeadErr := cmd.Run(); setHeadErr != nil {
			return "", fmt.Errorf("determine default branch for %s: %w", repoPath, setHeadErr)
		}
		ref, err = c.remoteHead(ctx, repoPath)
		if err != nil {
			return "", fmt.Errorf("determine default branch for %s: %w", repoPath, err)
		}
	}

	branch, ok := strings.CutPrefix(ref, "refs/remotes/")
	if !ok || branch == "" {
		return "", fmt.Errorf("determine default branch for %s: unexpected ref %q", repoPath, ref)
	}
	return branch, nil
}

func (c *Client) remoteHead(ctx context.Context, repoPath string) (string, error) {
	cmd := c.executor.CommandContext(ctx, "git", "-C", repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}
	return strings.TrimSpace(string(output)), nil
}
