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
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (o *Options) Run(ctx context.Context) error {
	var cfg project.Config
	if err := o.config.Unmarshal("projects", &cfg); err != nil {
		return err
	}

	projects, err := project.ListProjects(ctx, cfg, &project.ListOptions{
		Local:        true,
		RefreshCache: true,
	})
	if err != nil {
		return err
	}

	project, err := fzf.One(
		ctx,
		projects,
		projectByPath,
	)
	if errors.Is(err, fzf.ErrCancelled) {
		return nil
	}
	if err != nil {
		return err
	}

	if err := tmux.NewSession(
		ctx,
		&tmux.NewOptions{
			Name: project.ID,
			Dir:  project.AbsolutePath,
		},
	); err != nil {
		return err
	}

	return nil
}

func projectByPath(p project.Project, _ int) string {
	return p.ID
}
