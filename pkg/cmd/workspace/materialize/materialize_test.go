package materialize_test

import (
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
)

func TestMaterialize_missing_instance(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("missing")

	wstest.AssertErrorContains(t, err, "has not been created")
}
