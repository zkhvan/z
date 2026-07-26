package init

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/exec"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/workspace"
)

type Options struct {
	io       *iolib.IOStreams
	config   cmdutil.Config
	executor exec.Interface

	Name string
}

func NewCmdInit(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:       f.IOStreams,
		config:   f.Config,
		executor: exec.New(),
	}

	cmd := &cobra.Command{
		Use:   "init <name>",
		Short: "Scaffold a workspace definition",
		Long: heredoc.Doc(`
			Create a definition directory under the definitions root, holding a
			starter .z/definition.yaml.

			The scaffolded definition has no members yet, so it cannot be
			instantiated until you add them. Its branch pattern defaults to
			{instance}; the placeholders are {instance} and {repo}.
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

func (opts *Options) Complete(_ *cobra.Command, args []string) error {
	opts.Name = args[0]
	return nil
}

func (opts *Options) Run(ctx context.Context) error {
	svc, err := workspace.NewService(
		opts.config,
		workspace.WithExecutor(opts.executor),
	)
	if err != nil {
		return err
	}

	dir, err := svc.InitDefinition(ctx, opts.Name)
	if err != nil {
		return err
	}

	fmt.Fprintf(opts.io.Out, "Created definition %q at %s\n", opts.Name, dir)
	return nil
}
