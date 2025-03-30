package exec_test

import (
	"errors"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/exec"
)

func TestExec_CombinedOutput(t *testing.T) {
	tests := map[string]struct {
		cmd  string
		args []string
		err  error
		out  string
	}{
		"no arguments": {
			cmd:  "true",
			args: nil,
			err:  nil,
			out:  "",
		},
		"no arguments with exit status 1": {
			cmd:  "false",
			args: nil,
			err:  errors.New("exit status 1"),
			out:  "",
		},
		"with fake command": {
			cmd:  "/does/not/exist",
			args: nil,
			err:  errors.New("fork/exec /does/not/exist: no such file or directory"),
			out:  "",
		},
		"with arguments": {
			cmd:  "echo",
			args: []string{"hello", "world"},
			err:  nil,
			out:  "hello world\n",
		},
		"with stderr output": {
			cmd:  "/bin/sh",
			args: []string{"-c", "echo 'stderr' >&2"},
			err:  nil,
			out:  "stderr\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			exec := exec.New()

			cmd := exec.Command(test.cmd, test.args...)
			out, err := cmd.CombinedOutput()

			if test.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err, test.err)
			}

			assert.EqualString(t, string(out), test.out)
		})
	}

}

func TestStop(t *testing.T) {
	fakeexec := exec.New()
	cmd := fakeexec.Command("true")

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()

	cmd.Stop()
	_ = cmd.Run()
	cmd.Stop()
}
