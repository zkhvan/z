package checkout

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
		Use:   "checkout [<name>] [<repo>] <branch>",
		Short: "Switch one workspace member to another branch",
		Long: heredoc.Doc(`
			Switch a materialized member's worktree in place, keeping untracked
			build artifacts, then record the new branch in the manifest.

			The member is matched by directory name first, then by its full
			owner/repo remote ID.

			Arguments bind from the right, so the leading ones can be omitted and
			taken from the current directory: the name from inside an instance,
			the repo from inside a member worktree.

			Branches are never invented: an unknown branch is an error unless
			--create is given. A created branch starts from --base, which
			defaults to the worktree's current HEAD so stacked branches stack.
			A branch that exists on the remote is checked out normally, without
			--create.

			Uncommitted changes, or a branch already checked out in another
			worktree, abort the switch and leave the manifest untouched.
		`),
		Args: cobra.RangeArgs(1, 3),
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

// Complete binds arguments from the right, leaving the omitted ones — <repo>,
// then <name> — to the current directory.
func (opts *Options) Complete(_ *cobra.Command, args []string) error {
	opts.Branch = args[len(args)-1]
	if len(args) > 1 {
		opts.Repo = args[len(args)-2]
	}
	if len(args) > 2 {
		opts.Name = args[len(args)-3]
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

	name, repo, err := wsloc.NameAndMember(svc, opts.Name, opts.Repo)
	if err != nil {
		return err
	}

	result, err := svc.Checkout(ctx, name, repo, opts.Branch, workspace.CheckoutOptions{
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
