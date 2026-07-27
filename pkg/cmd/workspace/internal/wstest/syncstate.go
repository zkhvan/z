package wstest

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

// syncState is deliberately independent of the domain type, so a schema change
// breaks these assertions instead of following along silently.
type syncState struct {
	Version int `yaml:"version"`
	Files   []struct {
		Path string `yaml:"path"`
		Hash string `yaml:"hash"`
		Exec bool   `yaml:"exec"`
	} `yaml:"files"`
}

type SyncStateAssert struct {
	t     *testing.T
	name  string
	state syncState
}

// SeedDefinitionFile writes definition-authored content, the material sync
// propagates.
func (h *Harness) SeedDefinitionFile(definition, relPath, content string) {
	h.t.Helper()
	h.seedUnder(h.definitionsRoot, filepath.Join(definition, relPath), content)
}

// SeedDefinitionExecFile writes definition content that must stay executable,
// which is the property hooks and runtime verbs depend on.
func (h *Harness) SeedDefinitionExecFile(definition, relPath, content string) {
	h.t.Helper()
	h.SeedDefinitionFile(definition, relPath, content)
	path := filepath.Join(h.definitionsRoot, definition, filepath.FromSlash(relPath))
	if err := os.Chmod(path, 0o700); err != nil {
		h.t.Fatalf("chmod %q: %v", relPath, err)
	}
}

// RemoveDefinitionFile drops content from a definition between syncs.
func (h *Harness) RemoveDefinitionFile(definition, relPath string) {
	h.t.Helper()
	path := filepath.Join(h.definitionsRoot, definition, filepath.FromSlash(relPath))
	if err := os.Remove(path); err != nil {
		h.t.Fatalf("remove %q: %v", relPath, err)
	}
}

// RemoveFile deletes instance content, which is also how a user resolves a
// conflict in favor of the definition.
func (h *Harness) RemoveFile(relPath string) {
	h.t.Helper()
	if err := os.Remove(filepath.Join(h.root, filepath.FromSlash(relPath))); err != nil {
		h.t.Fatalf("remove %q: %v", relPath, err)
	}
}

// FileContains asserts the exact content of an instance file.
func (h *Harness) FileContains(relPath, want string) {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.root, filepath.FromSlash(relPath)))
	if err != nil {
		h.t.Fatalf("read %s: %v", relPath, err)
	}
	if string(data) != want {
		h.t.Fatalf("%s = %q, want %q", relPath, string(data), want)
	}
}

// FileIsExecutable asserts an instance file kept its executable bit.
func (h *Harness) FileIsExecutable(relPath string) {
	h.t.Helper()
	info, err := os.Stat(filepath.Join(h.root, filepath.FromSlash(relPath)))
	if err != nil {
		h.t.Fatalf("stat %s: %v", relPath, err)
	}
	if info.Mode()&0o100 == 0 {
		h.t.Fatalf("%s mode = %v, want executable", relPath, info.Mode())
	}
}

// SyncState reads and parses an instance's recorded sync state.
func (h *Harness) SyncState(name string) *SyncStateAssert {
	h.t.Helper()

	path := filepath.Join(h.root, name, ".z", "sync-state.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		h.t.Fatalf("workspace %q: read sync state: %v", name, err)
	}

	var s syncState
	if err := yaml.Unmarshal(data, &s); err != nil {
		h.t.Fatalf("workspace %q: parse sync state: %v", name, err)
	}

	return &SyncStateAssert{t: h.t, name: name, state: s}
}

// NoSyncState asserts nothing was recorded, which is what a dry run must leave
// behind.
func (h *Harness) NoSyncState(name string) {
	h.t.Helper()
	path := filepath.Join(h.root, name, ".z", "sync-state.yaml")
	if _, err := os.Stat(path); err == nil {
		h.t.Fatalf("workspace %q unexpectedly has sync state", name)
	}
}

func (s *SyncStateAssert) HasVersion(want int) *SyncStateAssert {
	s.t.Helper()
	if s.state.Version != want {
		s.t.Fatalf("workspace %q: sync state version = %d, want %d", s.name, s.state.Version, want)
	}
	return s
}

func (s *SyncStateAssert) FileCount(want int) *SyncStateAssert {
	s.t.Helper()
	if len(s.state.Files) != want {
		s.t.Fatalf("workspace %q: recorded %d file(s), want %d: %+v",
			s.name, len(s.state.Files), want, s.state.Files)
	}
	return s
}

func (s *SyncStateAssert) Records(path string) *SyncStateAssert {
	s.t.Helper()
	if s.find(path) == nil {
		s.t.Fatalf("workspace %q: %s is not recorded; recorded: %+v", s.name, path, s.state.Files)
	}
	return s
}

func (s *SyncStateAssert) DoesNotRecord(path string) *SyncStateAssert {
	s.t.Helper()
	if s.find(path) != nil {
		s.t.Fatalf("workspace %q: %s is still recorded", s.name, path)
	}
	return s
}

func (s *SyncStateAssert) RecordsExecutable(path string) *SyncStateAssert {
	s.t.Helper()
	f := s.find(path)
	if f == nil {
		s.t.Fatalf("workspace %q: %s is not recorded", s.name, path)
	}
	if !f.Exec {
		s.t.Fatalf("workspace %q: %s recorded as non-executable", s.name, path)
	}
	return s
}

// HashChanged asserts a recorded hash moved, which is how "the ancestor
// advanced" is observed from outside.
func (s *SyncStateAssert) HashOf(path string) string {
	s.t.Helper()
	f := s.find(path)
	if f == nil {
		s.t.Fatalf("workspace %q: %s is not recorded", s.name, path)
	}
	return f.Hash
}

func (s *SyncStateAssert) find(path string) *struct {
	Path string `yaml:"path"`
	Hash string `yaml:"hash"`
	Exec bool   `yaml:"exec"`
} {
	for i := range s.state.Files {
		if s.state.Files[i].Path == path {
			return &s.state.Files[i]
		}
	}
	return nil
}

// SeedDefinitionSymlink plants a symlink in a definition, the content sync must
// refuse to follow.
func (h *Harness) SeedDefinitionSymlink(definition, relPath, target string) {
	h.t.Helper()
	path := filepath.Join(h.definitionsRoot, definition, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		h.t.Fatalf("seed parent of %q: %v", relPath, err)
	}
	if err := os.Symlink(target, path); err != nil {
		h.t.Fatalf("symlink %q: %v", relPath, err)
	}
}
