package refresh

import (
	"context"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmd/project/internal"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/project"
)

type Options struct {
	*internal.ProjectOptions
	io     *iolib.IOStreams
	config cmdutil.Config
}

func NewCmdRefresh(f *cmdutil.Factory, projectOpts *internal.ProjectOptions) *cobra.Command {
	opts := &Options{
		ProjectOptions: projectOpts,
		io:             f.IOStreams,
		config:         f.Config,
	}

	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the project cache",
		Long: heredoc.Doc(`
			Refresh the cache of remote projects defined in the config file.

			This command will force a refresh of the remote projects cache.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (opts *Options) Run(ctx context.Context) error {
	var cfg project.Config
	if err := opts.config.Unmarshal("projects", &cfg); err != nil {
		return err
	}

	service := project.NewService(
		cfg,
		project.WithRefreshCache(true),
		project.WithCacheDir(opts.CacheDir))

	_, err := service.ListProjects(ctx, &project.ListOptions{
		Remote: true,
	})
	if err != nil {
		return err
	}

	return nil
}
