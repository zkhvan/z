package materialize_test

import (
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
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
