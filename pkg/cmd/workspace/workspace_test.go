package workspace_test

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmd/workspace"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func TestWorkspace_registers_every_subcommand(t *testing.T) {
	cmd := workspace.NewCmdWorkspace(&cmdutil.Factory{})

	registered := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		registered[sub.Name()] = true
	}

	want := []string{
		"archive", "checkout", "create", "definition",
		"delete", "list", "materialize", "sync",
	}
	for _, name := range want {
		if !registered[name] {
			t.Errorf("subcommand %q is not registered", name)
		}
	}
}

func TestWorkspaceDefinition_registers_every_subcommand(t *testing.T) {
	cmd := workspace.NewCmdWorkspace(&cmdutil.Factory{})

	var definition *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "definition" {
			definition = sub
		}
	}
	if definition == nil {
		t.Fatal("the definition group is not registered")
	}

	registered := make(map[string]bool)
	for _, sub := range definition.Commands() {
		registered[sub.Name()] = true
	}

	for _, name := range []string{"init", "list"} {
		if !registered[name] {
			t.Errorf("subcommand %q is not registered under definition", name)
		}
	}
}
