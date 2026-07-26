package delete

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

	Name  string
	Force bool
}

func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:       f.IOStreams,
		config:   f.Config,
		executor: exec.New(),
	}

	cmd := &cobra.Command{
		Use:   "delete [<name>]",
		Short: "Delete a workspace instance",
		Long: heredoc.Doc(`
			Remove the git worktrees of every member, then remove the instance
			directory.

			Members with uncommitted or untracked changes, or files in the
			instance directory that z does not manage, abort the delete before
			anything is removed; --force removes them anyway.

			The name defaults to the workspace containing the current directory,
			which means the directory the shell is sitting in can be the one that
			goes away.
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(cmd, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().BoolVar(&opts.Force, "force", false, heredoc.Doc(`
		Remove worktrees with uncommitted or untracked changes, and files in the
		instance directory that z does not manage.
	`))

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

	if err := svc.Delete(ctx, name, workspace.TeardownOptions{Force: opts.Force}); err != nil {
		return err
	}

	fmt.Fprintf(opts.io.Out, "Deleted workspace %q\n", name)
	return nil
}
