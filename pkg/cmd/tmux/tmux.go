package tmux

import (
	"github.com/spf13/cobra"

	sessionCmd "github.com/zkhvan/z/pkg/cmd/tmux/session"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdTmux(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tmux",
		Short: "Manage tmux",
	}

	cmd.AddCommand(sessionCmd.NewCmdSession(f))

	return cmd
}
