package list

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
)

type Options struct {
	io     *iolib.IOStreams
	config cmdutil.Config
}

func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		io:     f.IOStreams,
		config: f.Config,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the current config values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run()
		},
	}

	return cmd
}

func (o *Options) Run() error {
	list := o.config.List()
	fmt.Fprintln(o.io.Out, list)
	return nil
}
