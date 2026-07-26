package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const definitionManifestRelPath = ".z/definition.yaml"
const definitionVersion = 1

// Definition is a reusable template discovered on disk. Name is the directory
// name; it is never stored.
type Definition struct {
	Name          string
	Dir           string
	BranchPattern string
	Members       []Member // Branch is always empty; the pattern supplies it

	// Err is set when the directory holds a definition manifest that cannot be
	// used. The definition is still reported, because the marker file is
	// present and the user meant to author one.
	Err error
}

// Broken reports whether the definition can be instantiated.
func (d Definition) Broken() bool {
	return d.Err != nil
}

type definitionManifest struct {
	Version       int                        `yaml:"version"`
	BranchPattern string                     `yaml:"branch_pattern,omitempty"`
	Members       []definitionManifestMember `yaml:"members"`
}

type definitionManifestMember struct {
	Repo    string `yaml:"repo"`
	BaseRef string `yaml:"base_ref,omitempty"`
}

// definitionTemplate is written as raw bytes rather than marshaled, so the
// comments explaining the placeholders survive into the scaffolded file.
const definitionTemplate = `# z workspace definition
version: %d

# Branch name for each member at create time.
# Placeholders: {instance} = instance name, {repo} = member basename
branch_pattern: "{instance}"

# Uncommenting an example below is all it takes to add a member.
members:
  # - repo: acme/api
  #   base_ref: main         # optional; defaults to the repo's default branch
  # - repo: acme/ui
`

// readDefinition loads one definition. A manifest that parses but cannot be
// instantiated is returned with Err set rather than as a failure, so listing
// can report it in place.
func readDefinition(root, name string) (Definition, error) {
	dir := filepath.Join(root, name)
	path := filepath.Join(dir, definitionManifestRelPath)

	data, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, err
	}

	d := Definition{Name: name, Dir: dir}

	var dm definitionManifest
	if err := yaml.Unmarshal(data, &dm); err != nil {
		d.Err = fmt.Errorf("parsing %s: %w", path, err)
		return d, nil
	}

	d.BranchPattern = dm.BranchPattern
	if d.BranchPattern == "" {
		d.BranchPattern = DefaultBranchPattern
	}

	d.Members = make([]Member, len(dm.Members))
	for i, dmm := range dm.Members {
		d.Members[i] = Member{Repo: dmm.Repo, BaseRef: dmm.BaseRef}
	}

	if err := ValidateBranchPattern(d.BranchPattern); err != nil {
		d.Err = err
		return d, nil
	}

	// A definition with no members is a fresh scaffold, not a defect; only
	// create refuses it.
	if len(d.Members) > 0 {
		if err := validateDefinitionMembers(d.Members); err != nil {
			d.Err = err
			return d, nil
		}
	}

	return d, nil
}
