package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	configCmd "github.com/zkhvan/z/pkg/cmd/config"
	"github.com/zkhvan/z/pkg/cmd/plugin"
	projectCmd "github.com/zkhvan/z/pkg/cmd/project"
	tmuxCmd "github.com/zkhvan/z/pkg/cmd/tmux"
	versionCmd "github.com/zkhvan/z/pkg/cmd/version"
	"github.com/zkhvan/z/pkg/cmdutil"
)

func NewCmdRoot(f *cmdutil.Factory, version, date string) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "z <command> <subcommand> [flags]",
		Short: "Zhenya's CLI",
		Long:  `Work seamlessly with the command line.`,
		Annotations: map[string]string{
			"versionInfo": versionCmd.Format(version, date),
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == cobra.ShellCompRequestCmd {
				// This is the __complete or __completeNoDesc command which
				// indicates shell completion has been requested.
				plugin.SetupPluginCompletion(cmd, args)
			}

			return nil
		},
	}

	cmd.PersistentFlags().Bool("help", false, "Show help for command")

	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	// Define groups
	cmd.AddGroup(&cobra.Group{ID: "plugin", Title: "Plugin commands"})

	// Commands
	cmd.AddCommand(versionCmd.NewCmdVersion(f, version, date))
	cmd.AddCommand(tmuxCmd.NewCmdTmux(f))
	cmd.AddCommand(projectCmd.NewCmdProject(f))
	cmd.AddCommand(configCmd.NewCmdConfig(f))

	if f.PluginHandler == nil {
		return cmd, nil
	}

	if len(os.Args) > 1 {
		extraArgs := os.Args[1:]

		if _, _, err := cmd.Find(extraArgs); err != nil {
			// Also check the commands that will be added by Cobra.
			// These commands are only added once rootCmd.Execute() is called, so we
			// need to check them explicitly here.
			var cmdName string // first "non-flag" arguments
			for _, arg := range extraArgs {
				if !strings.HasPrefix(arg, "-") {
					cmdName = arg
					break
				}
			}

			switch cmdName {
			case "help", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
				// Don't search for a plugin
			default:
				if err := HandlePluginCommand(f.PluginHandler, extraArgs, false); err != nil {
					fmt.Fprintf(f.IOStreams.ErrOut, "Error: %v\n", err)
					os.Exit(1)
				}
			}
		}
	}

	return cmd, nil
}
