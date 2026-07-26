package git_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/exec"
	testingexec "github.com/zkhvan/z/pkg/exec/testing"
	"github.com/zkhvan/z/pkg/git"
)

func TestWorktreeAdd_creates_branch_from_base_ref(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd(
			"git", "-C", "/repo", "worktree", "add", "-b", "feature/login", "--", "/workspace/repo", "origin/main",
		)
		cmd.CombinedOutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return nil, nil, nil },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	err := client.WorktreeAdd(context.Background(), git.WorktreeAddOptions{
		RepoPath:     "/repo",
		WorktreePath: "/workspace/repo",
		Branch:       "feature/login",
		BaseRef:      "origin/main",
		CreateBranch: true,
	})
	if err != nil {
		t.Fatalf("WorktreeAdd returned error: %v", err)
	}
	assertCommandCalls(t, fake, 1)
}

func TestWorktreeAdd_uses_existing_branch(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "worktree", "add", "--", "/workspace/repo", "feature/login")
		cmd.CombinedOutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return nil, nil, nil },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	err := client.WorktreeAdd(context.Background(), git.WorktreeAddOptions{
		RepoPath:     "/repo",
		WorktreePath: "/workspace/repo",
		Branch:       "feature/login",
	})
	if err != nil {
		t.Fatalf("WorktreeAdd returned error: %v", err)
	}
	assertCommandCalls(t, fake, 1)
}

func TestWorktreeList_parses_porcelain_output(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "worktree", "list", "--porcelain")
		cmd.OutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) {
				output := "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
					"worktree /workspace/repo\nHEAD def\nbranch refs/heads/feature/login\n"
				return []byte(output), nil, nil
			},
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	worktrees, err := client.WorktreeList(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("WorktreeList returned error: %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %+v", worktrees)
	}
	if worktrees[0].Path != "/repo" || worktrees[0].Branch != "main" {
		t.Fatalf("unexpected first worktree: %+v", worktrees[0])
	}
	if worktrees[1].Path != "/workspace/repo" || worktrees[1].Branch != "feature/login" {
		t.Fatalf("unexpected second worktree: %+v", worktrees[1])
	}
	assertCommandCalls(t, fake, 1)
}

func TestWorktreeRemove_deregisters_worktree(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "worktree", "remove", "--", "/workspace/repo")
		cmd.CombinedOutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return nil, nil, nil },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	err := client.WorktreeRemove(context.Background(), git.WorktreeRemoveOptions{
		RepoPath:     "/repo",
		WorktreePath: "/workspace/repo",
	})
	if err != nil {
		t.Fatalf("WorktreeRemove returned error: %v", err)
	}
	assertCommandCalls(t, fake, 1)
}

func TestWorktreeRemove_force_precedes_path_separator(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd(
			"git", "-C", "/repo", "worktree", "remove", "--force", "--", "/workspace/repo",
		)
		cmd.CombinedOutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) { return nil, nil, nil },
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	err := client.WorktreeRemove(context.Background(), git.WorktreeRemoveOptions{
		RepoPath:     "/repo",
		WorktreePath: "/workspace/repo",
		Force:        true,
	})
	if err != nil {
		t.Fatalf("WorktreeRemove returned error: %v", err)
	}
	assertCommandCalls(t, fake, 1)
}

func TestWorktreeRemove_reports_git_output_on_failure(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "worktree", "remove", "--", "/workspace/repo")
		cmd.CombinedOutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) {
				return []byte("fatal: contains modified or untracked files"), nil, errors.New("exit status 128")
			},
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	err := client.WorktreeRemove(context.Background(), git.WorktreeRemoveOptions{
		RepoPath:     "/repo",
		WorktreePath: "/workspace/repo",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "modified or untracked files") {
		t.Fatalf("error %q does not carry git output", err.Error())
	}
}

func TestWorktreeList_marks_prunable_registrations(t *testing.T) {
	fake := fakeGit(t, func(_ string, _ ...string) exec.Cmd {
		cmd := testingexec.NewFakeCmd("git", "-C", "/repo", "worktree", "list", "--porcelain")
		cmd.OutputScripts = []testingexec.FakeAction{
			func() ([]byte, []byte, error) {
				output := "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
					"worktree /workspace/repo\nHEAD def\nbranch refs/heads/feature/login\n" +
					"prunable gitdir file points to non-existent location\n"
				return []byte(output), nil, nil
			},
		}
		return cmd
	})
	client := git.NewClient().SetExecutor(fake)

	worktrees, err := client.WorktreeList(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("WorktreeList returned error: %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %+v", worktrees)
	}
	if worktrees[0].Prunable {
		t.Fatalf("canonical worktree marked prunable: %+v", worktrees[0])
	}
	if !worktrees[1].Prunable {
		t.Fatalf("deleted worktree not marked prunable: %+v", worktrees[1])
	}
	if worktrees[1].Branch != "feature/login" {
		t.Fatalf("unexpected branch on prunable worktree: %+v", worktrees[1])
	}
}

func fakeGit(t *testing.T, scripts ...testingexec.FakeCommandAction) *testingexec.FakeExec {
	t.Helper()
	return &testingexec.FakeExec{CommandScript: scripts}
}

func assertCommandCalls(t *testing.T, fake *testingexec.FakeExec, want int) {
	t.Helper()
	if fake.CommandCalls != want {
		t.Fatalf("expected %d command(s), got %d", want, fake.CommandCalls)
	}
}
