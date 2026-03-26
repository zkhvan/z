package kill

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/tmux"
)

type Options struct {
	Name    string
	All     bool
	Zombies bool
}

func NewCmdKill(_ *cmdutil.Factory) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:   "kill [name]",
		Short: "Kill popup sessions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Name = args[0]
			}
			if opts.Name == "" && !opts.All && !opts.Zombies {
				return fmt.Errorf("provide a popup name or use --all/--zombies")
			}
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "Kill all popup sessions for the current session")
	cmd.Flags().BoolVar(&opts.Zombies, "zombies", false, "Kill orphaned popup sessions whose parent no longer exists")

	return cmd
}

func (opts *Options) Run(ctx context.Context) error {
	if opts.Zombies {
		return opts.killZombies(ctx)
	}

	parentName, err := tmux.CurrentSessionName(ctx)
	if err != nil {
		return err
	}

	if !opts.All {
		popupSessionName := tmux.ToPopupSessionName(parentName, opts.Name)
		if !tmux.HasSession(ctx, popupSessionName) {
			return fmt.Errorf("popup session %q not found", opts.Name)
		}
		return tmux.KillSession(ctx, tmux.Session{Name: popupSessionName})
	}

	sessions, err := tmux.ListSessions(ctx, nil)
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if _, ok := tmux.ExtractPopupName(session.Name, parentName); ok {
			if err := tmux.KillSession(ctx, session); err != nil {
				return err
			}
		}
	}

	return nil
}

func (opts *Options) killZombies(ctx context.Context) error {
	sessions, err := tmux.ListSessions(ctx, nil)
	if err != nil {
		return err
	}

	// Build a set of non-popup session names
	alive := make(map[string]struct{})
	for _, s := range sessions {
		if !tmux.IsPopupSession(s.Name) {
			alive[s.Name] = struct{}{}
		}
	}

	// Kill popups whose parent is no longer alive
	for _, s := range sessions {
		parent, ok := tmux.ExtractPopupParent(s.Name)
		if !ok {
			continue
		}
		if _, exists := alive[parent]; !exists {
			if err := tmux.KillSession(ctx, s); err != nil {
				return err
			}
		}
	}

	return nil
}
