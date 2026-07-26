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
		Short: "List workspace definitions",
		Long: heredoc.Doc(`
			Scan the definitions root one level deep and render each definition,
			sorted alphabetically.

			Definition rows: name, directory (~ collapsed), branch pattern.
			Member rows:     repo, base ref.

			A directory holding a definition manifest that cannot be used is
			reported with its error rather than skipped: the marker file says a
			definition was intended.
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

	defs, err := svc.ListDefinitions(ctx)
	if err != nil {
		return err
	}

	if !opts.io.IsTerminal() {
		return renderPlain(opts.io.Out, defs)
	}
	return renderTree(opts.io.StyledOut(), defs)
}

func renderPlain(out io.Writer, defs []workspace.Definition) error {
	for _, def := range defs {
		fmt.Fprintf(out, "%s\t%s\t%s\n",
			def.Name,
			collapseTilde(def.Dir),
			summary(def),
		)
		for _, m := range def.Members {
			fmt.Fprintf(out, "  %s\t%s\n", m.Repo, baseRefLabel(m.BaseRef))
		}
	}

	return nil
}

func renderTree(out io.Writer, defs []workspace.Definition) error {
	st := iolib.NewStyles()

	repoWidth := 0
	for _, def := range defs {
		for _, m := range def.Members {
			repoWidth = max(repoWidth, iolib.Width(m.Repo))
		}
	}

	for _, def := range defs {
		fmt.Fprintf(out, "%s %s  %s\n",
			st.Bold.Render(def.Name),
			st.Dim.Render(collapseTilde(def.Dir)),
			summaryLabel(st, def),
		)

		for i, m := range def.Members {
			connector := "\u251c\u2500"
			if i == len(def.Members)-1 {
				connector = "\u2514\u2500"
			}
			fmt.Fprintf(out, "%s %s  %s\n",
				st.Dim.Render(connector),
				pad(m.Repo, repoWidth),
				st.Dim.Render(baseRefLabel(m.BaseRef)),
			)
		}
	}

	return nil
}

// summary is the third column: the pattern a healthy definition will apply, or
// why a broken one cannot be used.
func summary(def workspace.Definition) string {
	if def.Broken() {
		return "broken: " + oneLine(def.Err.Error())
	}
	return def.BranchPattern
}

func summaryLabel(st iolib.Styles, def workspace.Definition) string {
	if def.Broken() {
		return st.Red.Render("\u2717 " + summary(def))
	}
	return st.Cyan.Render(def.BranchPattern)
}

func baseRefLabel(baseRef string) string {
	if baseRef == "" {
		return string(workspace.MemberStateUnknown)
	}
	return baseRef
}

// oneLine keeps a multi-line error (a YAML parse failure is typically two
// lines) from breaking the line-per-row contract of the plain rendering.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func pad(s string, width int) string {
	n := width - iolib.Width(s)
	if n <= 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
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
