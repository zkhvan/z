package use

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/fzf"
	"github.com/zkhvan/z/pkg/project"
	"github.com/zkhvan/z/pkg/tmux"
)

func NewCmdUse(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use",
		Short: "Use a tmux project session",
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg project.Config
			if err := f.Config.Unmarshal("projects", &cfg); err != nil {
				return err
			}

			projects, err := project.ListProjects(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			project, err := fzf.One(
				cmd.Context(),
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
				cmd.Context(),
				&tmux.NewOptions{
					Name: project.Path,
					Dir:  project.AbsolutePath,
				},
			); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func projectByPath(p project.Project, _ int) string {
	return p.Path
}
