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
			results, err := project.ListProjects(cmd.Context())
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
