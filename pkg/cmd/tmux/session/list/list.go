package list

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/tmux"
)

func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tmux sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessions, err := tmux.ListSessions(cmd.Context(), nil)
			if err != nil {
				return err
			}

			for _, session := range sessions {
				fmt.Fprintln(f.IOStreams.Out, session.Name)
			}

			return nil
		},
	}

	return cmd
}
