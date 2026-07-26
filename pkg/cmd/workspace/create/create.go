package create

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	From        string
	BranchFlags []string
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

			With --from, the member set is seeded from a definition and each
			branch is resolved from the definition's pattern. Use
			--branch <repo>=<name> to override one member's branch, keyed by
			either its base name or its full remote ID.

			--from and --member are mutually exclusive: --from declares the
			whole member set, and --branch is the way to deviate from it.

			Without --member or --from, an interactive wizard collects
			repositories, branches, and base refs. Running without either
			outside a terminal is an error rather than a wait for input.
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
	cmd.Flags().StringVar(&opts.From, "from", "", heredoc.Doc(`
		Definition to seed the member set from, by name.
	`))
	cmd.Flags().StringArrayVar(&opts.BranchFlags, "branch", nil, heredoc.Doc(`
		Override one member's branch, as <repo>=<name>. Repeatable. Requires --from.
	`))
	return cmd
}

func (opts *Options) Complete(_ *cobra.Command, args []string) error {
	opts.Name = args[0]
	return nil
}

func (opts *Options) Run(ctx context.Context) error {
	if opts.From != "" && len(opts.MemberFlags) > 0 {
		return errors.New("--from and --member are mutually exclusive: use --branch to deviate from the definition")
	}
	if opts.From == "" && len(opts.BranchFlags) > 0 {
		return errors.New("--branch requires --from: without a definition there is no pattern to override")
	}

	overrides, err := parseBranchOverrides(opts.BranchFlags)
	if err != nil {
		return err
	}

	members, err := opts.members(ctx)
	if err != nil {
		if errors.Is(err, errAborted) {
			fmt.Fprintln(opts.io.ErrOut, "Canceled; no workspace created")
			return nil
		}
		return err
	}

	svc, err := workspace.NewService(
		opts.config,
		workspace.WithExecutor(opts.executor),
	)
	if err != nil {
		return err
	}

	createOpts := workspace.CreateOptions{
		Members:         members,
		From:            opts.From,
		BranchOverrides: overrides,
	}
	if err := svc.Create(ctx, opts.Name, createOpts); err != nil {
		return err
	}

	fmt.Fprintf(opts.io.Out, "Created workspace %q\n", opts.Name)
	return nil
}

// parseBranchOverrides rejects a repeated key here rather than letting a map
// silently keep the last one.
func parseBranchOverrides(flags []string) (map[string]string, error) {
	if len(flags) == 0 {
		return nil, nil
	}

	overrides := make(map[string]string, len(flags))
	for _, flag := range flags {
		key, branch, ok := strings.Cut(flag, "=")
		if !ok || key == "" || branch == "" {
			return nil, fmt.Errorf("--branch %q: expected <repo>=<branch>", flag)
		}
		if err := workspace.ValidateBranch(branch); err != nil {
			return nil, fmt.Errorf("--branch %q: %w", flag, err)
		}
		if _, ok := overrides[key]; ok {
			return nil, fmt.Errorf("--branch %q: member %q already has an override", flag, key)
		}
		overrides[key] = branch
	}

	return overrides, nil
}

func (opts *Options) members(ctx context.Context) ([]workspace.Member, error) {
	if opts.From != "" {
		return nil, nil // the definition supplies them
	}

	if len(opts.MemberFlags) > 0 {
		members := make([]workspace.Member, 0, len(opts.MemberFlags))
		for _, flag := range opts.MemberFlags {
			m, err := workspace.ParseMember(flag)
			if err != nil {
				return nil, fmt.Errorf("--member %q: %w", flag, err)
			}
			members = append(members, m)
		}
		return members, nil
	}

	// Blocking on a pipe that will never answer is the one failure mode a
	// scripted caller cannot recover from.
	if !opts.io.IsInteractive() {
		return nil, errors.New("no members specified: pass --member or --from, or run interactively to use the wizard")
	}

	return opts.collectMembers(ctx)
}
