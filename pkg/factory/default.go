package factory

import (
	"os"

	"github.com/zkhvan/z/pkg/cmd"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
)

func New(appVersion string) *cmdutil.Factory {
	f := &cmdutil.Factory{
		AppVersion:     appVersion,
		ExecutableName: "z",
	}

	f.IOStreams = ioStreams(f)
	f.PluginHandler = defaultPluginHandler(f)

	return f
}

func ioStreams(f *cmdutil.Factory) *iolib.IOStreams {
	io := &iolib.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	return io
}

func defaultPluginHandler(f *cmdutil.Factory) cmdutil.PluginHandler {
	return cmd.NewDefaultPluginHandler([]string{"z"})
}
