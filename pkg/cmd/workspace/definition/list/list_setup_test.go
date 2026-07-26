package list_test

import (
	"testing"

	listCmd "github.com/zkhvan/z/pkg/cmd/workspace/definition/list"
	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
)

type harness struct{ *wstest.Harness }

func newCommandTest(t *testing.T) *harness {
	t.Helper()
	return &harness{wstest.New(t)}
}

// list takes no arguments or flags of its own.
func (h *harness) run() error {
	return h.Run(listCmd.NewCmdList)
}
