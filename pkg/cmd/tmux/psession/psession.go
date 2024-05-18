package psession

import (
	"github.com/spf13/cobra"

	useCmd "github.com/zkhvan/z/pkg/cmd/tmux/psession/use"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdPSession(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "psession",
		Short: "Manage project-based sessions",
	}

	cmd.AddCommand(useCmd.NewCmdUse(f))

	return cmd
}
