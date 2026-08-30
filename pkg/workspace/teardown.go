package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gitlib "github.com/zkhvan/z/pkg/git"
	"github.com/zkhvan/z/pkg/workspace/wssync"
)

// TeardownOptions applies to both Archive and Delete. Force accepts data loss
// (dirty worktrees, missing canonical clones, unmanaged files); it never bypasses
// a missing or invalid manifest, because those are what make a workspace
// something z can reason about at all.
type TeardownOptions struct {
	Force bool
}

// Archive deregisters every member worktree, leaving the instance directory,
// .z/, and the manifest untouched so it can be re-materialized.
func (s *Service) Archive(ctx context.Context, name string, opts TeardownOptions) error {
	instanceDir, def, err := s.teardown(ctx, name, opts, teardownArchive)
	if err != nil {
		return err
	}
	return s.runHook(def, name, instanceDir, HookPostArchive)
}

// Delete deregisters every member worktree before removing the instance
// directory. The removal happens only after a fully successful teardown, so a
// failure never strands registrations pointing into a directory that is gone.
func (s *Service) Delete(ctx context.Context, name string, opts TeardownOptions) error {
	instanceDir, def, err := s.teardown(ctx, name, opts, teardownDelete)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(instanceDir); err != nil {
		return fmt.Errorf("removing instance directory %s: %w", instanceDir, err)
	}
	return s.runHook(def, name, instanceDir, HookPostDelete)
}

type teardownMode int

const (
	teardownArchive teardownMode = iota
	teardownDelete
)

func (m teardownMode) verb() string {
	if m == teardownDelete {
		return "delete"
	}
	return "archive"
}

// teardownStep is the plan produced by the read-only preflight and consumed by
// the execute phase, so git is queried once per member rather than twice.
type teardownStep struct {
	member     Member
	canonical  string
	worktree   string
	registered bool
	// orphaned means the canonical clone is gone, so nothing can be
	// deregistered and the member directory is unrecoverable.
	orphaned bool
}

func (s *Service) teardown(
	ctx context.Context,
	name string,
	opts TeardownOptions,
	mode teardownMode,
) (string, Definition, error) {
	if err := ValidateName(name); err != nil {
		return "", Definition{}, err
	}

	instanceDir := filepath.Join(s.cfg.Root, name)
	mf, err := readManifest(instanceDir)
	if err != nil {
		if os.IsNotExist(err) {
			manifestPath := filepath.Join(instanceDir, manifestRelPath)
			return "", Definition{}, fmt.Errorf("workspace %q has not been created: missing %s", name, manifestPath)
		}
		return "", Definition{}, err
	}

	members := make([]Member, len(mf.Members))
	for i, mm := range mf.Members {
		members[i] = memberFromManifest(mm)
	}
	if err = ValidateMembers(members); err != nil {
		return "", Definition{}, fmt.Errorf("workspace %q manifest is invalid: %w", name, err)
	}

	plan, problems, err := s.planTeardown(ctx, instanceDir, members, mode)
	if err != nil {
		return "", Definition{}, err
	}
	if !problems.empty() && !opts.Force {
		return "", Definition{}, problems.err(mode.verb(), name)
	}

	// The pre-* hook runs after the problem gate but before any deregistration,
	// so a non-zero exit aborts with every worktree still intact. The post-* hook
	// is the caller's job: archive and delete tear down at different moments.
	def := s.hookDefinition(name, mf.Definition)
	prePhase := HookPreArchive
	if mode == teardownDelete {
		prePhase = HookPreDelete
	}
	if err := s.runHook(def, name, instanceDir, prePhase); err != nil {
		return "", Definition{}, err
	}

	for _, step := range plan {
		if err := s.executeTeardownStep(ctx, step, opts); err != nil {
			return "", Definition{}, err
		}
	}

	return instanceDir, def, nil
}

func (s *Service) planTeardown(
	ctx context.Context,
	instanceDir string,
	members []Member,
	mode teardownMode,
) ([]teardownStep, teardownProblems, error) {
	var problems teardownProblems
	plan := make([]teardownStep, 0, len(members))

	for _, m := range members {
		proj, err := s.project.Get(ctx, m.Repo)
		if err != nil {
			return nil, problems, err
		}

		step := teardownStep{
			member:    m,
			canonical: proj.AbsolutePath,
			worktree:  filepath.Join(instanceDir, m.BaseName()),
		}

		if _, statErr := os.Stat(step.canonical); statErr != nil {
			if !os.IsNotExist(statErr) {
				return nil, problems, fmt.Errorf("checking canonical clone %s: %w", step.canonical, statErr)
			}
			step.orphaned = true
			problems.missingCanonical = append(problems.missingCanonical,
				fmt.Sprintf("%s at %s", m.Repo, step.canonical))
			plan = append(plan, step)
			continue
		}

		worktrees, err := s.git.WorktreeList(ctx, step.canonical)
		if err != nil {
			return nil, problems, err
		}

		registration := findWorktreeByPath(worktrees, step.worktree)
		step.registered = registration != nil
		plan = append(plan, step)

		// A prunable registration has no directory left, so there is nothing to
		// protect and StatusPorcelain would only fail.
		if !step.registered || registration.Prunable {
			continue
		}

		status, err := s.git.StatusPorcelain(ctx, step.worktree)
		if err != nil {
			return nil, problems, err
		}
		if status != "" {
			problems.dirty = append(problems.dirty,
				fmt.Sprintf("%s (%s) at %s", m.Repo, m.Branch, step.worktree))
		}
	}

	if mode == teardownDelete {
		unmanaged, err := s.unmanagedEntries(instanceDir, members)
		if err != nil {
			return nil, problems, err
		}
		problems.unmanaged = unmanaged
	}

	return plan, problems, nil
}

func (s *Service) executeTeardownStep(ctx context.Context, step teardownStep, opts TeardownOptions) error {
	if step.registered {
		err := s.git.WorktreeRemove(ctx, gitlib.WorktreeRemoveOptions{
			RepoPath:     step.canonical,
			WorktreePath: step.worktree,
			Force:        opts.Force,
		})
		if err != nil {
			return fmt.Errorf("removing worktree for %s: %w", step.member.Repo, err)
		}
		return nil
	}

	// Without a canonical clone nothing can deregister the directory, so forcing
	// is the only way to make archive's postcondition true.
	if step.orphaned && opts.Force {
		if err := os.RemoveAll(step.worktree); err != nil {
			return fmt.Errorf("removing orphaned worktree directory %s: %w", step.worktree, err)
		}
	}
	return nil
}

// unmanagedEntries reports instance content that exists nowhere else. The test
// is recoverability, which is what the guard always meant: worktree content is
// held by the canonical clone, a synced file still matching what z wrote is held
// by the definition, and ignored junk is held by nobody who cares. Everything
// else is only here.
func (s *Service) unmanagedEntries(instanceDir string, members []Member) ([]string, error) {
	tree, err := wssync.Scan(instanceDir, s.instanceIgnores(members))
	if err != nil {
		return nil, err
	}

	recorded, err := wssync.LoadState(instanceDir)
	if err != nil {
		return nil, err
	}

	return unrecoverable("", tree, recorded), nil
}

func unrecoverable(path string, current, recorded *wssync.Entry) []string {
	switch {
	case current == nil, current.Kind == wssync.KindUntracked:
		return nil
	case current.Kind == wssync.KindDirectory:
		if len(current.Contents) == 0 && path != "" {
			// Nothing inside to lose, but a hand-made directory is still not
			// z's, so it stays visible rather than being silently removed.
			return []string{path + "/"}
		}
		names := make([]string, 0, len(current.Contents))
		for name := range current.Contents {
			names = append(names, name)
		}
		sort.Strings(names)

		var out []string
		for _, name := range names {
			child := path + name
			if path != "" {
				child = path + "/" + name
			}
			out = append(out, unrecoverable(child, current.Contents[name], recorded)...)
		}
		return out
	case current.Equal(wssync.Lookup(recorded, path), false):
		return nil
	default:
		return []string{path}
	}
}

type teardownProblems struct {
	dirty            []string
	missingCanonical []string
	unmanaged        []string
}

func (p teardownProblems) empty() bool {
	return len(p.dirty) == 0 && len(p.missingCanonical) == 0 && len(p.unmanaged) == 0
}

// err reports every problem at once so the user resolves them in one pass
// instead of re-running to discover the next one.
func (p teardownProblems) err(verb, name string) error {
	lines := []string{fmt.Sprintf("cannot %s workspace %q:", verb, name)}

	section := func(headline string, items []string) {
		if len(items) == 0 {
			return
		}
		lines = append(lines, "  "+headline)
		for _, item := range items {
			lines = append(lines, "    "+item)
		}
	}

	section(pluralize(len(p.dirty), "member", "members")+" with uncommitted or untracked changes:", p.dirty)
	section("members whose canonical clone is missing:", p.missingCanonical)
	section("files not managed by z:", p.unmanaged)
	lines = append(lines, "use --force to proceed anyway")

	return errors.New(strings.Join(lines, "\n"))
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}
