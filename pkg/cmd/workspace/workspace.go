package workspace

import (
	"github.com/spf13/cobra"

	archiveCmd "github.com/zkhvan/z/pkg/cmd/workspace/archive"
	checkoutCmd "github.com/zkhvan/z/pkg/cmd/workspace/checkout"
	createCmd "github.com/zkhvan/z/pkg/cmd/workspace/create"
	definitionCmd "github.com/zkhvan/z/pkg/cmd/workspace/definition"
	deleteCmd "github.com/zkhvan/z/pkg/cmd/workspace/delete"
	downCmd "github.com/zkhvan/z/pkg/cmd/workspace/down"
	execCmd "github.com/zkhvan/z/pkg/cmd/workspace/exec"
	listCmd "github.com/zkhvan/z/pkg/cmd/workspace/list"
	materializeCmd "github.com/zkhvan/z/pkg/cmd/workspace/materialize"
	statusCmd "github.com/zkhvan/z/pkg/cmd/workspace/status"
	syncCmd "github.com/zkhvan/z/pkg/cmd/workspace/sync"
	upCmd "github.com/zkhvan/z/pkg/cmd/workspace/up"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdWorkspace(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage workspaces",
	}

	cmd.AddCommand(archiveCmd.NewCmdArchive(f))
	cmd.AddCommand(checkoutCmd.NewCmdCheckout(f))
	cmd.AddCommand(createCmd.NewCmdCreate(f))
	cmd.AddCommand(definitionCmd.NewCmdDefinition(f))
	cmd.AddCommand(deleteCmd.NewCmdDelete(f))
	cmd.AddCommand(downCmd.NewCmdDown(f))
	cmd.AddCommand(execCmd.NewCmdExec(f))
	cmd.AddCommand(listCmd.NewCmdList(f))
	cmd.AddCommand(materializeCmd.NewCmdMaterialize(f))
	cmd.AddCommand(statusCmd.NewCmdStatus(f))
	cmd.AddCommand(syncCmd.NewCmdSync(f))
	cmd.AddCommand(upCmd.NewCmdUp(f))

	return cmd
}
