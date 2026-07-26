package definition

import (
	"github.com/spf13/cobra"

	initCmd "github.com/zkhvan/z/pkg/cmd/workspace/definition/init"
	listCmd "github.com/zkhvan/z/pkg/cmd/workspace/definition/list"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdDefinition(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "definition",
		Short: "Manage workspace definitions",
	}

	cmd.AddCommand(initCmd.NewCmdInit(f))
	cmd.AddCommand(listCmd.NewCmdList(f))

	return cmd
}
