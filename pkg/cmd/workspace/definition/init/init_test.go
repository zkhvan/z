package init_test

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	listCmd "github.com/zkhvan/z/pkg/cmd/workspace/definition/list"
	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestInit_scaffolds_a_definition(t *testing.T) {
	h := newCommandTest(t)

	if err := h.run("api-feature"); err != nil {
		t.Fatalf("init: %v", err)
	}

	h.DefinitionFileExists(filepath.Join("api-feature", ".z", "definition.yaml"))
	h.OutputContains("api-feature")
}

func TestInit_starter_manifest_keeps_its_comments(t *testing.T) {
	h := newCommandTest(t)

	if err := h.run("api-feature"); err != nil {
		t.Fatalf("init: %v", err)
	}

	contents := readDefinition(t, h, "api-feature")
	for _, want := range []string{
		"version: 1",
		`branch_pattern: "{instance}"`,
		"members:",
		"# Placeholders: {instance} = instance name, {repo} = member basename",
	} {
		if !strings.Contains(contents, want) {
			t.Fatalf("scaffolded manifest does not contain %q\ngot:\n%s", want, contents)
		}
	}
}

// A fresh scaffold has no members, so listing must not call it broken.
func TestInit_scaffold_lists_as_healthy(t *testing.T) {
	h := newCommandTest(t)

	if err := h.run("api-feature"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := h.Run(listCmd.NewCmdList); err != nil {
		t.Fatalf("list: %v", err)
	}

	// The summary column, not the whole line: the directory column holds a temp
	// path that embeds the test name.
	fields := strings.Split(strings.TrimRight(h.Output(), "\n"), "\t")
	if len(fields) < 3 {
		t.Fatalf("expected a definition row with 3 fields, got %q", h.Output())
	}
	if fields[2] != "{instance}" {
		t.Fatalf("scaffold summary = %q, want the default pattern", fields[2])
	}
}

func TestInit_scaffolds_example_hooks_for_every_phase(t *testing.T) {
	h := newCommandTest(t)

	if err := h.run("api-feature"); err != nil {
		t.Fatalf("init: %v", err)
	}

	phases := []string{
		"pre-create", "post-create",
		"pre-materialize", "post-materialize",
		"pre-archive", "post-archive",
		"pre-delete", "post-delete",
	}
	for _, phase := range phases {
		h.DefinitionFileExists(filepath.Join("api-feature", "hooks", phase+".example"))
	}
}

// The .example suffix keeps starters inert, but the executable bit must already
// be set so that enabling one is a rename, not a rename plus chmod.
func TestInit_example_hooks_are_executable_debug_starters(t *testing.T) {
	h := newCommandTest(t)

	if err := h.run("api-feature"); err != nil {
		t.Fatalf("init: %v", err)
	}

	path := filepath.Join(h.DefinitionsRoot(), "api-feature", "hooks", "post-create.example")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat example hook: %v", err)
	}
	if info.Mode()&0o100 == 0 {
		t.Fatalf("example hook is not executable: mode %v", info.Mode())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example hook: %v", err)
	}
	for _, want := range []string{"Z_HOOK_PHASE", "[z hook] phase=", "mv post-create.example post-create"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("example hook missing %q\ngot:\n%s", want, data)
		}
	}
}

// The orbstack template must land a complete, live runtime: four executable
// scripts that drive orb, a cloud-init seed, and a pre-delete hook that removes
// the machine when the workspace is deleted.
func TestInit_runtime_orbstack_scaffolds_the_contract(t *testing.T) {
	h := newCommandTest(t)

	if err := h.run("api-feature", "--runtime", "orbstack"); err != nil {
		t.Fatalf("init: %v", err)
	}

	for _, verb := range []string{"up", "exec", "down", "status"} {
		rel := filepath.Join("api-feature", "runtime", verb)
		h.DefinitionFileExists(rel)
		path := filepath.Join(h.DefinitionsRoot(), rel)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat runtime/%s: %v", verb, err)
		}
		if info.Mode()&0o100 == 0 {
			t.Fatalf("runtime/%s is not executable: mode %v", verb, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read runtime/%s: %v", verb, err)
		}
		if !strings.Contains(string(data), "orb") {
			t.Fatalf("runtime/%s does not drive orb:\n%s", verb, data)
		}
	}

	h.DefinitionFileExists(filepath.Join("api-feature", "runtime", "user-data.yml"))

	hookPath := filepath.Join(h.DefinitionsRoot(), "api-feature", "hooks", "pre-delete")
	hookData, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read pre-delete hook: %v", err)
	}
	if !strings.Contains(string(hookData), "orb delete") {
		t.Fatalf("pre-delete hook does not remove the machine:\n%s", hookData)
	}
}

func TestInit_without_runtime_scaffolds_no_runtime(t *testing.T) {
	h := newCommandTest(t)

	if err := h.run("api-feature"); err != nil {
		t.Fatalf("init: %v", err)
	}

	if _, err := os.Stat(filepath.Join(h.DefinitionsRoot(), "api-feature", "runtime")); !os.IsNotExist(err) {
		t.Fatalf("expected no runtime directory, stat error: %v", err)
	}
}

func TestInit_unknown_runtime_is_refused(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("api-feature", "--runtime", "bogus")

	wstest.AssertErrorContains(t, err, "unknown runtime")
	wstest.AssertErrorContains(t, err, "orbstack")
	h.NoDefinition("api-feature")
}

func TestInit_existing_definition_is_refused(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})

	err := h.run("api-feature")

	wstest.AssertErrorContains(t, err, "already exists")
}

func TestInit_invalid_name_is_refused(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("nested/name")

	if !errors.Is(err, workspace.ErrInvalidName) {
		t.Fatalf("error %v does not wrap ErrInvalidName", err)
	}
	wstest.AssertErrorContains(t, err, "definition name")
	h.NoDefinition("nested/name")
}

func TestInit_missing_name_arg(t *testing.T) {
	h := newCommandTest(t)

	err := h.run()

	wstest.AssertErrorContains(t, err, "accepts 1 arg")
}

func readDefinition(t *testing.T, h *harness, name string) string {
	t.Helper()
	path := filepath.Join(h.DefinitionsRoot(), name, ".z", "definition.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// Uncommenting the scaffolded example must produce a usable definition, not a
// parse error: "members: []" followed by indented block items is invalid YAML.
func TestInit_uncommenting_the_example_yields_a_usable_definition(t *testing.T) {
	h := newCommandTest(t)

	if err := h.run("api-feature"); err != nil {
		t.Fatalf("init: %v", err)
	}

	path := filepath.Join(h.DefinitionsRoot(), "api-feature", ".z", "definition.yaml")
	uncommented := regexp.MustCompile(`(?m)^(\s*)#\s?(- repo:|  base_ref:)`).
		ReplaceAllString(readDefinition(t, h, "api-feature"), "$1$2")
	if err := os.WriteFile(path, []byte(uncommented), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	if err := h.Run(listCmd.NewCmdList); err != nil {
		t.Fatalf("list: %v", err)
	}

	if strings.Contains(h.Output(), "broken:") {
		t.Fatalf("uncommenting the example broke the definition:\n%s\n---\n%s", h.Output(), uncommented)
	}
	h.OutputContains("acme/api")
	h.OutputContains("acme/ui")
}
