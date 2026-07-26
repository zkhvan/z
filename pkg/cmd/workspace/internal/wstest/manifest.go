package wstest

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

// instanceManifest is deliberately independent of the domain type, so a schema
// change breaks these assertions instead of following along silently.
type instanceManifest struct {
	Version int `yaml:"version"`
	Members []struct {
		Repo    string `yaml:"repo"`
		Branch  string `yaml:"branch"`
		BaseRef string `yaml:"base_ref"`
	} `yaml:"members"`
}

type ManifestAssert struct {
	t        *testing.T
	name     string
	manifest instanceManifest
}

// Workspace reads and parses the manifest for name, ready for assertions.
func (h *Harness) Workspace(name string) *ManifestAssert {
	h.t.Helper()

	path := filepath.Join(h.root, name, ".z", "instance.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		h.t.Fatalf("workspace %q: read manifest: %v", name, err)
	}

	var m instanceManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		h.t.Fatalf("workspace %q: parse manifest: %v", name, err)
	}

	return &ManifestAssert{t: h.t, name: name, manifest: m}
}

func (m *ManifestAssert) HasVersion(want int) *ManifestAssert {
	m.t.Helper()
	if m.manifest.Version != want {
		m.t.Fatalf("workspace %q: version = %d, want %d", m.name, m.manifest.Version, want)
	}
	return m
}

func (m *ManifestAssert) MemberCount(want int) *ManifestAssert {
	m.t.Helper()
	if len(m.manifest.Members) != want {
		m.t.Fatalf("workspace %q: member count = %d, want %d", m.name, len(m.manifest.Members), want)
	}
	return m
}

func (m *ManifestAssert) HasMember(repo, branch, baseRef string) *ManifestAssert {
	m.t.Helper()
	for _, member := range m.manifest.Members {
		if member.Repo == repo && member.Branch == branch && member.BaseRef == baseRef {
			return m
		}
	}
	m.t.Fatalf("workspace %q: member {repo:%q branch:%q base_ref:%q} not found in %+v",
		m.name, repo, branch, baseRef, m.manifest.Members)
	return m
}
