package create_test

import (
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/create"
	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
)

type harness struct{ *wstest.Harness }

func newCommandTest(t *testing.T) *harness {
	t.Helper()
	return &harness{wstest.New(t)}
}

func (h *harness) run(args ...string) error {
	return h.Run(create.NewCmdCreate, args...)
}
