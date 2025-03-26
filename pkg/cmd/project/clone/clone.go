package clone

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/samber/lo"
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

	ID           string
	RefreshCache bool
}

func NewCmdClone(f *cmdutil.Factory, projectOpts *internal.ProjectOptions) *cobra.Command {
	opts := &Options{
		ProjectOptions: projectOpts,
		io:             f.IOStreams,
		config:         f.Config,
	}

	cmd := &cobra.Command{
		Use:   "clone <id>",
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

	cmd.Flags().BoolVar(&opts.RefreshCache, "refresh-cache", false, "Refresh the cache")

	return cmd
}

func (opts *Options) Complete(cmd *cobra.Command, args []string) error {
	opts.ID = args[0]
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

	// TODO: This is a bit of a hack to get the projects. Should probably
	// refactor this to be more efficient. This method allows proper handling of
	// projects that use an alternate path.
	projects, err := service.ListProjects(ctx, &project.ListOptions{
		Local:  true,
		Remote: true,
	})
	if err != nil {
		return err
	}

	proj, ok := lo.Find(projects, func(p project.Project) bool {
		return p.ID == opts.ID
	})
	if !ok {
		return fmt.Errorf("project not found: %s", opts.ID)
	}

	if err = service.CloneProject(ctx, proj); err != nil {
		return err
	}

	return nil
}
