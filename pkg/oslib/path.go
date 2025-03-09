package oslib

import (
	"os"
	"strings"
)

func Expand(path string) string {
	if strings.HasPrefix(path, "~") {
		path = expandTilde(path)
	}

	return os.ExpandEnv(path)
}

func expandTilde(path string) string {
	if len(path) > 1 && !strings.HasPrefix(path, "~/") {
		return path
	}

	return strings.Replace(path, "~", "$HOME", 1)
}
