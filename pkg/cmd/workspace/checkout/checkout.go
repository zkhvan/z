package checkout

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

	Name   string
	Repo   string
	Branch string
	Create bool
	Base   string
}

func NewCmdCheckout(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:       f.IOStreams,
		config:   f.Config,
		executor: exec.New(),
	}

	cmd := &cobra.Command{
		Use:   "checkout <name> <repo> <branch>",
		Short: "Switch one workspace member to another branch",
		Long: heredoc.Doc(`
			Switch a materialized member's worktree in place, keeping untracked
			build artifacts, then record the new branch in the manifest.

			The member is matched by directory name first, then by its full
			owner/repo remote ID.

			Branches are never invented: an unknown branch is an error unless
			--create is given. A created branch starts from --base, which
			defaults to the worktree's current HEAD so stacked branches stack.
			A branch that exists on the remote is checked out normally, without
			--create.

			Uncommitted changes, or a branch already checked out in another
			worktree, abort the switch and leave the manifest untouched.
		`),
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(cmd, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().BoolVarP(&opts.Create, "create", "c", false, "Create the branch if it does not exist")
	cmd.Flags().StringVar(&opts.Base, "base", "", heredoc.Doc(`
		Start point for --create, defaulting to the worktree's current HEAD.
	`))

	return cmd
}

func (opts *Options) Complete(_ *cobra.Command, args []string) error {
	opts.Name = args[0]
	opts.Repo = args[1]
	opts.Branch = args[2]
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

	result, err := svc.Checkout(ctx, opts.Name, opts.Repo, opts.Branch, workspace.CheckoutOptions{
		Create: opts.Create,
		Base:   opts.Base,
	})
	if err != nil {
		return err
	}

	switch {
	case result.Previous == result.Branch:
		fmt.Fprintf(opts.io.Out, "%s: already on %s\n", result.Repo, result.Branch)
	case result.Created:
		fmt.Fprintf(opts.io.Out, "%s: %s -> %s (new branch)\n", result.Repo, result.Previous, result.Branch)
	default:
		fmt.Fprintf(opts.io.Out, "%s: %s -> %s\n", result.Repo, result.Previous, result.Branch)
	}

	return nil
}
