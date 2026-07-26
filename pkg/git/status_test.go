package git_test

import (
	"context"
	"testing"

	"github.com/zkhvan/z/pkg/exec"
	testingexec "github.com/zkhvan/z/pkg/exec/testing"
	"github.com/zkhvan/z/pkg/git"
)

func TestStatusPorcelain_returns_porcelain_output(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/workspace/repo", "status", "--porcelain")
		cmd.OutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return []byte(" M file.txt\n"), nil, nil },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	status, err := client.StatusPorcelain(context.Background(), "/workspace/repo")
	if err != nil {
		t.Fatalf("StatusPorcelain returned error: %v", err)
	}
	if status != " M file.txt" {
		t.Fatalf("expected trimmed status, got %q", status)
	}
	assertCommandCalls(t, fake, 1)
}
