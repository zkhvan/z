package kill

import (
	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/tmux"
)

func NewCmdKill(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kill",
		Short: "Kill the current tmux session and switch to the last active one.",
		RunE: func(cmd *cobra.Command, args []string) error {
			currentSessionID, err := tmux.CurrentSessionID(cmd.Context())
			if err != nil {
				return err
			}

			if err := tmux.SwitchClientLast(cmd.Context()); err != nil {
				return err
			}

			if err := tmux.KillSession(cmd.Context(), tmux.Session{ID: currentSessionID}); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
