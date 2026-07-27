package sync_test

import (
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	syncCmd "github.com/zkhvan/z/pkg/cmd/workspace/sync"
	"github.com/zkhvan/z/pkg/workspace"
)

type harness struct{ *wstest.Harness }

func newCommandTest(t *testing.T) *harness {
	t.Helper()
	return &harness{wstest.New(t)}
}

func (h *harness) run(args ...string) error {
	return h.Run(syncCmd.NewCmdSync, args...)
}

// seedInstanceFromDefinition arranges an instance that came from the "feature"
// definition without running sync, so a test can drive the first sync itself.
func (h *harness) seedInstanceFromDefinition(instance string, members ...workspace.Member) {
	h.SeedDefinition(definitionName, "{instance}", members...)
	h.SeedManifest(instance, manifestFor(definitionName, members))
}

const definitionName = "feature"

func manifestFor(definition string, members []workspace.Member) string {
	raw := "version: 1\ndefinition: " + definition + "\nmembers:\n"
	for _, m := range members {
		raw += "  - repo: " + m.Repo + "\n    branch: " + m.Branch + "\n"
	}
	if len(members) == 0 {
		raw += "  []\n"
	}
	return raw
}
