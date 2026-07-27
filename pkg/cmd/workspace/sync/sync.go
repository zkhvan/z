package sync

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

	Name            string
	Force           bool
	DryRun          bool
	ErrorOnConflict bool
}

func NewCmdSync(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:       f.IOStreams,
		config:   f.Config,
		executor: exec.New(),
	}

	cmd := &cobra.Command{
		Use:   "sync [<name>]",
		Short: "Refresh an instance from its definition",
		Long: heredoc.Doc(`
			Copy the definition's files into the instance, updating any that have
			not been touched since z last wrote them.

			A file you have edited locally is reported and left alone. To accept
			the definition's version, delete your copy and sync again; deletions
			are always safe to overwrite.

			Member worktrees are never scanned, propagated, or removed.

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

	cmd.Flags().BoolVar(&opts.Force, "force", false, heredoc.Doc(`
		Mirror the definition exactly: overwrite locally modified files and
		remove content the definition does not ship. Ignored content, including
		member worktrees, is still never touched.
	`))
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false,
		"Report the planned changes without writing anything.")
	cmd.Flags().BoolVar(&opts.ErrorOnConflict, "error-on-conflict", false,
		"Exit non-zero when any file was skipped as locally modified.")

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

	report, err := svc.Sync(ctx, name, workspace.SyncOptions{
		Force:  opts.Force,
		DryRun: opts.DryRun,
	})
	if err != nil {
		return err
	}

	opts.render(name, report)

	if opts.ErrorOnConflict && len(report.Conflicts) > 0 {
		return fmt.Errorf("%d file(s) were skipped as locally modified", len(report.Conflicts))
	}
	return nil
}

func (opts *Options) render(name string, report workspace.SyncReport) {
	for _, p := range report.Problems {
		fmt.Fprintf(opts.io.ErrOut, "skipped %s: %s\n", p.Path, p.Reason)
	}

	for _, c := range report.Conflicts {
		fmt.Fprintf(opts.io.ErrOut,
			"skipped %s: modified locally; delete it and sync again to take the definition's version\n",
			c.Path)
	}

	verb := "Synced"
	if report.DryRun {
		verb = "Would sync"
	}

	skipped := len(report.Conflicts) + len(report.Problems)

	// "Up to date" has to mean it: a run that skipped a conflict has unfinished
	// business, even though it wrote nothing.
	if !report.Changed() && skipped == 0 {
		fmt.Fprintf(opts.io.Out, "%s %q: already up to date\n", verb, name)
		return
	}

	for _, c := range report.Applied {
		action := "update"
		switch {
		case c.Deletion():
			action = "remove"
		case c.Old == nil:
			action = "create"
		}
		fmt.Fprintf(opts.io.Out, "  %s %s\n", action, c.Path)
	}

	fmt.Fprintf(opts.io.Out, "%s %q from definition %q: %d change(s)",
		verb, name, report.Definition, len(report.Applied))
	if skipped > 0 {
		fmt.Fprintf(opts.io.Out, ", %d skipped", skipped)
	}
	fmt.Fprintln(opts.io.Out)
}
