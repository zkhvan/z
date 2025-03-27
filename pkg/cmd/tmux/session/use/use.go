package use

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/fzf"
	"github.com/zkhvan/z/pkg/tmux"
)

type Options struct{}

func NewCmdUse(_ *cmdutil.Factory) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:   "use",
		Short: "Use a tmux session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (opts *Options) Run(ctx context.Context) error {
	sessions, err := tmux.ListSessions(ctx, &tmux.ListOptions{
		ExcludeCurrentSession: true,
	})
	if err != nil {
		return err
	}

	session, err := fzf.One(ctx, sessions, sessionByName)
	if errors.Is(err, fzf.ErrCanceled) {
		return nil
	}
	if err != nil {
		return err
	}

	return tmux.SwitchClient(ctx, session)
}

func sessionByName(s tmux.Session, _ int) string {
	return s.Name
}
