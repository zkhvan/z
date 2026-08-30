package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	gitlib "github.com/zkhvan/z/pkg/git"
)

const materializedMarkerRelPath = ".z/materialized"

func (s *Service) Materialize(ctx context.Context, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}

	instanceDir := filepath.Join(s.cfg.Root, name)
	mf, err := readManifest(instanceDir)
	if err != nil {
		if os.IsNotExist(err) {
			manifestPath := filepath.Join(instanceDir, manifestRelPath)
			return fmt.Errorf("workspace %q has not been created: missing %s", name, manifestPath)
		}
		return err
	}

	members := make([]Member, len(mf.Members))
	for i, mm := range mf.Members {
		members[i] = memberFromManifest(mm)
	}
	if err := ValidateMembers(members); err != nil {
		return fmt.Errorf("workspace %q manifest is invalid: %w", name, err)
	}

	def := s.hookDefinition(name, mf.Definition)
	if err := s.runHook(def, name, instanceDir, HookPreMaterialize); err != nil {
		return err
	}

	for _, m := range members {
		if err := s.materializeMember(ctx, instanceDir, m); err != nil {
			return fmt.Errorf("materializing %s: %w", m.Repo, err)
		}
	}

	if err := writeMaterializedMarker(instanceDir); err != nil {
		return err
	}
	return s.runHook(def, name, instanceDir, HookPostMaterialize)
}

func (s *Service) materializeMember(ctx context.Context, instanceDir string, m Member) error {
	canonicalPath, err := s.ensureCanonicalClone(ctx, m.Repo)
	if err != nil {
		return err
	}

	worktreePath := filepath.Join(instanceDir, m.BaseName())
	worktrees, err := s.git.WorktreeList(ctx, canonicalPath)
	if err != nil {
		return err
	}

	if _, statErr := os.Stat(worktreePath); statErr == nil {
		wt := findWorktreeByPath(worktrees, worktreePath)
		if wt == nil {
			return fmt.Errorf("%s exists but is not a worktree for %s", worktreePath, m.Repo)
		}
		if wt.Branch != m.Branch {
			return fmt.Errorf("%s is on branch %q, want %q", worktreePath, wt.Branch, m.Branch)
		}
		return nil
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("checking worktree path %s: %w", worktreePath, statErr)
	}

	if conflict := findWorktreeByBranch(worktrees, m.Branch); conflict != nil {
		if conflict.Prunable {
			return fmt.Errorf(
				"branch %q is registered to a missing worktree at %s; "+
					"if that path is gone for good, run `git -C %s worktree prune`",
				m.Branch, conflict.Path, canonicalPath)
		}
		return fmt.Errorf("branch %q is already checked out at %s", m.Branch, conflict.Path)
	}

	exists, err := s.git.BranchExists(ctx, canonicalPath, m.Branch)
	if err != nil {
		return err
	}

	if exists {
		return s.git.WorktreeAdd(ctx, gitlib.WorktreeAddOptions{
			RepoPath:     canonicalPath,
			WorktreePath: worktreePath,
			Branch:       m.Branch,
		})
	}

	baseRef := m.BaseRef
	if baseRef == "" {
		baseRef, err = s.git.DefaultBranch(ctx, canonicalPath)
		if err != nil {
			return err
		}
	}

	return s.git.WorktreeAdd(ctx, gitlib.WorktreeAddOptions{
		RepoPath:     canonicalPath,
		WorktreePath: worktreePath,
		Branch:       m.Branch,
		BaseRef:      baseRef,
		CreateBranch: true,
	})
}

func (s *Service) ensureCanonicalClone(ctx context.Context, repo string) (string, error) {
	proj, err := s.project.Get(ctx, repo)
	if err != nil {
		return "", err
	}

	st, err := os.Stat(proj.AbsolutePath)
	if err == nil {
		if !st.IsDir() {
			return "", fmt.Errorf("canonical clone path %s is not a directory", proj.AbsolutePath)
		}
		return proj.AbsolutePath, nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("checking canonical clone %s: %w", proj.AbsolutePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(proj.AbsolutePath), 0o700); err != nil {
		return "", fmt.Errorf("creating canonical clone parent: %w", err)
	}
	if _, err := s.project.CloneProject(ctx, proj); err != nil {
		return "", err
	}
	return proj.AbsolutePath, nil
}

func findWorktreeByPath(worktrees []gitlib.Worktree, path string) *gitlib.Worktree {
	path = filepath.Clean(path)
	for i := range worktrees {
		if filepath.Clean(worktrees[i].Path) == path {
			return &worktrees[i]
		}
	}
	return nil
}

func findWorktreeByBranch(worktrees []gitlib.Worktree, branch string) *gitlib.Worktree {
	for i := range worktrees {
		if worktrees[i].Branch == branch {
			return &worktrees[i]
		}
	}
	return nil
}

func writeMaterializedMarker(instanceDir string) error {
	path := filepath.Join(instanceDir, materializedMarkerRelPath)
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		return fmt.Errorf("writing materialized marker: %w", err)
	}
	return nil
}

func (s *Service) deriveInstanceStatus(
	ctx context.Context,
	instanceDir string,
	members []InstanceMember,
) InstanceStatus {
	present := 0
	for i := range members {
		proj, err := s.project.Get(ctx, members[i].Repo)
		if err != nil {
			continue
		}
		if _, statErr := os.Stat(proj.AbsolutePath); statErr != nil {
			continue
		}

		worktrees, err := s.git.WorktreeList(ctx, proj.AbsolutePath)
		if err != nil {
			continue
		}
		worktreePath := filepath.Join(instanceDir, members[i].BaseName())
		worktree := findWorktreeByPath(worktrees, worktreePath)
		if worktree == nil || worktree.Branch != members[i].Branch {
			continue
		}

		status, err := s.git.StatusPorcelain(ctx, worktreePath)
		if err != nil {
			continue
		}
		present++
		if status == "" {
			members[i].State = MemberStateClean
		} else {
			members[i].State = MemberStateDirty
		}
	}

	if present == len(members) && len(members) > 0 {
		return InstanceStatusMaterialized
	}
	if present > 0 {
		return InstanceStatusPartial
	}
	if materializedMarkerExists(instanceDir) {
		return InstanceStatusArchived
	}
	return InstanceStatusNew
}

func materializedMarkerExists(instanceDir string) bool {
	_, err := os.Stat(filepath.Join(instanceDir, materializedMarkerRelPath))
	return err == nil
}
