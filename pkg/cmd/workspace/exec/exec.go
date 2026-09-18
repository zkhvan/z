package exec

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wsloc"
	"github.com/zkhvan/z/pkg/cmdutil"
	execlib "github.com/zkhvan/z/pkg/exec"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/workspace"
)

type Options struct {
	io       *iolib.IOStreams
	config   cmdutil.Config
	executor execlib.Interface

	Name    string
	Command []string
}

func NewCmdExec(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:       f.IOStreams,
		config:   f.Config,
		executor: execlib.New(),
	}

	cmd := &cobra.Command{
		Use:   "exec [<instance>] -- <command>...",
		Short: "Run a command inside a workspace's runtime",
		Long: heredoc.Doc(`
			Invoke the instance's runtime/exec script, passing everything after
			-- through as the command to run inside the container or VM. stdin,
			stdout, stderr, and the command's exit code all pass through, so an
			interactive shell (-- bash) works.

			The -- is required; the instance defaults to the workspace
			containing the current directory:

			    z workspace exec my-ws -- npm test
			    z workspace exec -- bash
		`),
		// Trailing command tokens are arbitrary; the -- position is read via
		// ArgsLenAtDash in Complete rather than constrained here.
		Args: cobra.ArbitraryArgs,
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
	dash := cmd.ArgsLenAtDash()
	if dash < 0 {
		return fmt.Errorf("exec requires a command after --, as in: z workspace exec <instance> -- bash")
	}

	before, command := args[:dash], args[dash:]
	if len(before) > 1 {
		return fmt.Errorf("unexpected arguments before --: %v (want at most one instance name)", before[1:])
	}
	if len(command) == 0 {
		return fmt.Errorf("no command given after --")
	}
	if len(before) == 1 {
		opts.Name = before[0]
	}
	opts.Command = command
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

	code, err := svc.Runtime(ctx, name, workspace.RuntimeExec, opts.Command)
	if err != nil {
		return err
	}
	if code != 0 {
		return cmdutil.ExitCodeError{Code: code}
	}
	return nil
}
