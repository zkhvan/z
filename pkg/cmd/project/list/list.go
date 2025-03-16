package list

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/project"
)

type Options struct {
	io     *iolib.IOStreams
	config cmdutil.Config

	FullPath bool
	NoCache  bool
	CacheDir string
}

func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:     f.IOStreams,
		config: f.Config,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long: heredoc.Doc(`
			List the projects defined in the config file.

			Local projects are found by searching for '.git' directories.
			Remote projects are found by searching for repositories on GitHub.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().BoolVar(&opts.FullPath, "full-path", false, "Output the full path")
	cmd.Flags().BoolVar(&opts.NoCache, "no-cache", false, "Do not use the cache")
	cmd.Flags().StringVar(&opts.CacheDir, "cache-dir", "", heredoc.Doc(`
		The directory to cache the list of projects. By default, the cache
		will be saved in $XDG_CACHE_DIR/z or ~/.cache/z/
	`))

	return cmd
}

func (opts *Options) Run(ctx context.Context) error {
	var cfg project.Config
	if err := opts.config.Unmarshal("projects", &cfg); err != nil {
		return err
	}

	results, err := project.ListProjects(ctx, cfg, &project.ListOptions{
		Remote:   true,
		NoCache:  opts.NoCache,
		CacheDir: opts.CacheDir,
	})
	if err != nil {
		return err
	}

	for _, result := range results {
		path := result.ID

		if opts.FullPath {
			path = result.AbsolutePath
		}

		fmt.Fprintln(opts.io.Out, path)
	}

	return nil
}
