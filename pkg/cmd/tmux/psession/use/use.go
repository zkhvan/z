package use

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/fzf"
	"github.com/zkhvan/z/pkg/project"
	"github.com/zkhvan/z/pkg/tmux"
)

type Options struct {
	config cmdutil.Config
}

func NewCmdUse(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		config: f.Config,
	}

	cmd := &cobra.Command{
		Use:   "use",
		Short: "Use a tmux project session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (opts *Options) Run(ctx context.Context) error {
	service, err := project.NewService(
		opts.config,
		project.WithRefreshCache(true),
	)
	if err != nil {
		return err
	}

	projects, err := service.ListProjects(ctx, &project.ListOptions{
		Local: true,
	})
	if err != nil {
		return err
	}

	proj, err := fzf.One(
		ctx,
		projects,
		projectByPath,
	)
	if errors.Is(err, fzf.ErrCanceled) {
		return nil
	}
	if err != nil {
		return err
	}

	return tmux.NewSession(
		ctx,
		&tmux.NewOptions{
			Name: proj.LocalID,
			Dir:  proj.AbsolutePath,
		},
	)
}

func projectByPath(p project.Project, _ int) string {
	return p.LocalID
}
