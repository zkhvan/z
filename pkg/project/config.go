package project

import (
	"cmp"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/oslib"
)

type Config struct {
	// MaxDepth is the maximum depth of the project tree to search for projects.
	MaxDepth int `json:"max_depth"`

	// Root is the root directory for the projects.
	Root string `json:"root"`

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

	// remotePatterns is a list of parsed remote patterns.
	remotePatterns []remotePattern `json:"-"`
}

func NewConfig(cfg cmdutil.Config) (Config, error) {
	var c Config
	if err := cfg.Unmarshal("projects", &c); err != nil {
		if !config.IsNotFound(err) {
			return c, fmt.Errorf("error unmarshalling project config: %w", err)
		}
	}

	c = c.setDefaults()
	c.Root = oslib.Expand(c.Root)

	patterns, err := c.parseRemotePatterns()
	if err != nil {
		return c, fmt.Errorf("error parsing remote patterns: %w", err)
	}
	c.remotePatterns = patterns

	return c, nil
}

func (c Config) setDefaults() Config {
	c.MaxDepth = cmp.Or(c.MaxDepth, 3)

	if c.Root == "" {
		c.Root = "~/Projects"
	}

	if c.TTL == 0 {
		c.TTL = 15 * 60 // 15 minutes
	}

	return c
}

// remotePattern represents a pattern with the following format:
//
//	owner/repo -> ./alternate-path
//
// If the pattern is in the format above, the AlternatePath will be set.
type remotePattern struct {
	original string

	Owner         string
	Repo          string
	AlternatePath string
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

func parseRemotePattern(pattern string) (remotePattern, error) {
	out := remotePattern{
		original: pattern,
	}

	// Check and parse if the pattern contains an alternate path.
	parts := strings.Split(pattern, "->")
	if len(parts) == 2 {
		alternatePath := strings.TrimSpace(parts[1])
		out.AlternatePath = filepath.Clean(alternatePath)
	}

	// Parse the owner/repo
	parts = strings.Split(strings.TrimSpace(parts[0]), "/")
	if len(parts) != 2 {
		return out, fmt.Errorf("invalid pattern: %q", pattern)
	}

	out.Owner = parts[0]
	out.Repo = parts[1]

	return out, nil
}
