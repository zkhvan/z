package use

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/tmux"
)

type Options struct {
	Name   string
	Width  string
	Height string
}

func NewCmdUse(_ *cmdutil.Factory) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:   "use <name>",
		Short: "Open a popup session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&opts.Width, "width", "80%", "Popup width")
	cmd.Flags().StringVar(&opts.Height, "height", "80%", "Popup height")

	return cmd
}

func (opts *Options) Run(ctx context.Context) error {
	parentName, err := tmux.CurrentSessionName(ctx)
	if err != nil {
		return err
	}

	popupSessionName := tmux.ToPopupSessionName(parentName, opts.Name)

	if !tmux.HasSession(ctx, popupSessionName) {
		session, err := tmux.NewSessionDetached(ctx, &tmux.NewOptions{
			Name: popupSessionName,
		})
		if err != nil {
			return err
		}

		if err := tmux.SetSessionOption(ctx, session.ID, "status", "off"); err != nil {
			return err
		}
		if err := tmux.SetSessionOption(ctx, session.ID, "prefix", "None"); err != nil {
			return err
		}
		if err := tmux.SetSessionOption(ctx, session.ID, "key-table", "popup"); err != nil {
			return err
		}
		if err := tmux.BindKey(ctx, "popup", "C-_", "detach"); err != nil {
			return err
		}
		if err := tmux.BindKey(ctx, "popup", "M-[", "copy-mode"); err != nil {
			return err
		}
	}

	return tmux.DisplayPopup(ctx, &tmux.DisplayPopupOptions{
		Width:        opts.Width,
		Height:       opts.Height,
		ShellCommand: fmt.Sprintf("tmux attach-session -t '=%s'", popupSessionName),
	})
}
