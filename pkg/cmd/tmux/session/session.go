package session

import (
	"github.com/spf13/cobra"

	killCmd "github.com/zkhvan/z/pkg/cmd/tmux/session/kill"
	listCmd "github.com/zkhvan/z/pkg/cmd/tmux/session/list"
	newCmd "github.com/zkhvan/z/pkg/cmd/tmux/session/new"
	useCmd "github.com/zkhvan/z/pkg/cmd/tmux/session/use"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdSession(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage tmux sessions",
	}

	cmd.AddCommand(killCmd.NewCmdKill(f))
	cmd.AddCommand(listCmd.NewCmdList(f))
	cmd.AddCommand(newCmd.NewCmdNew(f))
	cmd.AddCommand(useCmd.NewCmdUse(f))

	return cmd
}
