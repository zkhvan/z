package version

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
)

type Options struct {
	io          *iolib.IOStreams
	VersionInfo string
}

func NewCmdVersion(f *cmdutil.Factory, version, date string) *cobra.Command {
	opts := &Options{
		io:          f.IOStreams,
		VersionInfo: version,
	}

	cmd := &cobra.Command{
		Use:    "version",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(cmd, args); err != nil {
				return err
			}
			return opts.Run()
		},
	}

	return cmd
}

func (opts *Options) Complete(cmd *cobra.Command, args []string) error {
	opts.VersionInfo = cmd.Root().Annotations["versionInfo"]
	return nil
}

func (o *Options) Run() error {
	fmt.Fprint(o.io.Out, o.VersionInfo)
	return nil
}

func Format(version, buildDate string) string {
	version = strings.TrimPrefix(version, "v")

	var dateStr string
	if buildDate != "" {
		dateStr = fmt.Sprintf(" (%s)", buildDate)
	}

	return fmt.Sprintf("z version %s%s\n%s\n", version, dateStr, changelogURL(version))
}

func changelogURL(version string) string {
	path := "https://github.com/zkhvan/z"
	r := regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[\w.]+)?$`)
	if !r.MatchString(version) {
		return fmt.Sprintf("%s/releases/latest", path)
	}

	url := fmt.Sprintf("%s/releases/tag/v%s", path, strings.TrimPrefix(version, "v"))
	return url
}
