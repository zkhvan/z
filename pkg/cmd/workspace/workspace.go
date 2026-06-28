package workspace

import (
	"github.com/spf13/cobra"

	createCmd "github.com/zkhvan/z/pkg/cmd/workspace/create"
	listCmd "github.com/zkhvan/z/pkg/cmd/workspace/list"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdWorkspace(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage workspaces",
	}

	cmd.AddCommand(createCmd.NewCmdCreate(f))
	cmd.AddCommand(listCmd.NewCmdList(f))

	return cmd
}
