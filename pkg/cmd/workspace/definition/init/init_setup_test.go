package init_test

import (
	"testing"

	initCmd "github.com/zkhvan/z/pkg/cmd/workspace/definition/init"
	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
)

type harness struct{ *wstest.Harness }

func newCommandTest(t *testing.T) *harness {
	t.Helper()
	return &harness{wstest.New(t)}
}

func (h *harness) run(args ...string) error {
	return h.Run(initCmd.NewCmdInit, args...)
}
