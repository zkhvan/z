package materialize

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wsloc"
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

func NewCmdMaterialize(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:       f.IOStreams,
		config:   f.Config,
		executor: exec.New(),
	}

	cmd := &cobra.Command{
		Use:   "materialize [<name>]",
		Short: "Materialize a workspace instance",
		Long: heredoc.Doc(`
			Create git worktrees for every member in a workspace instance.

			The name defaults to the workspace containing the current directory.
		`),
		Args: cobra.MaximumNArgs(1),
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
	if len(args) > 0 {
		opts.Name = args[0]
	}
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

	name, err := wsloc.Name(svc, opts.Name)
	if err != nil {
		return err
	}

	if err := svc.Materialize(ctx, name); err != nil {
		return err
	}

	fmt.Fprintf(opts.io.Out, "Materialized workspace %q\n", name)
	return nil
}
