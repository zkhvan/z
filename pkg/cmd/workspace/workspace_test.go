package workspace_test

import (
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func TestWorkspace_registers_every_subcommand(t *testing.T) {
	cmd := workspace.NewCmdWorkspace(&cmdutil.Factory{})

	registered := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		registered[sub.Name()] = true
	}

	for _, name := range []string{"archive", "checkout", "create", "delete", "list", "materialize"} {
		if !registered[name] {
			t.Errorf("subcommand %q is not registered", name)
		}
	}
}
