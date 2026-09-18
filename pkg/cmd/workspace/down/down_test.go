package down_test

import (
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestDown_invokes_runtime_down(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})
	h.SeedInstanceRuntimeScript("login", workspace.RuntimeDown, "#!/bin/sh\ntouch went.down\n")

	if err := h.run("login"); err != nil {
		t.Fatalf("down: %v", err)
	}

	h.FileExists(filepath.Join("login", "went.down"))
}

func TestDown_missing_runtime_is_a_clear_error(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("login", workspace.Member{Repo: "acme/api", Branch: "feat/login"})

	err := h.run("login")

	wstest.AssertErrorContains(t, err, "no runtime configured")
}
