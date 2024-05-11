package use

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/fzf"
	"github.com/zkhvan/z/pkg/tmux"
)

func NewCmdUse(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use",
		Short: "Use a tmux session",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessions, err := tmux.ListSessions(cmd.Context(), &tmux.ListOptions{
				ExcludeCurrentSession: true,
			})
			if err != nil {
				return err
			}

			session, err := fzf.One(cmd.Context(), sessions, sessionByName)
			if errors.Is(err, fzf.ErrCancelled) {
				return nil
			}
			if err != nil {
				return err
			}

			return tmux.SwitchClient(cmd.Context(), session)
		},
	}

	return cmd
}

func sessionByName(s tmux.Session, _ int) string {
	return s.Name
}
