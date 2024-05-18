package list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/project"
)

func NewCmdList(f *cmdutil.Factory) *cobra.Command {

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
				fmt.Fprintln(f.IOStreams.Out, result.Path)
			}

			return nil
		},
	}

	return cmd
}
