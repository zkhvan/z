package up_test

import (
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

// The runtime script runs from the instance copy with the documented env and
// cwd. Writing to a relative path proves cwd is the instance directory; the
// recorded values prove the environment.
func TestUp_runs_runtime_up_with_env_and_cwd(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstanceFromDefinition("login", "api-feature",
		workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	h.SeedInstanceRuntimeScript("login", workspace.RuntimeUp,
		"#!/bin/sh\nprintf '%s\\n%s\\n%s\\n' \"$Z_INSTANCE_PATH\" \"$Z_INSTANCE_NAME\" \"$Z_DEFINITION_PATH\" > ran.out\n")

	if err := h.run("login"); err != nil {
		t.Fatalf("up: %v", err)
	}

	instanceDir := filepath.Join(h.Root(), "login")
	h.FileContains(filepath.Join("login", "ran.out"),
		instanceDir+"\nlogin\n"+h.DefinitionDir("api-feature")+"\n")
}

func TestUp_defaults_to_the_cwd_workspace(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	h.SeedInstanceRuntimeScript("login", workspace.RuntimeUp, "#!/bin/sh\ntouch ran.marker\n")
	h.InDir("login")

	if err := h.run(); err != nil {
		t.Fatalf("up: %v", err)
	}

	h.FileExists(filepath.Join("login", "ran.marker"))
}

func TestUp_missing_runtime_is_a_clear_error(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})

	err := h.run("login")

	wstest.AssertErrorContains(t, err, "no runtime configured")
	wstest.AssertErrorContains(t, err, filepath.Join("login", "runtime", "up"))
}

func TestUp_non_executable_runtime_is_an_error(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	// SeedFile writes without the executable bit.
	h.SeedFile(filepath.Join("login", "runtime", "up"), "#!/bin/sh\n")

	err := h.run("login")

	wstest.AssertErrorContains(t, err, "not executable")
}

func TestUp_unknown_workspace_is_refused(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("ghost")

	wstest.AssertErrorContains(t, err, "has not been created")
}

func TestUp_surfaces_the_runtime_exit_code(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	h.SeedInstanceRuntimeScript("login", workspace.RuntimeUp, "#!/bin/sh\nexit 5\n")

	err := h.run("login")

	wstest.AssertExitCode(t, err, 5)
}
