package use

import (
	"errors"
	"os"
	"path/filepath"

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
			projects, err := project.ListProjects(cmd.Context())
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
					Dir:  filepath.Join(os.ExpandEnv("$HOME/Projects"), project.Path),
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
