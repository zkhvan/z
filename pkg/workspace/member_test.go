package workspace_test

import (
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestParseMember(t *testing.T) {
	t.Run("parses repo and branch without base", func(t *testing.T) {
		m, err := workspace.ParseMember("owner/repo@feature/login")
		assert.NoError(t, err)
		if m.Repo != "owner/repo" {
			t.Fatalf("expected Repo %q, got %q", "owner/repo", m.Repo)
		}
		if m.Branch != "feature/login" {
			t.Fatalf("expected Branch %q, got %q", "feature/login", m.Branch)
		}
		if m.BaseRef != "" {
			t.Fatalf("expected empty BaseRef, got %q", m.BaseRef)
		}
	})

	t.Run("parses repo, branch, and base_ref", func(t *testing.T) {
		m, err := workspace.ParseMember("owner/repo@feature/login:main")
		assert.NoError(t, err)
		if m.Repo != "owner/repo" {
			t.Fatalf("expected Repo %q, got %q", "owner/repo", m.Repo)
		}
		if m.Branch != "feature/login" {
			t.Fatalf("expected Branch %q, got %q", "feature/login", m.Branch)
		}
		if m.BaseRef != "main" {
			t.Fatalf("expected BaseRef %q, got %q", "main", m.BaseRef)
		}
	})
}

func TestParseMember_Errors(t *testing.T) {
	cases := []string{
		"owner/repo",                     // no @ at all
		"@branch",                        // empty repo
		"noslash@branch",                 // repo missing owner/
		"../../outside@branch",           // repo escapes the projects root
		`owner/repo\evil@branch`,         // repo uses a platform path separator
		" owner/repo@branch",             // repo has leading whitespace
		"owner/re po@branch",             // repo has internal whitespace
		"owner/repo@",                    // empty branch
		"owner/repo@feature..login",      // repeated dots
		"owner/repo@-feature",            // option-like branch
		"owner/repo@feature.lock",        // lock-file suffix
		"owner/repo@feature@{1}",         // reflog syntax
		"owner/repo@feature login",       // whitespace
		"owner/repo@feature/.hidden",     // dot-prefixed path component
		"owner/repo@feature/",            // trailing slash
		"owner/repo@HEAD",                // reserved branch name
		`owner/repo@feature\login`,       // invalid ref character
		"owner/repo@feature:-base",       // option-like base ref
		"owner/repo@feature:main\nother", // control character in base ref
	}

	for _, s := range cases {
		t.Run(s, func(t *testing.T) {
			_, err := workspace.ParseMember(s)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", s)
			}
		})
	}
}

func TestValidateMembers(t *testing.T) {
	t.Run("accepts distinct worktree names", func(t *testing.T) {
		err := workspace.ValidateMembers([]workspace.Member{
			{Repo: "owner/api", Branch: "main"},
			{Repo: "owner/web", Branch: "main"},
		})
		assert.NoError(t, err)
	})

	t.Run("rejects zero members", func(t *testing.T) {
		if err := workspace.ValidateMembers(nil); err == nil {
			t.Fatal("expected error for zero members, got nil")
		}
	})

	t.Run("rejects duplicate worktree names", func(t *testing.T) {
		err := workspace.ValidateMembers([]workspace.Member{
			{Repo: "zkhvan/api", Branch: "main"},
			{Repo: "other/api", Branch: "main"},
		})
		if err == nil {
			t.Fatal("expected error for duplicate base name, got nil")
		}
	})
}
