package factory

import (
	"github.com/zkhvan/z/pkg/cmd"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/iolib"
)

func New(appVersion string) *cmdutil.Factory {
	f := &cmdutil.Factory{
		AppVersion:     appVersion,
		ExecutableName: "z",
	}

	f.IOStreams = ioStreams(f)
	f.PluginHandler = defaultPluginHandler(f)
	f.Config = defaultConfig(f)

	return f
}

func ioStreams(_ *cmdutil.Factory) *iolib.IOStreams {
	return iolib.System()
}

func defaultPluginHandler(_ *cmdutil.Factory) cmdutil.PluginHandler {
	return cmd.NewDefaultPluginHandler([]string{"z"})
}

func defaultConfig(_ *cmdutil.Factory) cmdutil.Config {
	c, err := config.New()
	if err != nil {
		panic(err)
	}
	return c
}
