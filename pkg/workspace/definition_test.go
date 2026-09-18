package workspace_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/workspace"
)

// The default follows the config dir the provider actually loaded, so a test or
// a Z_CONFIG_DIR override cannot reach the real one.
func TestDefinitionsRoot_defaults_beside_the_config_file(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	dir, err := svc.InitDefinition(context.Background(), "api-feature", workspace.InitDefinitionOptions{})
	assert.NoError(t, err)

	want := filepath.Join(td.configDir, "definitions", "api-feature")
	if dir != want {
		t.Fatalf("definition dir = %q, want %q", dir, want)
	}
}

func TestDefinitionsRoot_is_configurable(t *testing.T) {
	td := setupServiceTestDir(t)
	elsewhere := filepath.Join(td.root, "elsewhere")
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
  definitions_root: `+elsewhere+`
`)

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	dir, err := svc.InitDefinition(context.Background(), "api-feature", workspace.InitDefinitionOptions{})
	assert.NoError(t, err)

	if want := filepath.Join(elsewhere, "api-feature"); dir != want {
		t.Fatalf("definition dir = %q, want %q", dir, want)
	}
}

func TestInitDefinition_scaffold_is_readable_and_not_broken(t *testing.T) {
	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)

	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)
	_, err = svc.InitDefinition(context.Background(), "api-feature", workspace.InitDefinitionOptions{})
	assert.NoError(t, err)

	def, err := svc.Definition("api-feature")
	assert.NoError(t, err)

	if def.Broken() {
		t.Fatalf("scaffolded definition is broken: %v", def.Err)
	}
	if def.BranchPattern != workspace.DefaultBranchPattern {
		t.Fatalf("branch pattern = %q, want %q", def.BranchPattern, workspace.DefaultBranchPattern)
	}
	if len(def.Members) != 0 {
		t.Fatalf("scaffold has %d members, want 0", len(def.Members))
	}
}
