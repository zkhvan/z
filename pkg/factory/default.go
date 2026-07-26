package factory

import (
	"github.com/zkhvan/z/pkg/cmd"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/iolib"
)

// New fails rather than panicking on unusable configuration: a mistyped
// $Z_CONFIG_DIR is ordinary user error and deserves a message, not a stack
// trace.
func New(appVersion string) (*cmdutil.Factory, error) {
	f := &cmdutil.Factory{
		AppVersion:     appVersion,
		ExecutableName: "z",
	}

	f.IOStreams = ioStreams(f)
	f.PluginHandler = defaultPluginHandler(f)

	cfg, err := config.New()
	if err != nil {
		return nil, err
	}
	f.Config = cfg

	return f, nil
}

func ioStreams(_ *cmdutil.Factory) *iolib.IOStreams {
	return iolib.System()
}

func defaultPluginHandler(_ *cmdutil.Factory) cmdutil.PluginHandler {
	return cmd.NewDefaultPluginHandler([]string{"z"})
}
