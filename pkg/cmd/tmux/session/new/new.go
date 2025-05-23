package new

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/tmux"
)

type Options struct {
	Name string
	Dir  string
}

//nolint:revive
func NewCmdNew(_ *cmdutil.Factory) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:   "new",
		Short: "New tmux session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "The session name")
	cmd.Flags().StringVar(&opts.Dir, "dir", "", "The start directory")

	if err := cmdutil.MarkFlagsRequired(cmd, "name", "dir"); err != nil {
		panic(err)
	}

	return cmd
}

func (opts *Options) Run(ctx context.Context) error {
	return tmux.NewSession(
		ctx,
		&tmux.NewOptions{
			Name: opts.Name,
			Dir:  opts.Dir,
		},
	)
}
