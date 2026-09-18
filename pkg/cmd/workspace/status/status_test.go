package status_test

import (
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

// status must surface both the script's output and its exit code: a non-zero
// exit is a legitimate status ("not running"), not a z error.
func TestStatus_reports_output_and_exit_code(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	h.SeedInstanceRuntimeScript("login", workspace.RuntimeStatus,
		"#!/bin/sh\necho 'not running'\nexit 3\n")

	err := h.run("login")

	h.OutputContains("not running")
	wstest.AssertExitCode(t, err, 3)
}

func TestStatus_zero_exit_is_success(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	h.SeedInstanceRuntimeScript("login", workspace.RuntimeStatus, "#!/bin/sh\necho running\nexit 0\n")

	if err := h.run("login"); err != nil {
		t.Fatalf("status: %v", err)
	}

	h.OutputContains("running")
}
