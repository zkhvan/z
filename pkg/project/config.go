package project

import (
	"cmp"
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
	RemotePatterns []string `json:"remote_patterns"`
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
