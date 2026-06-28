package workspace

import (
	"fmt"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/oslib"
)

type Config struct {
	// Root holds workspace instances; defaults to ~/Workspaces.
	Root string `json:"root"`
}

func NewConfig(cfg cmdutil.Config) (Config, error) {
	var c Config
	if err := cfg.Unmarshal("workspaces", &c); err != nil {
		if !config.IsNotFound(err) {
			return c, fmt.Errorf("error reading workspaces config: %w", err)
		}
	}

	if c.Root == "" {
		c.Root = "~/Workspaces"
	}
	c.Root = oslib.Expand(c.Root)

	return c, nil
}
