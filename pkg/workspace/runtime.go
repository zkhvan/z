package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const runtimeDirName = "runtime"

// RuntimeVerb names one of the four runtime-contract executables a definition
// may ship under runtime/. The value doubles as the script's filename and, like
// HookPhase, is the entire coupling between z and whatever runs underneath — a
// runtime is anything that answers to up, exec, down, and status.
type RuntimeVerb string

const (
	RuntimeUp     RuntimeVerb = "up"
	RuntimeExec   RuntimeVerb = "exec"
	RuntimeDown   RuntimeVerb = "down"
	RuntimeStatus RuntimeVerb = "status"
)

// Runtime invokes the instance's runtime/<verb> script with cwd at the instance
// directory and Z_INSTANCE_PATH / Z_INSTANCE_NAME (always) plus Z_DEFINITION_PATH
// (best-effort) in the environment. args are passed through as positional
// arguments; real stdin is attached so an interactive exec works.
//
// Runtime scripts live in the instance copy, synced from the definition, so
// resolution never touches the definition — unlike hooks, which run from the
// definition itself. The returned int is the script's exit code (0 when it did
// not run). A script that ran and exited non-zero is (code, nil), not an error:
// a status script legitimately exits 3 for "not running", and an exec'd command
// carries its own code. The error is reserved for z-level failures — an unknown
// workspace, no runtime configured, a non-executable script, or a start failure.
func (s *Service) Runtime(_ context.Context, name string, verb RuntimeVerb, args []string) (int, error) {
	if err := ValidateName(name); err != nil {
		return 0, err
	}

	instanceDir := filepath.Join(s.cfg.Root, name)
	mf, err := readManifest(instanceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("workspace %q has not been created at %s", name, instanceDir)
		}
		return 0, err
	}

	path := filepath.Join(instanceDir, runtimeDirName, string(verb))
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf(
				"no runtime configured for workspace %q: expected an executable script at %s", name, path)
		}
		return 0, fmt.Errorf("checking %s runtime script: %w", verb, err)
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf(
			"no runtime configured for workspace %q: %s is not a regular file", name, path)
	}
	if info.Mode()&0o111 == 0 {
		return 0, fmt.Errorf("runtime %s script is not executable; run: chmod +x %s", verb, path)
	}

	env := append(os.Environ(),
		"Z_INSTANCE_PATH="+instanceDir,
		"Z_INSTANCE_NAME="+name,
	)
	if defDir := s.runtimeDefinitionDir(mf.Definition); defDir != "" {
		env = append(env, "Z_DEFINITION_PATH="+defDir)
	}

	err = s.runScript(path, instanceDir, env, s.io.In, args...)
	if err == nil {
		return 0, nil
	}
	// A script that ran and exited non-zero is data, not a z failure: hand its
	// code back so the caller can surface it as z's own exit code.
	var coder interface{ ExitCode() int }
	if errors.As(err, &coder) {
		return coder.ExitCode(), nil
	}
	return 0, fmt.Errorf("running %s runtime script: %w", verb, err)
}

// runtimeDefinitionDir resolves the definition directory for Z_DEFINITION_PATH.
// It is a convenience for scripts that reach back to their definition, never a
// requirement: the runtime scripts are synced into the instance and run without
// it. So an unset, missing, or broken definition yields "" and stays quiet —
// loudness is reserved for a genuinely missing runtime, not a missing extra.
func (s *Service) runtimeDefinitionDir(definition string) string {
	if definition == "" {
		return ""
	}
	def, err := s.Definition(definition)
	if err != nil || def.Broken() {
		return ""
	}
	return def.Dir
}

// runtimeTemplate scaffolds a working runtime into a definition directory.
type runtimeTemplate func(defDir string) error

// runtimeTemplates maps a --runtime name to its scaffolder. OrbStack is the only
// entry today; the map is the seam for the docker, compose, podman, and Apple
// Container templates the PRD foresees, added without touching any caller.
var runtimeTemplates = map[string]runtimeTemplate{
	"orbstack": writeOrbstackRuntime,
}

// runtimeScaffold looks up the template for name. The empty name scaffolds no
// runtime (a definition may legitimately be runtime-less); an unknown name is
// refused with the supported set rather than silently doing nothing.
func runtimeScaffold(name string) (runtimeTemplate, error) {
	if name == "" {
		return nil, nil
	}
	t, ok := runtimeTemplates[name]
	if !ok {
		return nil, fmt.Errorf("unknown runtime %q; supported: %s", name, supportedRuntimes())
	}
	return t, nil
}

func supportedRuntimes() string {
	names := make([]string, 0, len(runtimeTemplates))
	for n := range runtimeTemplates {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// The OrbStack template maps the four verbs onto isolated machines. up/exec pass
// signals and exit codes straight through by exec'ing orb; down is a reversible
// stop; status reports machine presence with a 0/3 exit code. A pre-delete hook
// removes the machine on `z workspace delete`, since down only stops it.
const (
	orbstackUpScript = `#!/bin/sh
# z runtime (OrbStack): bring the workspace's isolated machine up.
#
# Idempotent: start an existing machine, otherwise create a fresh isolated one
# that mounts the instance directory at /workspace and provisions via cloud-init.
# The machine name is derived from the instance name, so keep instance names
# OrbStack-safe (lowercase letters, digits, hyphens) if you rely on the default.
set -e

name="z-$Z_INSTANCE_NAME"

if orb info "$name" >/dev/null 2>&1; then
	exec orb start "$name"
fi

# --isolate-network blocks other machines and the macOS host while keeping
# internet access. Drop it to reach host services; add --forward-ssh-agent if
# you need to push over SSH from inside the machine.
exec orb create \
	--isolated \
	--isolate-network \
	--mount "$Z_INSTANCE_PATH:/workspace" \
	-c "$Z_INSTANCE_PATH/runtime/user-data.yml" \
	ubuntu "$name"
`

	orbstackExecScript = `#!/bin/sh
# z runtime (OrbStack): run a command inside the workspace's machine, in
# /workspace. stdio and the command's exit code pass straight through.
#
# Isolated machines have no Mac-filesystem mapping, so 'orb run' would land in
# $HOME; the inner 'cd /workspace' puts the command where the mount is. The
# 'sh -c ... sh "$@"' form preserves argument boundaries (so 'exec "$@"' runs
# the command with its args intact, and '-- bash' still attaches a shell). The
# explicit 'run' subcommand keeps a passed command that shares a name with an
# orb subcommand (start, status, list, ...) from being taken for one.
set -e
exec orb run -m "z-$Z_INSTANCE_NAME" -- sh -c 'cd /workspace && exec "$@"' sh "$@"
`

	orbstackDownScript = `#!/bin/sh
# z runtime (OrbStack): stop the workspace's machine (reversible).
# The machine and its disk are preserved, so the next 'up' just starts it again.
# For a full teardown that forces a clean rebuild, replace 'stop' with 'delete'.
set -e
exec orb stop "z-$Z_INSTANCE_NAME"
`

	orbstackStatusScript = `#!/bin/sh
# z runtime (OrbStack): report whether the workspace's machine exists.
# Exit 0 = present, 3 = absent; z surfaces this exit code as its own. 'orb info'
# succeeds for a stopped machine too — refine with 'orb list' if you need to
# tell running from stopped.
name="z-$Z_INSTANCE_NAME"
if orb info "$name" >/dev/null 2>&1; then
	echo "$name: present"
	exit 0
fi
echo "$name: not found"
exit 3
`

	orbstackPreDeleteHook = `#!/bin/sh
# z lifecycle hook (OrbStack runtime): remove the workspace's machine when the
# workspace is deleted, so 'z workspace delete' leaves no orphan. 'down' only
# stops the machine; this deletes it. Best-effort: a missing machine is fine.
orb delete "z-$Z_INSTANCE_NAME" 2>/dev/null || true
`

	orbstackUserData = `#cloud-config
# Cloud-init user data for the workspace's OrbStack machine. Same format as AWS
# EC2 and other clouds; see https://cloudinit.readthedocs.io/. This file is
# synced into every instance and applied on the first 'up'. Edit freely.

packages:
  - git

write_files:
  - path: /etc/z-workspace
    content: |
      provisioned by z runtime (orbstack)
`
)

func writeOrbstackRuntime(defDir string) error {
	runtimeDir := filepath.Join(defDir, runtimeDirName)
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return fmt.Errorf("creating runtime directory: %w", err)
	}

	scripts := map[RuntimeVerb]string{
		RuntimeUp:     orbstackUpScript,
		RuntimeExec:   orbstackExecScript,
		RuntimeDown:   orbstackDownScript,
		RuntimeStatus: orbstackStatusScript,
	}
	for verb, contents := range scripts {
		path := filepath.Join(runtimeDir, string(verb))
		//nolint:gosec // a runtime script must be executable; a non-executable one is an error
		if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
			return fmt.Errorf("writing runtime/%s: %w", verb, err)
		}
	}

	if err := os.WriteFile(filepath.Join(runtimeDir, "user-data.yml"), []byte(orbstackUserData), 0o600); err != nil {
		return fmt.Errorf("writing runtime/user-data.yml: %w", err)
	}

	// The teardown hook lives in the definition's hooks/ dir (host-side, excluded
	// from sync) rather than runtime/, because it fires from the workspace
	// lifecycle, not the runtime contract.
	hooksDir := filepath.Join(defDir, hooksDirName)
	if err := os.MkdirAll(hooksDir, 0o700); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}
	preDelete := filepath.Join(hooksDir, string(HookPreDelete))
	//nolint:gosec // a hook must be executable; a non-executable one is an error
	if err := os.WriteFile(preDelete, []byte(orbstackPreDeleteHook), 0o700); err != nil {
		return fmt.Errorf("writing pre-delete hook: %w", err)
	}

	return nil
}
