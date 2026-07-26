package git_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/exec"
	testingexec "github.com/zkhvan/z/pkg/exec/testing"
	"github.com/zkhvan/z/pkg/git"
)

func TestBranchExists_true_when_show_ref_succeeds(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "show-ref", "--verify", "--quiet", "refs/heads/feature/login")
		cmd.RunScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return nil, nil, nil },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	exists, err := client.BranchExists(context.Background(), "/repo", "feature/login")
	if err != nil {
		t.Fatalf("BranchExists returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected branch to exist")
	}
	assertCommandCalls(t, fake, 1)
}

func TestBranchExists_false_when_show_ref_fails(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "show-ref", "--verify", "--quiet", "refs/heads/feature/login")
		cmd.RunScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return nil, nil, fakeExitError(1) },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	exists, err := client.BranchExists(context.Background(), "/repo", "feature/login")
	if err != nil {
		t.Fatalf("BranchExists returned error: %v", err)
	}
	if exists {
		t.Fatal("expected branch to be missing")
	}
	assertCommandCalls(t, fake, 1)
}

func TestBranchExists_returns_execution_errors(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd(
			"git", "-C", "/repo", "show-ref", "--verify", "--quiet", "refs/heads/feature/login",
		)
		cmd.RunScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return nil, nil, errors.New("git unavailable") },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	_, err := client.BranchExists(context.Background(), "/repo", "feature/login")
	if err == nil || !strings.Contains(err.Error(), "git unavailable") {
		t.Fatalf("BranchExists error = %v, want execution error", err)
	}
	assertCommandCalls(t, fake, 1)
}

func TestDefaultBranch_reads_origin_head(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "symbolic-ref", "refs/remotes/origin/HEAD")
		cmd.OutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return []byte("refs/remotes/origin/main\n"), nil, nil },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	branch, err := client.DefaultBranch(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("DefaultBranch returned error: %v", err)
	}
	if branch != "origin/main" {
		t.Fatalf("expected origin/main, got %q", branch)
	}
	assertCommandCalls(t, fake, 1)
}

func TestDefaultBranch_falls_back_to_remote_set_head_auto(t *testing.T) {
	fake := fakeGit(t,
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "symbolic-ref", "refs/remotes/origin/HEAD")
			cmd.OutputScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return nil, nil, errors.New("missing origin HEAD") },
			}
			return cmd
		},
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "remote", "set-head", "origin", "--auto")
			cmd.RunScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return nil, nil, nil },
			}
			return cmd
		},
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "symbolic-ref", "refs/remotes/origin/HEAD")
			cmd.OutputScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return []byte("refs/remotes/origin/trunk\n"), nil, nil },
			}
			return cmd
		},
	)
	client := git.NewClient().SetExecutor(fake)

	branch, err := client.DefaultBranch(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("DefaultBranch returned error: %v", err)
	}
	if branch != "origin/trunk" {
		t.Fatalf("expected origin/trunk, got %q", branch)
	}
	assertCommandCalls(t, fake, 3)
}

type fakeExitError int

func (e fakeExitError) Error() string {
	return fmt.Sprintf("exit %d", e)
}

func (e fakeExitError) ExitCode() int {
	return int(e)
}

func TestDefaultBranch_errors_when_origin_head_remains_unknown(t *testing.T) {
	fake := fakeGit(t,
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "symbolic-ref", "refs/remotes/origin/HEAD")
			cmd.OutputScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return nil, nil, errors.New("missing origin HEAD") },
			}
			return cmd
		},
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "remote", "set-head", "origin", "--auto")
			cmd.RunScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return nil, nil, nil },
			}
			return cmd
		},
		func(_ string, _ ...string) exec.Cmd {
			cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "symbolic-ref", "refs/remotes/origin/HEAD")
			cmd.OutputScripts = []testingexec.FakeAction{
				func() ([]byte, []byte, error) { return nil, nil, errors.New("still missing") },
			}
			return cmd
		},
	)
	client := git.NewClient().SetExecutor(fake)

	_, err := client.DefaultBranch(context.Background(), "/repo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if strings.Contains(err.Error(), "main") || strings.Contains(err.Error(), "master") {
		t.Fatalf("error should not guess main/master: %v", err)
	}
	assertCommandCalls(t, fake, 3)
}
