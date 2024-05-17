package session

import (
	"github.com/spf13/cobra"

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

	cmd.AddCommand(listCmd.NewCmdList(f))
	cmd.AddCommand(useCmd.NewCmdUse(f))
	cmd.AddCommand(newCmd.NewCmdNew(f))

	return cmd
}
