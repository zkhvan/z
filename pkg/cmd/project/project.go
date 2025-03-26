package project

import (
	"github.com/spf13/cobra"

	cloneCmd "github.com/zkhvan/z/pkg/cmd/project/clone"
	listCmd "github.com/zkhvan/z/pkg/cmd/project/list"
	refreshCmd "github.com/zkhvan/z/pkg/cmd/project/refresh"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdProject(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	cmd.AddCommand(listCmd.NewCmdList(f))
	cmd.AddCommand(refreshCmd.NewCmdRefresh(f))
	cmd.AddCommand(cloneCmd.NewCmdClone(f))

	return cmd
}
