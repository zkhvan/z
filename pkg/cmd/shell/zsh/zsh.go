package zsh

import (
	"context"
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
)

//go:embed z.zsh
var zshScript []byte

type Options struct {
	io *iolib.IOStreams
}

func NewCmdZsh(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "zsh",
		Short: "A zsh shell integration wrapper for z",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(cmd, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (opts *Options) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

func (opts *Options) Run(ctx context.Context) error {
	_, err := opts.io.Out.Write(zshScript)
	if err != nil {
		return err
	}

	return nil
}
