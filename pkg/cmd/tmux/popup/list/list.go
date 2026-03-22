package list

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/tmux"
)

type Options struct {
	io *iolib.IOStreams
}

func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List popup sessions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (opts *Options) Run(ctx context.Context) error {
	parentName, err := tmux.CurrentSessionName(ctx)
	if err != nil {
		return err
	}

	sessions, err := tmux.ListSessions(ctx, nil)
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if name, ok := tmux.ExtractPopupName(session.Name, parentName); ok {
			fmt.Fprintln(opts.io.Out, name)
		}
	}

	return nil
}
