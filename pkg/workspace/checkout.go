package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitlib "github.com/zkhvan/z/pkg/git"
)

// CheckoutOptions controls branch creation. Create is required to bring a new
// branch into existence; Base is the start point, defaulting to the worktree's
// current HEAD so branches stack instead of flattening onto the trunk.
type CheckoutOptions struct {
	Create bool
	Base   string
}

type CheckoutResult struct {
	Repo     string
	Branch   string
	Previous string
	// Created reports that no local branch of this name existed beforehand,
	// which covers both --create and git's remote-tracking DWIM.
	Created bool
}

// Checkout switches one materialized member onto another branch in place. The
// manifest is updated only after git succeeds, so a failed switch leaves disk
// and manifest in agreement.
func (s *Service) Checkout(
	ctx context.Context,
	name, repo, branch string,
	opts CheckoutOptions,
) (CheckoutResult, error) {
	var result CheckoutResult

	if err := ValidateName(name); err != nil {
		return result, err
	}
	if err := validateBranchName(branch); err != nil {
		return result, err
	}
	if opts.Base != "" && !opts.Create {
		return result, fmt.Errorf("--base requires --create")
	}
	if err := validateBaseRef(opts.Base); err != nil {
		return result, err
	}

	instanceDir := filepath.Join(s.cfg.Root, name)
	mf, err := readManifest(instanceDir)
	if err != nil {
		if os.IsNotExist(err) {
			manifestPath := filepath.Join(instanceDir, manifestRelPath)
			return result, fmt.Errorf("workspace %q has not been created: missing %s", name, manifestPath)
		}
		return result, err
	}

	// Checkout is the only command that rewrites a manifest, and the round-trip
	// drops fields this version does not know about.
	if mf.Version != manifestVersion {
		return result, fmt.Errorf(
			"workspace %q has manifest version %d, but this z understands version %d",
			name, mf.Version, manifestVersion)
	}

	members := make([]Member, len(mf.Members))
	for i, mm := range mf.Members {
		members[i] = memberFromManifest(mm)
	}
	if vErr := ValidateMembers(members); vErr != nil {
		return result, fmt.Errorf("workspace %q manifest is invalid: %w", name, vErr)
	}

	idx, err := findMemberIndex(members, repo)
	if err != nil {
		return result, fmt.Errorf("%w in workspace %q", err, name)
	}
	member := members[idx]

	worktreePath, current, err := s.materializedWorktree(ctx, name, member)
	if err != nil {
		return result, err
	}

	result = CheckoutResult{Repo: member.Repo, Branch: branch, Previous: current}

	// Staying put is a no-op, except under --create, where the branch existing
	// is exactly what the user asserted was not true.
	if current != branch || opts.Create {
		existed, err := s.git.BranchExists(ctx, worktreePath, branch)
		if err != nil {
			return CheckoutResult{}, err
		}

		err = s.git.Switch(ctx, gitlib.SwitchOptions{
			WorktreePath: worktreePath,
			Branch:       branch,
			BaseRef:      opts.Base,
			CreateBranch: opts.Create,
		})
		if err != nil {
			return CheckoutResult{}, s.explainSwitchFailure(ctx, worktreePath, member.Repo, branch, opts, err)
		}
		result.Created = !existed
	}

	if mf.Members[idx].Branch != branch {
		mf.Members[idx].Branch = branch
		if err := writeManifest(instanceDir, mf); err != nil {
			return CheckoutResult{}, err
		}
	}

	return result, nil
}

// explainSwitchFailure names --create when the branch resolves nowhere, which
// is the failure a user hits after a typo or a stale fetch.
func (s *Service) explainSwitchFailure(
	ctx context.Context,
	worktreePath, repo, branch string,
	opts CheckoutOptions,
	cause error,
) error {
	if !opts.Create {
		if exists, err := s.git.RefExists(ctx, worktreePath, branch); err == nil && !exists {
			return fmt.Errorf(
				"branch %q does not exist for %s; pass --create to branch it from the current HEAD: %w",
				branch, repo, cause)
		}
	}
	return fmt.Errorf("switching %s to %q: %w", repo, branch, cause)
}

// materializedWorktree resolves the member's worktree and the branch it is
// currently on, using the same registration-based definition of "materialized"
// as materialize, teardown, and list.
func (s *Service) materializedWorktree(
	ctx context.Context,
	name string,
	m Member,
) (string, string, error) {
	worktreePath := filepath.Join(s.cfg.Root, name, m.BaseName())

	notMaterialized := fmt.Errorf(
		"%s is not materialized at %s; run `z workspace materialize %s` first",
		m.Repo, worktreePath, name)

	proj, err := s.project.Get(ctx, m.Repo)
	if err != nil {
		return "", "", err
	}
	if _, statErr := os.Stat(proj.AbsolutePath); statErr != nil {
		if os.IsNotExist(statErr) {
			return "", "", notMaterialized
		}
		return "", "", fmt.Errorf("checking canonical clone %s: %w", proj.AbsolutePath, statErr)
	}

	worktrees, err := s.git.WorktreeList(ctx, proj.AbsolutePath)
	if err != nil {
		return "", "", err
	}

	registration := findWorktreeByPath(worktrees, worktreePath)
	if registration == nil {
		return "", "", notMaterialized
	}
	if registration.Prunable {
		return "", "", fmt.Errorf(
			"worktree for %s is registered at %s but the directory is missing; "+
				"run `git -C %s worktree prune` and materialize again",
			m.Repo, worktreePath, proj.AbsolutePath)
	}

	return worktreePath, registration.Branch, nil
}

// findMemberIndex matches the base name first, since that is the directory the
// user is looking at, then the full remote ID.
func findMemberIndex(members []Member, repo string) (int, error) {
	for i, m := range members {
		if m.BaseName() == repo {
			return i, nil
		}
	}
	for i, m := range members {
		if m.Repo == repo {
			return i, nil
		}
	}

	names := make([]string, len(members))
	for i, m := range members {
		names[i] = fmt.Sprintf("%s (%s)", m.BaseName(), m.Repo)
	}
	return 0, fmt.Errorf("no member %q, have %s", repo, strings.Join(names, ", "))
}
