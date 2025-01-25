package kill

import (
	"sort"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/tmux"
)

func NewCmdKill(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kill",
		Short: "Kill the current tmux session, automatically switch to another session if possible.",
		Long: heredoc.Doc(`
			Kill the current tmux session, automatically switch to another session if possible.

			When you kill a session, it will do the following in order:

			1. Automatically switch to the last active session (if available)
			2. Automatically switch to the first session alphabetically (if available)
			3. If no other sessions are available, kill the current session.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			currentSessionID, err := tmux.CurrentSessionID(cmd.Context())
			if err != nil {
				return err
			}
			// If we successfully switch to another session, kill the current one
			defer func() {
				if err == nil {
					err = tmux.KillSession(cmd.Context(), tmux.Session{ID: currentSessionID})
				}
			}()

			// Try to switch to last active session
			if err := tmux.SwitchClientLast(cmd.Context()); err == nil {
				return nil
			}

			// Get all sessions except current one
			sessions, err := tmux.ListSessions(cmd.Context(), &tmux.ListOptions{
				ExcludeCurrentSession: true,
			})
			if err != nil {
				return err
			}

			// If no other sessions available, just kill the current one
			if len(sessions) == 0 {
				return nil
			}

			// Sort sessions by name and switch to first one
			sort.Slice(sessions, func(i, j int) bool {
				return sessions[i].Name < sessions[j].Name
			})
			if err := tmux.SwitchClient(cmd.Context(), sessions[0]); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
