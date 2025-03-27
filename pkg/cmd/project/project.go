package project

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	cloneCmd "github.com/zkhvan/z/pkg/cmd/project/clone"
	"github.com/zkhvan/z/pkg/cmd/project/internal"
	listCmd "github.com/zkhvan/z/pkg/cmd/project/list"
	refreshCmd "github.com/zkhvan/z/pkg/cmd/project/refresh"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdProject(f *cmdutil.Factory) *cobra.Command {
	projectOpts := &internal.ProjectOptions{}

	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	cmd.PersistentFlags().StringVar(&projectOpts.CacheDir, "cache-dir", "", heredoc.Doc(`
		The directory to cache the list of projects. By default, the cache
		will be saved in $XDG_CACHE_DIR/z or ~/.cache/z/
	`))

	cmd.AddCommand(listCmd.NewCmdList(f, projectOpts))
	cmd.AddCommand(refreshCmd.NewCmdRefresh(f, projectOpts))
	cmd.AddCommand(cloneCmd.NewCmdClone(f, projectOpts))

	return cmd
}
