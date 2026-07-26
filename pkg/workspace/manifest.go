package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const manifestRelPath = ".z/instance.yaml"
const manifestVersion = 1

// manifest is the on-disk instance file. Name and status are never stored;
// they are derived from disk.
type manifest struct {
	Version int              `yaml:"version"`
	Members []manifestMember `yaml:"members"`
}

type manifestMember struct {
	Repo    string `yaml:"repo"`
	Branch  string `yaml:"branch"`
	BaseRef string `yaml:"base_ref,omitempty"`
}

func manifestMemberFromMember(m Member) manifestMember {
	return manifestMember{Repo: m.Repo, Branch: m.Branch, BaseRef: m.BaseRef}
}

func memberFromManifest(mm manifestMember) Member {
	return Member{Repo: mm.Repo, Branch: mm.Branch, BaseRef: mm.BaseRef}
}

// writeManifest writes m atomically (temp file + rename) to dir/.z/instance.yaml.
func writeManifest(dir string, m manifest) error {
	dotZ := filepath.Join(dir, ".z")
	if err := os.MkdirAll(dotZ, 0o700); err != nil {
		return fmt.Errorf("creating .z directory: %w", err)
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	tmp, err := os.CreateTemp(dotZ, "instance.*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp manifest: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing temp manifest: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("closing temp manifest: %w", err)
	}

	dest := filepath.Join(dotZ, "instance.yaml")
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("renaming manifest: %w", err)
	}

	return nil
}

func readManifest(dir string) (manifest, error) {
	path := filepath.Join(dir, manifestRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, err
	}
	var m manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return manifest{}, fmt.Errorf("parsing manifest %s: %w", path, err)
	}
	return m, nil
}
