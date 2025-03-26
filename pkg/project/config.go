package project

import (
	"cmp"
	"fmt"

	"github.com/zkhvan/z/pkg/cmdutil"
)

type Config struct {
	MaxDepth int    `json:"max_depth"`
	Root     string `json:"root"`

	// TTL is the time to live (in seconds) for the cache.
	TTL int64 `json:"ttl"`

	// RemotePatterns is a list of patterns to match remote repositories.
	//
	// The pattern format is as follows:
	//
	//	owner/repo -> ./alternate-path
	//
	// The repo can be "*" to find all the repos under that owner. The
	// alternate path is relative to the root directory. If the alternate path
	// ends with a "/", the repo name (without the owner) will be used
	// instead.
	RemotePatterns []string        `json:"remote_patterns"`
	remotePatterns []remotePattern `json:"-"`
}

func NewConfig(cfg cmdutil.Config) (Config, error) {
	var c Config
	if err := cfg.Unmarshal("projects", &c); err != nil {
		return c, fmt.Errorf("error unmarshalling project config: %w", err)
	}

	c = c.setDefaults()

	patterns, err := c.parseRemotePatterns()
	if err != nil {
		return c, fmt.Errorf("error parsing remote patterns: %w", err)
	}
	c.remotePatterns = patterns

	return c, nil
}

func (c Config) setDefaults() Config {
	c.MaxDepth = cmp.Or(c.MaxDepth, 4)

	if c.Root == "" {
		c.Root = "~/Projects"
	}

	if c.TTL == 0 {
		c.TTL = 15 * 60 // 15 minutes
	}

	return c
}

func (c Config) parseRemotePatterns() ([]remotePattern, error) {
	patterns := make([]remotePattern, 0, len(c.RemotePatterns))

	for _, pattern := range c.RemotePatterns {
		parsed, err := parseRemotePattern(pattern)
		if err != nil {
			return nil, err
		}

		patterns = append(patterns, parsed)
	}

	return patterns, nil
}
