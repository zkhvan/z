package list

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
)

type Options struct {
	IO     *iolib.IOStreams
	Config cmdutil.Config
}

func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Config: f.Config,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the current config values",
		RunE: func(cmd *cobra.Command, args []string) error {
			list := f.Config.List()
			fmt.Fprintln(opts.IO.Out, list)

			return nil
		},
	}

	return cmd
}
