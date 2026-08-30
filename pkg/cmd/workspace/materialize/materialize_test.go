package materialize_test

import (
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestMaterialize_omitted_name_outside_the_workspaces_root(t *testing.T) {
	h := newCommandTest(t)

	err := h.run()

	wstest.AssertErrorContains(t, err, "is not inside a workspace under")
}

// A directory under the root without a manifest is still a name, so inference
// reaches the service and fails there rather than refusing to infer.
func TestMaterialize_name_from_the_current_directory(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDir("scratch")
	h.InDir("scratch")

	err := h.run()

	wstest.AssertErrorContains(t, err, `workspace "scratch" has not been created`)
}

func TestMaterialize_missing_instance(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("missing")

	wstest.AssertErrorContains(t, err, "has not been created")
}

func TestMaterialize_pre_materialize_hook_failure_aborts(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstanceFromDefinition("login", "feature", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
	h.SeedDefinitionHook("feature", workspace.HookPreMaterialize, "#!/bin/sh\nexit 1\n")

	err := h.run("login")

	wstest.AssertErrorContains(t, err, "pre-materialize hook failed")
	h.PathMissing(filepath.Join("login", ".z", "materialized"))
}
