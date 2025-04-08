package zsh

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"

	_ "embed"
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (opts *Options) Run(_ context.Context) error {
	_, err := opts.io.Out.Write(zshScript)
	if err != nil {
		return err
	}

	return nil
}
