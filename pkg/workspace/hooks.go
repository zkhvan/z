package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

// HookPhase names a lifecycle phase at which a definition may supply a hook. The
// value doubles as the hook's filename under <definition>/hooks/ and as the
// Z_HOOK_PHASE handed to the script.
type HookPhase string

const (
	HookPreCreate       HookPhase = "pre-create"
	HookPostCreate      HookPhase = "post-create"
	HookPreMaterialize  HookPhase = "pre-materialize"
	HookPostMaterialize HookPhase = "post-materialize"
	HookPreArchive      HookPhase = "pre-archive"
	HookPostArchive     HookPhase = "post-archive"
	HookPreDelete       HookPhase = "pre-delete"
	HookPostDelete      HookPhase = "post-delete"
)

const hooksDirName = "hooks"

// hookPhases lists every phase in lifecycle order. Scaffolding walks it.
var hookPhases = []HookPhase{
	HookPreCreate, HookPostCreate,
	HookPreMaterialize, HookPostMaterialize,
	HookPreArchive, HookPostArchive,
	HookPreDelete, HookPostDelete,
}

// hookExampleTemplate is a starter hook written for every phase at definition
// init time under the name "<phase>.example". The suffix keeps it inert (only an
// exact phase name executes), so the whole set documents the surface without
// running. %[1]s is the phase.
const hookExampleTemplate = `#!/bin/sh
# Example z lifecycle hook — a starter that prints the hook environment.
#
# To enable, drop the ".example" suffix so the filename is exactly the phase
# name. It is already executable, and the rename preserves that:
#
#     mv %[1]s.example %[1]s
#
# Phases: pre-create, post-create, pre-materialize, post-materialize,
#         pre-archive, post-archive, pre-delete, post-delete.
#
# A pre-* hook that exits non-zero aborts the operation before any change.
# A post-* hook runs only after the operation succeeds and is never rolled back.
#
# cwd is the instance directory, except pre-create and post-delete, which run
# from the workspaces root because the instance directory does not exist at
# those phases.

echo "[z hook] phase=${Z_HOOK_PHASE}"
echo "[z hook] instance=${Z_INSTANCE_NAME}"
echo "[z hook] instance_path=${Z_INSTANCE_PATH}"
echo "[z hook] definition_path=${Z_DEFINITION_PATH}"
echo "[z hook] cwd=$(pwd)"
`

// writeExampleHooks scaffolds an inert starter for every phase. The 0o700 mode
// means enabling one is a rename, not a rename plus chmod — and a hook that is
// present but not executable is an error, so the bit has to be right up front.
func writeExampleHooks(dir string) error {
	hooksDir := filepath.Join(dir, hooksDirName)
	if err := os.MkdirAll(hooksDir, 0o700); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}
	for _, phase := range hookPhases {
		path := filepath.Join(hooksDir, string(phase)+".example")
		contents := fmt.Sprintf(hookExampleTemplate, phase)
		//nolint:gosec // a hook must be executable; a non-executable one is an error
		if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
			return fmt.Errorf("writing example hook %s: %w", phase, err)
		}
	}
	return nil
}

// hookDefinition resolves the definition named in an instance manifest for the
// sole purpose of running hooks. Hooks are best-effort: an unresolvable or
// broken definition is never a reason to block a lifecycle operation, so it
// warns and returns a zero Definition, which runHook treats as "no hooks".
func (s *Service) hookDefinition(name, definition string) Definition {
	if definition == "" {
		return Definition{}
	}
	def, err := s.Definition(definition)
	if err == nil && def.Broken() {
		err = def.Err
	}
	if err != nil {
		fmt.Fprintf(s.io.ErrOut, "warning: skipping hooks for %q: %v\n", name, err)
		return Definition{}
	}
	return def
}

// runHook runs the definition's hook for phase if one is present. A missing or
// non-regular hook is a no-op; a present-but-non-executable file is an error,
// because it is almost always a forgotten chmod rather than an intent to skip.
//
// cwd is the instance directory, except for the two boundary phases whose
// instance directory does not exist when they run: pre-create (not yet created)
// and post-delete (already removed) run from the workspaces root instead.
// Z_INSTANCE_PATH still names the instance either way.
//
// Abort policy belongs to the caller: a pre-* error must abort before side
// effects; a post-* error is surfaced without rollback.
func (s *Service) runHook(def Definition, name, instancePath string, phase HookPhase) error {
	if def.Dir == "" {
		return nil
	}

	path := filepath.Join(def.Dir, hooksDirName, string(phase))
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking %s hook: %w", phase, err)
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("%s hook is not executable; run: chmod +x %s", phase, path)
	}

	cwd := instancePath
	if phase == HookPreCreate || phase == HookPostDelete {
		cwd = filepath.Dir(instancePath)
	}

	cmd := s.executor.Command(path)
	cmd.SetDir(cwd)
	cmd.SetEnv(append(os.Environ(),
		"Z_INSTANCE_PATH="+instancePath,
		"Z_INSTANCE_NAME="+name,
		"Z_DEFINITION_PATH="+def.Dir,
		"Z_HOOK_PHASE="+string(phase),
	))
	cmd.SetStdout(s.io.Out)
	cmd.SetStderr(s.io.ErrOut)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s hook failed: %w", phase, err)
	}
	return nil
}
