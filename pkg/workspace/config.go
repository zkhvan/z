package workspace

import (
	"fmt"
	"path/filepath"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/oslib"
)

type Config struct {
	// Root holds workspace instances; defaults to ~/Workspaces.
	Root string `json:"root"`

	// DefinitionsRoot holds workspace definitions; defaults to a definitions
	// directory beside the z config file, so it follows a config dir override
	// rather than reaching the real user config dir.
	DefinitionsRoot string `json:"definitions_root"`
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

	if c.DefinitionsRoot == "" {
		c.DefinitionsRoot = filepath.Join(cfg.Dir(), "definitions")
	}
	c.DefinitionsRoot = oslib.Expand(c.DefinitionsRoot)

	return c, nil
}
