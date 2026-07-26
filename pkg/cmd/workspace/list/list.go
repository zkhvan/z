package list

import (
	"context"
	"fmt"
	"io"
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
			Scan the workspaces root one level deep and render each instance,
			sorted alphabetically.

			Instance rows: name, directory (~ collapsed), status.
			Member rows:   repo, branch, worktree state.

			On a terminal the instances are drawn as an aligned tree with
			status symbols and color. Anywhere else — a pipe, a file, CI — the
			output stays tab-separated plain text so it can be parsed.
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

	if !opts.io.IsTerminal() {
		return renderPlain(opts.io.Out, instances)
	}
	return renderTree(opts.io.StyledOut(), instances)
}

// renderPlain is the machine-readable rendering: tab-separated, no symbols, no
// escapes. Its bytes are a contract for anything parsing z's output.
func renderPlain(out io.Writer, instances []workspace.Instance) error {
	for _, inst := range instances {
		fmt.Fprintf(out, "%s\t%s\t%s\n",
			inst.Name,
			collapseTilde(inst.Dir),
			inst.Status,
		)
		for _, m := range inst.Members {
			fmt.Fprintf(out, "  %s\t%s\t%s\n",
				m.Repo,
				m.Branch,
				m.State,
			)
		}
	}

	return nil
}

func renderTree(out io.Writer, instances []workspace.Instance) error {
	st := iolib.NewStyles()

	// Widths span every instance so the member columns form one grid rather
	// than a ragged block per workspace, and are measured in terminal cells
	// rather than bytes.
	repoWidth, branchWidth := 0, 0
	for _, inst := range instances {
		for _, m := range inst.Members {
			repoWidth = max(repoWidth, iolib.Width(m.Repo))
			branchWidth = max(branchWidth, iolib.Width(m.Branch))
		}
	}

	for _, inst := range instances {
		fmt.Fprintf(out, "%s %s  %s\n",
			st.Bold.Render(inst.Name),
			st.Dim.Render(collapseTilde(inst.Dir)),
			statusLabel(st, inst.Status),
		)

		for i, m := range inst.Members {
			connector := "\u251c\u2500"
			if i == len(inst.Members)-1 {
				connector = "\u2514\u2500"
			}
			fmt.Fprintf(out, "%s %s  %s  %s\n",
				st.Dim.Render(connector),
				pad(m.Repo, repoWidth),
				st.Cyan.Render(m.Branch)+spaces(branchWidth-iolib.Width(m.Branch)),
				memberLabel(st, m.State),
			)
		}
	}

	return nil
}

func statusLabel(st iolib.Styles, status workspace.InstanceStatus) string {
	switch status {
	case workspace.InstanceStatusMaterialized:
		return st.Green.Render("\u2713 " + string(status))
	case workspace.InstanceStatusPartial:
		return st.Yellow.Render("\u25d0 " + string(status))
	case workspace.InstanceStatusArchived:
		return st.Dim.Render("\u25cb " + string(status))
	default:
		return st.Dim.Render("\u00b7 " + string(status))
	}
}

func memberLabel(st iolib.Styles, state workspace.MemberState) string {
	switch state {
	case workspace.MemberStateClean:
		return st.Green.Render("\u2713 " + string(state))
	case workspace.MemberStateDirty:
		return st.Yellow.Render("\u25cf " + string(state))
	default:
		// The unknown state is already a dash; a symbol in front of it reads as
		// two placeholders.
		return st.Dim.Render(string(state))
	}
}

func pad(s string, width int) string {
	return s + spaces(width-iolib.Width(s))
}

// spaces pads outside the styling, so trailing padding never carries color.
func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

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
