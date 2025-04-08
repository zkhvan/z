package selectcmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmd/project/internal"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/fzf"
	"github.com/zkhvan/z/pkg/gh"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/project"
	"github.com/zkhvan/z/pkg/tmux"
)

type Options struct {
	*internal.ProjectOptions
	io     *iolib.IOStreams
	config cmdutil.Config

	RefreshCache bool
	Remote       bool
	Local        bool
	Tmux         bool
}

func NewCmdSelect(f *cmdutil.Factory, projectOpts *internal.ProjectOptions) *cobra.Command {
	opts := &Options{
		ProjectOptions: projectOpts,
		io:             f.IOStreams,
		config:         f.Config,
	}

	cmd := &cobra.Command{
		Use:   "select",
		Short: "Select a project interactively",
		Long: heredoc.Doc(`
			Interactively select a project from the list of known projects
			using a fuzzy finder. Outputs the selected project's absolute path
			to stdout.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(cmd, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().BoolVar(&opts.RefreshCache, "refresh-cache", false, "Refresh the cache")
	cmd.Flags().BoolVar(&opts.Remote, "remote", true, "List remote projects")
	cmd.Flags().BoolVar(&opts.Local, "local", true, "List local projects")
	cmd.Flags().BoolVar(&opts.Tmux, "tmux", false, "Open in tmux")

	return cmd
}

// Complete handles the logic for default flags when filtering by type.
func (opts *Options) Complete(cmd *cobra.Command, _ []string) error {
	remoteChanged := cmd.Flags().Changed("remote")
	localChanged := cmd.Flags().Changed("local")

	// If the user explicitly sets either --local or --remote,
	// assume they want *only* that type unless the other is also explicitly set.
	if remoteChanged || localChanged {
		if !remoteChanged {
			opts.Remote = false
		}
		if !localChanged {
			opts.Local = false
		}
	}
	return nil
}

func (opts *Options) Run(ctx context.Context) error {
	service, err := project.NewService(
		opts.config,
		project.WithRefreshCache(opts.RefreshCache),
		project.WithCacheDir(opts.CacheDir),
	)
	if err != nil {
		return err
	}

	results, err := service.ListProjects(ctx, &project.ListOptions{
		Local:  opts.Local,
		Remote: opts.Remote,
	})
	if err != nil {
		return err
	}

	shouldCD := true

	fzfOpts := []fzf.Option[project.Project]{
		fzf.WithIterator(projectByPath),
		fzf.WithBinding("ctrl-y", func(p project.Project) error {
			shouldCD = false

			return clipboard.WriteAll(p.AbsolutePath)
		}),
		fzf.WithBinding("alt-enter", func(p project.Project) error {
			shouldCD = false

			opts := &gh.RepoViewOptions{Web: true}

			switch p.Source {
			case project.SourceTypeRemote, project.SourceTypeSynced:
				opts.RepositoryID = p.RemoteID
			case project.SourceTypeLocal:
				opts.WorkingDirectory = p.AbsolutePath
			default:
				return fmt.Errorf("unsupported project source: %s", p.Source)
			}

			_, err = gh.NewClient().RepoView(ctx, opts)
			return err
		}),
	}

	header := "CTRL-Y: Yank | ALT-ENTER: View in browser"
	if opts.Tmux {
		header = fmt.Sprintf("ENTER: Change session | %s", header)
	} else {
		header = fmt.Sprintf("ENTER: Change directory | %s", header)
	}
	fzfOpts = append(fzfOpts, fzf.WithHeader[project.Project](header))

	proj, err := fzf.One(
		ctx,
		results,
		fzfOpts...,
	)
	if errors.Is(err, fzf.ErrCanceled) {
		return nil
	}
	if err != nil {
		return err
	}

	if _, err := os.Lstat(proj.AbsolutePath); os.IsNotExist(err) {
		output, err := service.CloneProject(ctx, proj)
		if err != nil {
			return err
		}
		fmt.Fprintln(opts.io.Out, output)
	}

	if shouldCD {
		if opts.Tmux {
			return tmux.NewSession(ctx, &tmux.NewOptions{
				Name: proj.LocalID,
				Dir:  proj.AbsolutePath,
			})
		}

		fmt.Fprintf(opts.io.Out, "cd %s\n", proj.AbsolutePath)
	}
	return nil
}

func projectByPath(p project.Project, _ int) string {
	return fmt.Sprintf("%s %s", p.Source, p.LocalID)
}
