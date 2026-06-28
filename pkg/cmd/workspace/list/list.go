package list

import (
	"context"
	"fmt"
	"os/user"
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
}

func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:       f.IOStreams,
		config:   f.Config,
		executor: exec.New(),
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspace instances",
		Long: heredoc.Doc(`
			Scan the workspaces root one level deep and render each instance
			as a plain ASCII tree, sorted alphabetically.

			Instance rows: name, directory (~ collapsed), status.
			Member rows:   repo, branch (no worktree state when unmaterialized).
		`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (opts *Options) Run(ctx context.Context) error {
	svc, err := workspace.NewService(
		opts.config,
		workspace.WithExecutor(opts.executor),
	)
	if err != nil {
		return err
	}

	instances, err := svc.List(ctx)
	if err != nil {
		return err
	}

	for _, inst := range instances {
		fmt.Fprintf(opts.io.Out, "%s\t%s\t%s\n",
			inst.Name,
			collapseTilde(inst.Dir),
			inst.Status,
		)
		for _, m := range inst.Members {
			fmt.Fprintf(opts.io.Out, "  %s\t%s\t%s\n",
				m.Repo,
				m.Branch,
				"—",
			)
		}
	}

	return nil
}

// collapseTilde replaces the home directory prefix with "~".
func collapseTilde(path string) string {
	u, err := user.Current()
	if err != nil || u.HomeDir == "" {
		return path
	}
	home := u.HomeDir
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
