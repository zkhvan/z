package refresh

import (
	"context"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/project"
)

type Options struct {
	io     *iolib.IOStreams
	config cmdutil.Config

	CacheDir string
}

func NewCmdRefresh(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:     f.IOStreams,
		config: f.Config,
	}

	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the project cache",
		Long: heredoc.Doc(`
			Refresh the cache of projects defined in the config file.

			This command will force a refresh of the remote projects cache.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd.Context())
		},
	}

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

	_, err := project.ListProjects(ctx, cfg, &project.ListOptions{
		Remote:       true,
		RefreshCache: true,
		CacheDir:     opts.CacheDir,
	})
	if err != nil {
		return err
	}

	return nil
}
