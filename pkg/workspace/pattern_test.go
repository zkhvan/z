package workspace_test

import (
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/workspace"
)

func TestResolveBranch(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		instance string
		repo     string
		want     string
	}{
		{"instance only", "{instance}", "login", "acme/api", "login"},
		{"prefixed instance", "feat/{instance}", "login", "acme/api", "feat/login"},
		{"repo is the base name, not the remote id", "feat/{instance}-{repo}", "login", "acme/api", "feat/login-api"},
		{"repeated placeholder", "{instance}/{instance}", "login", "acme/api", "login/login"},
		{"no placeholder is a constant branch", "main-work", "login", "acme/api", "main-work"},
		{"repo only", "wip/{repo}", "login", "acme/ui", "wip/ui"},
		{"unbraced text is literal", "feat/instance", "login", "acme/api", "feat/instance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := workspace.ResolveBranch(tt.pattern, tt.instance, tt.repo)
			if err != nil {
				t.Fatalf("ResolveBranch(%q, %q, %q): %v", tt.pattern, tt.instance, tt.repo, err)
			}
			if got != tt.want {
				t.Fatalf("ResolveBranch(%q, %q, %q) = %q, want %q",
					tt.pattern, tt.instance, tt.repo, got, tt.want)
			}
		})
	}
}

func TestResolveBranch_unknown_placeholder_is_rejected(t *testing.T) {
	_, err := workspace.ResolveBranch("feat/{user}-{instance}", "login", "acme/api")
	if err == nil {
		t.Fatal("expected an error for an unknown placeholder")
	}
	for _, want := range []string{"{user}", "{instance}, {repo}"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %q", err, want)
		}
	}
}

// An invalid result is reported, never slugified: a silent rename would leave
// the user with a branch they did not write.
func TestResolveBranch_invalid_result_is_reported_not_sanitized(t *testing.T) {
	got, err := workspace.ResolveBranch("feat/{instance}", "my feature", "acme/api")
	if err == nil {
		t.Fatalf("expected an error, got branch %q", got)
	}
	for _, want := range []string{"acme/api", "feat/{instance}", "feat/my feature"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %q", err, want)
		}
	}
}

func TestValidateBranchPattern_empty(t *testing.T) {
	if err := workspace.ValidateBranchPattern(""); err == nil {
		t.Fatal("expected an error for an empty pattern")
	}
}
