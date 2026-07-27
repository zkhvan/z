package wssync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// StateRelPath is where an instance records what z last wrote into it.
const StateRelPath = ".z/sync-state.yaml"

const stateVersion = 1

// stateFile is the serialized ancestor. Only files are written: directories are
// implied by the paths and rebuilt on load, which keeps the file readable and
// matches git's inability to carry an empty directory anyway.
type stateFile struct {
	Version int         `yaml:"version"`
	Files   []stateData `yaml:"files"`
}

type stateData struct {
	Path string `yaml:"path"`
	Hash string `yaml:"hash"`
	Exec bool   `yaml:"exec,omitempty"`
}

// LoadState reads the recorded tree. A missing file is not an error: an
// instance that has never synced has an absent ancestor, which is exactly what
// reconciliation expects on a first run.
func LoadState(instanceDir string) (*Entry, error) {
	path := filepath.Join(instanceDir, StateRelPath)

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading sync state: %w", err)
	}

	var sf stateFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	// Reading a newer state leniently would drop fields it does not know about
	// — including, once ZK-16 lands, the per-file policy that decides whether a
	// file may be overwritten at all. Refusing is the safe half of the manifest
	// rule "read lenient, write strict".
	if sf.Version > stateVersion {
		return nil, fmt.Errorf(
			"sync state at %s is version %d; this z understands up to %d",
			path, sf.Version, stateVersion,
		)
	}

	root := &Entry{Kind: KindDirectory}
	for _, f := range sf.Files {
		root = Apply(root, []Change{{
			Path: f.Path,
			New:  &Entry{Kind: KindFile, Hash: f.Hash, Exec: f.Exec},
		}})
	}
	return root, nil
}

// SaveState writes the recorded tree atomically.
func SaveState(instanceDir string, root *Entry) error {
	dotZ := filepath.Join(instanceDir, ".z")
	if err := os.MkdirAll(dotZ, 0o700); err != nil {
		return fmt.Errorf("creating .z directory: %w", err)
	}

	data, err := yaml.Marshal(stateFile{Version: stateVersion, Files: flatten("", root)})
	if err != nil {
		return fmt.Errorf("marshaling sync state: %w", err)
	}

	tmp, err := os.CreateTemp(dotZ, "sync-state.*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp sync state: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing temp sync state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("closing temp sync state: %w", err)
	}

	if err := os.Rename(tmpName, filepath.Join(instanceDir, StateRelPath)); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing sync state: %w", err)
	}
	return nil
}

// StatePaths lists every recorded file path, sorted. Teardown uses it to tell
// recoverable synced content from content that exists nowhere else.
func StatePaths(root *Entry) []string {
	entries := flatten("", root)
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	return paths
}

func flatten(path string, e *Entry) []stateData {
	if e == nil {
		return nil
	}
	if e.Kind == KindFile {
		return []stateData{{Path: path, Hash: e.Hash, Exec: e.Exec}}
	}
	var out []stateData
	for _, name := range sortedNames(e.Contents) {
		out = append(out, flatten(join(path, name), e.Contents[name])...)
	}
	return out
}
