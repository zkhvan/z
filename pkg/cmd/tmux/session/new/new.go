package new

import (
	"github.com/spf13/cobra"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/tmux"
)

func NewCmdNew(f *cmdutil.Factory) *cobra.Command {
	var (
		name = ""
		dir  = ""
	)

	cmd := &cobra.Command{
		Use:   "new",
		Short: "New tmux session",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := tmux.NewSession(
				cmd.Context(),
				&tmux.NewOptions{
					Name: name,
					Dir:  dir,
				},
			); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "The session name")
	cmd.Flags().StringVar(&dir, "dir", "", "The start directory")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("dir")

	return cmd
}
