package list

import (
	"context"
	"fmt"

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

	FullPath     bool
	RefreshCache bool
	Remote       bool
	Local        bool
}

func NewCmdList(f *cmdutil.Factory, projectOpts *internal.ProjectOptions) *cobra.Command {
	opts := &Options{
		ProjectOptions: projectOpts,
		io:             f.IOStreams,
		config:         f.Config,
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
			if err := opts.Complete(cmd, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().BoolVar(&opts.FullPath, "full-path", false, "Output the full path")
	cmd.Flags().BoolVar(&opts.RefreshCache, "refresh-cache", false, "Refresh the cache")
	cmd.Flags().BoolVar(&opts.Remote, "remote", true, "List remote projects")
	cmd.Flags().BoolVar(&opts.Local, "local", true, "List local projects")

	return cmd
}

func (opts *Options) Complete(cmd *cobra.Command, args []string) error {
	remoteChanged := cmd.Flags().Changed("remote")
	localChanged := cmd.Flags().Changed("local")

	if remoteChanged || localChanged {
		// Since the user has specified a type filter, reset the default values to false.
		if !remoteChanged {
			opts.Remote = false
		}

		if !localChanged {
			opts.Local = false
		}
	}

	return nil
}

func (opts *Options) Run(ctx context.Context) error {
	service, err := project.NewService(
		opts.config,
		project.WithRefreshCache(opts.RefreshCache),
		project.WithCacheDir(opts.CacheDir),
	)
	if err != nil {
		return err
	}

	results, err := service.ListProjects(ctx, &project.ListOptions{
		Local:  opts.Local,
		Remote: opts.Remote,
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
