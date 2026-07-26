// Package wsloc fills in workspace arguments the shell already knows: run a
// command from inside an instance and its name, and inside a member worktree
// its repo, can be omitted. An explicit argument always wins.
package wsloc

import (
	"fmt"
	"os"

	"github.com/zkhvan/z/pkg/workspace"
)

// Name returns explicit when given, otherwise the workspace containing the
// current directory.
func Name(svc *workspace.Service, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	loc, cwd, err := current(svc)
	if err != nil {
		return "", err
	}
	if loc.Name == "" {
		return "", notInWorkspace(svc, cwd)
	}

	return loc.Name, nil
}

// NameAndMember resolves both arguments from a single look at the current
// directory. A member inferred from a different instance than the requested
// one is ignored, so an explicit name never drags an unrelated repo along.
func NameAndMember(svc *workspace.Service, name, member string) (string, string, error) {
	if name != "" && member != "" {
		return name, member, nil
	}

	loc, cwd, err := current(svc)
	if err != nil {
		return "", "", err
	}

	if name == "" {
		if loc.Name == "" {
			return "", "", notInWorkspace(svc, cwd)
		}
		name = loc.Name
	}
	if member == "" {
		if loc.Name != name || loc.Member == "" {
			return "", "", fmt.Errorf(
				"no repo given and %s is not inside a member worktree of workspace %q", cwd, name)
		}
		member = loc.Member
	}

	return name, member, nil
}

// current returns the zero Location when the current directory is not inside
// an instance; the caller decides whether that is fatal.
func current(svc *workspace.Service) (workspace.Location, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return workspace.Location{}, "", fmt.Errorf("determining the current directory: %w", err)
	}

	loc, ok, err := svc.ResolveDir(cwd)
	if err != nil {
		return workspace.Location{}, cwd, err
	}
	if !ok {
		return workspace.Location{}, cwd, nil
	}

	return loc, cwd, nil
}

func notInWorkspace(svc *workspace.Service, cwd string) error {
	return fmt.Errorf("no workspace name given and %s is not inside a workspace under %s", cwd, svc.Root())
}
