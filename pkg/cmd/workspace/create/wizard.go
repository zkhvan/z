package create

import (
	"context"
	"errors"
	"fmt"

	"charm.land/huh/v2"

	"github.com/zkhvan/z/pkg/fzf"
	"github.com/zkhvan/z/pkg/project"
	"github.com/zkhvan/z/pkg/workspace"
)

// errAborted marks a user walking away from the wizard, which is not a failure.
var errAborted = errors.New("aborted")

// collectMembers walks the user through declaring members. It is a front-end
// for the same []workspace.Member the --member flags produce: no cloning, no
// validation beyond what create already does, no network beyond the project
// listing.
func (opts *Options) collectMembers(ctx context.Context) ([]workspace.Member, error) {
	projSvc, err := project.NewService(opts.config, project.WithExecutor(opts.executor))
	if err != nil {
		return nil, fmt.Errorf("creating project service: %w", err)
	}

	candidates, err := projSvc.ListProjects(ctx, &project.ListOptions{Local: true, Remote: true})
	if err != nil {
		return nil, err
	}

	var members []workspace.Member
	for {
		available := availableProjects(candidates, members)
		if len(available) == 0 {
			if len(members) == 0 {
				return nil, errors.New("no repositories available to add as members")
			}
			return members, nil
		}

		proj, err := fzf.One(ctx, available,
			fzf.WithIterator(projectByPath),
			fzf.WithHeader[project.Project](fmt.Sprintf("Select repository %d", len(members)+1)),
		)
		if err != nil {
			if errors.Is(err, fzf.ErrCanceled) {
				if len(members) == 0 {
					return nil, errAborted
				}
				return members, nil
			}
			return nil, err
		}

		member, more, err := opts.promptMember(ctx, proj.RemoteID)
		if err != nil {
			return nil, err
		}

		members = append(members, member)
		if !more {
			return members, nil
		}
	}
}

func (opts *Options) promptMember(ctx context.Context, repo string) (workspace.Member, bool, error) {
	var (
		branch  string
		baseRef string
		more    bool
	)

	form := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title(fmt.Sprintf("Branch for %s", repo)).
			Value(&branch).
			Validate(workspace.ValidateBranch),
		huh.NewInput().
			Title("Base ref").
			Description("Empty resolves the repository's default branch at materialize time.").
			Value(&baseRef).
			Validate(workspace.ValidateBaseRef),
		huh.NewConfirm().
			Title("Add another member?").
			Value(&more),
	)).
		WithInput(opts.io.In).
		// Prompts go to stderr so stdout stays the command's real output.
		WithOutput(opts.io.ErrOut)

	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return workspace.Member{}, false, errAborted
		}
		return workspace.Member{}, false, err
	}

	return workspace.Member{Repo: repo, Branch: branch, BaseRef: baseRef}, more, nil
}

// availableProjects drops what cannot become a member: repositories with no
// remote ID, and any whose worktree directory name is taken — the collision
// create would reject at the end, prevented at the point of choosing instead.
func availableProjects(candidates []project.Project, chosen []workspace.Member) []project.Project {
	used := make(map[string]bool, len(chosen))
	for _, m := range chosen {
		used[m.BaseName()] = true
	}

	available := make([]project.Project, 0, len(candidates))
	for _, p := range candidates {
		// BaseName comes from the domain type so the wizard and the collision
		// check in ValidateMembers can never disagree about what a worktree
		// directory is called.
		if p.RemoteID == "" || used[(workspace.Member{Repo: p.RemoteID}).BaseName()] {
			continue
		}
		available = append(available, p)
	}
	return available
}

func projectByPath(p project.Project, _ int) string {
	return fmt.Sprintf("%s %s", p.Source, p.RemoteID)
}
