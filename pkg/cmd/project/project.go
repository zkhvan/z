package project

import (
	"github.com/spf13/cobra"

	listCmd "github.com/zkhvan/z/pkg/cmd/project/list"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdProject(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	cmd.AddCommand(listCmd.NewCmdList(f))

	return cmd
}
