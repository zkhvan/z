package list

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/project"
)

type Options struct {
	io     *iolib.IOStreams
	config cmdutil.Config

	FullPath bool
}

func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:     f.IOStreams,
		config: f.Config,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long:  `List the projects by searching for '.git' directories.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().BoolVar(&opts.FullPath, "full-path", false, "Output the full path")

	return cmd
}

func (opts *Options) Run(ctx context.Context) error {
	var cfg project.Config
	if err := opts.config.Unmarshal("projects", &cfg); err != nil {
		return err
	}

	// Only search for local projects
	cfg.RemotePatterns = nil

	results, err := project.ListProjects(ctx, cfg)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.Type == project.Local {
			path := result.Path

			if opts.FullPath {
				path = result.AbsolutePath
			}

			fmt.Fprintln(opts.io.Out, path)
		}
	}

	return nil
}
