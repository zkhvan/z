package up

import (
	"context"

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

func NewCmdUp(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:       f.IOStreams,
		config:   f.Config,
		executor: exec.New(),
	}

	cmd := &cobra.Command{
		Use:   "up [<instance>]",
		Short: "Bring a workspace's runtime up",
		Long: heredoc.Doc(`
			Invoke the instance's runtime/up script, which brings its container
			or VM up.

			The runtime script runs with cwd at the instance directory and with
			Z_INSTANCE_PATH, Z_INSTANCE_NAME, and Z_DEFINITION_PATH in the
			environment. Its exit code becomes z's. The name defaults to the
			workspace containing the current directory.
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
		workspace.WithIOStreams(opts.io),
	)
	if err != nil {
		return err
	}

	name, err := wsloc.Name(svc, opts.Name)
	if err != nil {
		return err
	}

	code, err := svc.Runtime(ctx, name, workspace.RuntimeUp, nil)
	if err != nil {
		return err
	}
	if code != 0 {
		return cmdutil.ExitCodeError{Code: code}
	}
	return nil
}
