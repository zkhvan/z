package shell

import (
	"github.com/spf13/cobra"

	zshCmd "github.com/zkhvan/z/pkg/cmd/shell/zsh"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdShell(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Shell configuration",
	}

	cmd.AddCommand(zshCmd.NewCmdZsh(f))

	return cmd
}
