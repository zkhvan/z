package config

import (
	"github.com/spf13/cobra"

	getCmd "github.com/zkhvan/z/pkg/cmd/config/list"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdConfig(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage config",
	}

	cmd.AddCommand(getCmd.NewCmdList(f))

	return cmd
}
