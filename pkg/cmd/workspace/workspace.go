package workspace

import (
	"github.com/spf13/cobra"

	archiveCmd "github.com/zkhvan/z/pkg/cmd/workspace/archive"
	createCmd "github.com/zkhvan/z/pkg/cmd/workspace/create"
	deleteCmd "github.com/zkhvan/z/pkg/cmd/workspace/delete"
	listCmd "github.com/zkhvan/z/pkg/cmd/workspace/list"
	materializeCmd "github.com/zkhvan/z/pkg/cmd/workspace/materialize"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdWorkspace(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage workspaces",
	}

	cmd.AddCommand(archiveCmd.NewCmdArchive(f))
	cmd.AddCommand(createCmd.NewCmdCreate(f))
	cmd.AddCommand(deleteCmd.NewCmdDelete(f))
	cmd.AddCommand(listCmd.NewCmdList(f))
	cmd.AddCommand(materializeCmd.NewCmdMaterialize(f))

	return cmd
}
