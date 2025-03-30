package testingexec_test

import (
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/exec"
	testingexec "github.com/zkhvan/z/pkg/exec/testing"
)

func TestFakeExec_Command(t *testing.T) {
	fakeExec := &testingexec.FakeExec{}
	fakeExec.CommandScript = []testingexec.FakeCommandAction{
		func(_ string, _ ...string) exec.Cmd {
			fakeCmd := testingexec.NewFakeCmd("cat", "/var/log")
			fakeCmd.CombinedOutputScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) {
					return []byte("hello world\n"), nil, nil
				},
			}
			return fakeCmd
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()

	fakeCmd := fakeExec.Command("cat", "/var/log")
	out, err := fakeCmd.CombinedOutput()
	assert.NoError(t, err)
	assert.EqualString(t, "hello world\n", string(out))
}
