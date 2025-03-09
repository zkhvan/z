package list

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/project"
)

func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	var (
		fullPath bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long:  `List the projects by searching for '.git' directories.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg project.Config
			if err := f.Config.Unmarshal("projects", &cfg); err != nil {
				return err
			}

			results, err := project.ListProjects(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			for _, result := range results {
				path := result.Path

				if fullPath {
					path = filepath.Join(os.ExpandEnv("$HOME/Projects"), path)
				}

				fmt.Fprintln(f.IOStreams.Out, path)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&fullPath, "full-path", false, "Output the full path")

	return cmd
}
