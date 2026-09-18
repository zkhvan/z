package exec_test

import (
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

// The command tokens after -- must reach runtime/exec as separate positional
// arguments, not a joined string, so boundaries and quoting survive.
func TestExec_passes_command_args_as_argv(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	h.SeedInstanceRuntimeScript("login", workspace.RuntimeExec,
		"#!/bin/sh\nprintf '%s\\n' \"$@\" > argv.out\n")

	if err := h.run("login", "--", "alpha", "beta", "gamma"); err != nil {
		t.Fatalf("exec: %v", err)
	}

	h.FileContains(filepath.Join("login", "argv.out"), "alpha\nbeta\ngamma\n")
}

func TestExec_passes_stdin_through(t *testing.T) {
	h := newCommandTest(t)
	h.WithStdin("hello from stdin\n")
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	h.SeedInstanceRuntimeScript("login", workspace.RuntimeExec, "#!/bin/sh\ncat > stdin.out\n")

	if err := h.run("login", "--", "cat"); err != nil {
		t.Fatalf("exec: %v", err)
	}

	h.FileContains(filepath.Join("login", "stdin.out"), "hello from stdin\n")
}

func TestExec_surfaces_the_command_exit_code(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	h.SeedInstanceRuntimeScript("login", workspace.RuntimeExec, "#!/bin/sh\nexit 7\n")

	err := h.run("login", "--", "whatever")

	wstest.AssertExitCode(t, err, 7)
}

func TestExec_defaults_instance_to_the_cwd_workspace(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	h.SeedInstanceRuntimeScript("login", workspace.RuntimeExec, "#!/bin/sh\ntouch ran.marker\n")
	h.InDir("login")

	if err := h.run("--", "cmd"); err != nil {
		t.Fatalf("exec: %v", err)
	}

	h.FileExists(filepath.Join("login", "ran.marker"))
}

func TestExec_requires_a_command_after_dashdash(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})

	err := h.run("login")

	wstest.AssertErrorContains(t, err, "requires a command after --")
}

func TestExec_empty_command_after_dashdash_is_refused(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})

	err := h.run("login", "--")

	wstest.AssertErrorContains(t, err, "no command given after --")
}

func TestExec_extra_args_before_dashdash_are_refused(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("one", "two", "--", "cmd")

	wstest.AssertErrorContains(t, err, "unexpected arguments before --")
}
