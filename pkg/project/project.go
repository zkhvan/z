package project

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/zkhvan/z/pkg/fd"
	"github.com/zkhvan/z/pkg/gh"
	"github.com/zkhvan/z/pkg/oslib"
)

const (
	CACHE_FILE = "projects.json"
)

type Config struct {
	MaxDepth int    `json:"max_depth"`
	Root     string `json:"root"`

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

	return c
}

type ProjectType string

const (
	Local  ProjectType = "local"
	Remote ProjectType = "remote"
)

type Project struct {
	Type         ProjectType `json:"type"`
	ID           string      `json:"id"`
	AbsolutePath string      `json:"absolute_path"`
}

func (p Project) Compare(other Project) int {
	return strings.Compare(p.AbsolutePath, other.AbsolutePath)
}

func newLocalProject(id, abs string) Project {
	return Project{
		Type:         Local,
		ID:           id,
		AbsolutePath: abs,
	}
}

func newRemoteProject(id, abs string) Project {
	return Project{
		Type:         Remote,
		ID:           id,
		AbsolutePath: abs,
	}
}

type ListOptions struct {
	Remote   bool
	NoCache  bool
	CacheDir string
}

// ListProjects will search for repositories using the given config and options.
//
// By default, it will only search for local repositories. To search for remote
// repositories, set opts.Remote to true.
func ListProjects(ctx context.Context, cfg Config, opts *ListOptions) ([]Project, error) {
	cfg = cfg.setDefaults()

	if opts == nil {
		opts = &ListOptions{}
	}

	if opts.CacheDir == "" {
		cacheDir := oslib.Expand("~/.cache")
		if os.Getenv("XDG_CACHE_DIR") != "" {
			cacheDir = os.Getenv("XDG_CACHE_DIR")
		}

		opts.CacheDir = filepath.Join(cacheDir, "z")
	}

	if !opts.NoCache {
		projects, err := loadProjectsFromCache(opts.CacheDir)
		if errors.Is(err, os.ErrNotExist) {
			return listProjects(ctx, cfg, opts)
		}
		if err != nil {
			return nil, err
		}

		return projects, nil
	}

	return listProjects(ctx, cfg, opts)
}

func listProjects(ctx context.Context, cfg Config, opts *ListOptions) ([]Project, error) {
	projects, err := listLocalProjects(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if opts.Remote {
		remoteProjects, err := listRemoteProjects(ctx, cfg)
		if err != nil {
			return nil, err
		}

		projects = append(projects, remoteProjects...)
	}

	projects = lo.UniqBy(projects, func(p Project) string {
		return p.AbsolutePath
	})

	slices.SortFunc(projects, func(a, b Project) int {
		return a.Compare(b)
	})

	if err := saveProjectsToCache(opts.CacheDir, projects); err != nil {
		return projects, err
	}

	return projects, nil
}

func listLocalProjects(ctx context.Context, cfg Config) ([]Project, error) {
	var (
		glob        = true
		hidden      = true
		maxDepth    = cfg.MaxDepth
		noIgnoreVCS = true
		root        = oslib.Expand(cfg.Root)
	)

	var projects []Project
	rr, err := fd.Run(
		ctx,
		".git",
		&fd.FdOptions{
			Glob:        &glob,
			Hidden:      &hidden,
			MaxDepth:    &maxDepth,
			NoIgnoreVCS: &noIgnoreVCS,
			Path:        &root,
		},
	)
	if err != nil {
		return nil, err
	}

	for _, r := range rr {
		abs := filepath.Dir(filepath.Clean(r))

		id, err := filepath.Rel(root, abs)
		if err != nil {
			return nil, fmt.Errorf("error convert absolute path to relative path %q: %w", r, err)
		}

		project := newLocalProject(id, abs)
		projects = append(projects, project)
	}

	return projects, nil
}

func listRemoteProjects(ctx context.Context, cfg Config) ([]Project, error) {
	var (
		projects = make([]Project, 0)
		root     = oslib.Expand(cfg.Root)
	)

	for _, rp := range cfg.RemotePatterns {
		pattern, err := parseRemotePattern(rp)
		if err != nil {
			return nil, fmt.Errorf("error parsing remote pattern %q: %w", rp, err)
		}

		var repos []*gh.Repo
		if pattern.Repo == nil {
			repos, err = gh.ListRepos(ctx, &gh.RepoListOptions{Owner: pattern.Owner})
			if err != nil {
				return nil, err
			}
		} else {
			repos = append(repos, &gh.Repo{Owner: pattern.Owner, Name: *pattern.Repo})
		}

		for _, r := range repos {
			abs := filepath.Join(root, r.String())
			if pattern.AlternatePath != nil {
				alt := *pattern.AlternatePath
				dir := r.String()

				if strings.HasSuffix(alt, "/") {
					dir = r.Name
				}

				abs = filepath.Join(root, alt, dir)
			}

			id, err := filepath.Rel(root, abs)
			if err != nil {
				return nil, fmt.Errorf("error convert absolute path to relative path %q: %w", abs, err)
			}

			project := newRemoteProject(id, abs)
			projects = append(projects, project)
		}
	}

	return projects, nil
}

type remotePattern struct {
	original string

	Owner         string
	Repo          *string
	AlternatePath *string
}

// parseRemotePattern will parse the pattern with the following format:
//
//	owner/repo -> ./alternate-path
//
// If the pattern is in the format above, the AlternatePath will be set.
func parseRemotePattern(pattern string) (remotePattern, error) {
	out := remotePattern{
		original: pattern,
	}

	// Check and parse if the pattern contains an alternate path.
	parts := strings.Split(pattern, "->")
	if len(parts) == 2 {
		alternatePath := strings.TrimSpace(parts[1])
		out.AlternatePath = &alternatePath
	}

	// Parse the owner/repo
	parts = strings.Split(strings.TrimSpace(parts[0]), "/")
	if len(parts) != 2 {
		return out, fmt.Errorf("invalid pattern: %q", pattern)
	}

	out.Owner = parts[0]
	out.Repo = &parts[1]

	if *out.Repo == "*" {
		out.Repo = nil
	}

	return out, nil
}

func loadProjectsFromCache(cacheDir string) ([]Project, error) {
	root, err := os.OpenRoot(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("error open cache dir %q: %w", cacheDir, err)
	}

	file, err := root.OpenFile(CACHE_FILE, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("error open projects file %q: %w", CACHE_FILE, err)
	}
	defer file.Close()

	var projects []Project
	if err := json.NewDecoder(file).Decode(&projects); err != nil {
		return nil, fmt.Errorf("error decoding projects file %q: %w", CACHE_FILE, err)
	}

	return projects, nil
}

func saveProjectsToCache(cacheDir string, projects []Project) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("error create cache dir %q: %w", cacheDir, err)
	}

	root, err := os.OpenRoot(cacheDir)
	if err != nil {
		return fmt.Errorf("error open cache dir %q: %w", cacheDir, err)
	}

	file, err := root.OpenFile(CACHE_FILE, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error open projects file %q: %w", CACHE_FILE, err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(projects); err != nil {
		return fmt.Errorf("error encoding projects file %q: %w", CACHE_FILE, err)
	}

	return nil
}
