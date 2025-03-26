package clone

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

	ID string
}

func NewCmdClone(f *cmdutil.Factory, projectOpts *internal.ProjectOptions) *cobra.Command {
	opts := &Options{
		ProjectOptions: projectOpts,
		io:             f.IOStreams,
		config:         f.Config,
	}

	cmd := &cobra.Command{
		Use:   "clone <remote-id>",
		Short: "Clone a project",
		Long: heredoc.Doc(`
			Clone a project to the default path.
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(cmd, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (opts *Options) Complete(cmd *cobra.Command, args []string) error {
	opts.ID = args[0]
	return nil
}

func (opts *Options) Run(ctx context.Context) error {
	service, err := project.NewService(
		opts.config,
		project.WithCacheDir(opts.CacheDir),
	)
	if err != nil {
		return err
	}

	project, err := service.GetRemoteProject(ctx, opts.ID)
	if err != nil {
		return err
	}

	output, err := service.CloneProject(ctx, project)
	if err != nil {
		return err
	}
	fmt.Fprintln(opts.io.Out, output)

	return nil
}
