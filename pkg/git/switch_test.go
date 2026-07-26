package git_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/exec"
	testingexec "github.com/zkhvan/z/pkg/exec/testing"
	"github.com/zkhvan/z/pkg/git"
)

func TestSwitch_existing_branch(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/workspace/repo", "switch", "--", "feature/login")
		cmd.CombinedOutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return nil, nil, nil },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	err := client.Switch(context.Background(), git.SwitchOptions{
		WorktreePath: "/workspace/repo",
		Branch:       "feature/login",
	})
	if err != nil {
		t.Fatalf("Switch returned error: %v", err)
	}
	assertCommandCalls(t, fake, 1)
}

func TestSwitch_creates_branch_from_base_ref(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/workspace/repo", "switch", "-c", "feature/two", "--", "feature/one")
		cmd.CombinedOutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return nil, nil, nil },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	err := client.Switch(context.Background(), git.SwitchOptions{
		WorktreePath: "/workspace/repo",
		Branch:       "feature/two",
		BaseRef:      "feature/one",
		CreateBranch: true,
	})
	if err != nil {
		t.Fatalf("Switch returned error: %v", err)
	}
	assertCommandCalls(t, fake, 1)
}

func TestSwitch_creates_branch_from_head_when_base_ref_is_empty(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/workspace/repo", "switch", "-c", "feature/two")
		cmd.CombinedOutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return nil, nil, nil },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	err := client.Switch(context.Background(), git.SwitchOptions{
		WorktreePath: "/workspace/repo",
		Branch:       "feature/two",
		CreateBranch: true,
	})
	if err != nil {
		t.Fatalf("Switch returned error: %v", err)
	}
	assertCommandCalls(t, fake, 1)
}

func TestSwitch_reports_git_output_on_failure(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/workspace/repo", "switch", "--", "feature/login")
		cmd.CombinedOutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) {
				return []byte("error: Your local changes would be overwritten\n"), nil, fakeExitError(1)
			},
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	err := client.Switch(context.Background(), git.SwitchOptions{
		WorktreePath: "/workspace/repo",
		Branch:       "feature/login",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "local changes would be overwritten") {
		t.Fatalf("git diagnostics missing from error: %v", err)
	}
}
