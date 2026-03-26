package popup

import (
	"github.com/spf13/cobra"

	killCmd "github.com/zkhvan/z/pkg/cmd/tmux/popup/kill"
	listCmd "github.com/zkhvan/z/pkg/cmd/tmux/popup/list"
	useCmd "github.com/zkhvan/z/pkg/cmd/tmux/popup/use"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdPopup(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "popup",
		Short: "Manage tmux popup sessions",
	}

	cmd.AddCommand(killCmd.NewCmdKill(f))
	cmd.AddCommand(listCmd.NewCmdList(f))
	cmd.AddCommand(useCmd.NewCmdUse(f))

	return cmd
}
