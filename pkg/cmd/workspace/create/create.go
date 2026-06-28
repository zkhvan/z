package create

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

	Name        string
	MemberFlags []string
}

func NewCmdCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:       f.IOStreams,
		config:   f.Config,
		executor: exec.New(),
	}

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a workspace instance",
		Long: heredoc.Doc(`
			Create a multi-repo workspace instance on disk.

			Each --member flag is in the form owner/repo@branch[:base_ref].
			The @branch is required; :base_ref is optional and defaults to
			the repository's default branch at materialize time.
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(cmd, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().StringArrayVar(&opts.MemberFlags, "member", nil, heredoc.Doc(`
		Repository member in the form owner/repo@branch[:base_ref]. Repeatable.
	`))
	_ = cmd.MarkFlagRequired("member")

	return cmd
}

func (opts *Options) Complete(_ *cobra.Command, args []string) error {
	opts.Name = args[0]
	return nil
}

func (opts *Options) Run(ctx context.Context) error {
	members := make([]workspace.Member, 0, len(opts.MemberFlags))
	for _, flag := range opts.MemberFlags {
		m, err := workspace.ParseMember(flag)
		if err != nil {
			return fmt.Errorf("--member %q: %w", flag, err)
		}
		members = append(members, m)
	}

	svc, err := workspace.NewService(
		opts.config,
		workspace.WithExecutor(opts.executor),
	)
	if err != nil {
		return err
	}

	if err := svc.Create(ctx, opts.Name, members); err != nil {
		return err
	}

	fmt.Fprintf(opts.io.Out, "Created workspace %q\n", opts.Name)
	return nil
}
