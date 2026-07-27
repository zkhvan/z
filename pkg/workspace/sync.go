package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zkhvan/z/pkg/workspace/wssync"
)

// SyncOptions controls one reconciliation.
type SyncOptions struct {
	// Force selects replica mode: the instance becomes an exact mirror of the
	// definition. It accepts data loss, and like every other --force in this
	// package it never bypasses a precondition — a missing or broken definition
	// still refuses.
	Force bool
	// DryRun computes the plan and writes nothing, including no sync state.
	DryRun bool
}

// SyncReport is what happened, for the command layer to render.
type SyncReport struct {
	Definition string
	Applied    []wssync.Change
	Conflicts  []wssync.Conflict
	Problems   []wssync.Problem
	DryRun     bool
	Replica    bool
}

// Changed reports whether anything was written (or would be).
func (r SyncReport) Changed() bool { return len(r.Applied) > 0 }

// Sync reconciles a definition into an instance. Offline and structural: it
// never touches git, so it works on a new or archived instance alike.
func (s *Service) Sync(_ context.Context, name string, opts SyncOptions) (SyncReport, error) {
	if err := ValidateName(name); err != nil {
		return SyncReport{}, err
	}

	instanceDir := filepath.Join(s.cfg.Root, name)
	mf, err := readManifest(instanceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return SyncReport{}, fmt.Errorf("workspace %q has not been created at %s", name, instanceDir)
		}
		return SyncReport{}, err
	}

	members := make([]Member, len(mf.Members))
	for i, mm := range mf.Members {
		members[i] = memberFromManifest(mm)
	}

	def, err := s.syncDefinition(name, mf.Definition)
	if err != nil {
		return SyncReport{}, err
	}

	return s.sync(instanceDir, def, members, opts)
}

// syncDefinition resolves the instance's definition, refusing rather than
// falling back. An unresolvable definition must never be read as an empty one:
// every recorded file would become a deletion, and a stale definitions_root
// would quietly strip instances of their configuration.
func (s *Service) syncDefinition(name, definition string) (Definition, error) {
	if definition == "" {
		return Definition{}, fmt.Errorf(
			"workspace %q was not created from a definition; there is nothing to sync", name)
	}

	def, err := s.Definition(definition)
	if err != nil {
		return Definition{}, fmt.Errorf("%w (definitions root: %s)", err, s.cfg.DefinitionsRoot)
	}
	if def.Broken() {
		return Definition{}, fmt.Errorf("definition %q is not usable: %w", def.Name, def.Err)
	}
	return def, nil
}

// sync is the shared pipeline. Create calls it with a freshly written manifest
// and no sync state, which is just reconciliation against an absent ancestor.
func (s *Service) sync(instanceDir string, def Definition, members []Member, opts SyncOptions) (SyncReport, error) {
	report := SyncReport{Definition: def.Name, DryRun: opts.DryRun, Replica: opts.Force}

	ancestor, err := wssync.LoadState(instanceDir)
	if err != nil {
		return report, err
	}

	definitionTree, err := wssync.Scan(def.Dir, s.definitionIgnores(def))
	if err != nil {
		return report, err
	}
	definitionTree = wssync.PruneEmptyDirs(definitionTree)
	instanceTree, err := wssync.Scan(instanceDir, s.instanceIgnores(members))
	if err != nil {
		return report, err
	}

	mode := wssync.ModeSafe
	if opts.Force {
		mode = wssync.ModeReplica
	}

	result := wssync.Reconcile(ancestor, definitionTree, instanceTree, mode)
	report.Conflicts = result.Conflicts
	report.Problems = append(definitionTree.Problems(""), instanceTree.Problems("")...)

	if opts.DryRun {
		report.Applied = result.Changes
		return report, nil
	}

	applied, transitionErr := wssync.Transition(instanceDir, def.Dir, result.Changes)
	report.Applied = applied

	// The state is advanced by exactly what landed, even on the error path:
	// discarding that bookkeeping is what would turn a resumable sync into a
	// pile of phantom conflicts.
	if len(applied) > 0 || len(result.AncestorChanges) > 0 {
		next := wssync.Apply(ancestor, append(result.AncestorChanges, applied...))
		if err := wssync.SaveState(instanceDir, next); err != nil {
			return report, errors.Join(transitionErr, err)
		}
	}

	return report, transitionErr
}

// definitionIgnores excludes z's own namespace plus whatever the definition and
// the user declared.
func (s *Service) definitionIgnores(def Definition) wssync.Matcher {
	return wssync.Ignores{
		Structural: []string{".z"},
		Patterns:   append(append([]string{}, def.SyncIgnore...), s.cfg.Sync.Ignore...),
	}
}

// instanceIgnores additionally excludes the member worktrees. These are
// structural and non-overridable: nothing definition-authored lives inside a
// worktree, and in replica mode this list is all that stands between --force
// and a directory full of uncommitted work.
func (s *Service) instanceIgnores(members []Member) wssync.Matcher {
	structural := make([]string, 0, len(members)+1)
	structural = append(structural, ".z")
	for _, m := range members {
		structural = append(structural, m.BaseName())
	}
	return wssync.Ignores{Structural: structural, Patterns: s.cfg.Sync.Ignore}
}

// SyncedPaths reports the recorded files whose instance copy still matches what
// z wrote, and those that have been modified since. Teardown uses the split to
// tell recoverable content from content that exists nowhere else.
func (s *Service) SyncedPaths(instanceDir string) (unchanged, modified []string, err error) {
	ancestor, err := wssync.LoadState(instanceDir)
	if err != nil {
		return nil, nil, err
	}
	if ancestor == nil {
		return nil, nil, nil
	}

	instance, err := wssync.Scan(instanceDir, wssync.NoIgnores())
	if err != nil {
		return nil, nil, err
	}

	for _, path := range wssync.StatePaths(ancestor) {
		// A recorded file that is simply gone is not on disk to be guarded.
		current := wssync.Lookup(instance, path)
		if current == nil {
			continue
		}
		if current.Equal(wssync.Lookup(ancestor, path), false) {
			unchanged = append(unchanged, path)
			continue
		}
		modified = append(modified, path)
	}
	return unchanged, modified, nil
}
