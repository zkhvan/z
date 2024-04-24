package cmdutil

import (
	"github.com/zkhvan/z/pkg/iolib"
)

type Factory struct {
	AppVersion     string
	ExecutableName string

	PluginHandler PluginHandler
	IOStreams     *iolib.IOStreams
}
