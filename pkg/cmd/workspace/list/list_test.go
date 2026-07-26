package list_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestList_plain_output_is_tab_separated(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("auth",
		workspace.Member{Repo: "acme/api", Branch: "feat-1"},
		workspace.Member{Repo: "acme/web", Branch: "feat-1"},
	)

	assert.NoError(t, h.run())

	want := fmt.Sprintf("auth\t%s/auth\tnew\n  acme/api\tfeat-1\t—\n  acme/web\tfeat-1\t—\n", h.Root())
	if h.Output() != want {
		t.Fatalf("plain output changed\n got: %q\nwant: %q", h.Output(), want)
	}
}

func TestList_plain_output_has_no_escape_sequences(t *testing.T) {
	h := newCommandTest(t)
	h.SeedInstance("auth", workspace.Member{Repo: "acme/api", Branch: "feat-1"})

	assert.NoError(t, h.run())

	if strings.ContainsRune(h.Output(), '\x1b') {
		t.Fatalf("plain output contains escapes: %q", h.Output())
	}
}

func TestList_terminal_output_is_a_colored_tree(t *testing.T) {
	h := newCommandTest(t)
	h.WithTTY()
	h.SeedInstance("auth",
		workspace.Member{Repo: "acme/api", Branch: "feat-1"},
		workspace.Member{Repo: "acme/web", Branch: "feat-1"},
	)

	assert.NoError(t, h.run())

	out := h.Output()
	for _, want := range []string{"\x1b[1mauth", "├─", "└─", "· new"} {
		if !strings.Contains(out, want) {
			t.Fatalf("terminal output missing %q:\n%q", want, out)
		}
	}
	if strings.ContainsRune(out, '\t') {
		t.Fatalf("terminal output should be padded, not tabbed: %q", out)
	}
}

// Members align into one grid across instances, not a ragged block per
// workspace.
func TestList_terminal_output_aligns_members_across_instances(t *testing.T) {
	h := newCommandTest(t)
	h.WithTTY()
	h.SeedInstance("auth",
		workspace.Member{Repo: "acme/api", Branch: "feat-1"},
		workspace.Member{Repo: "acme/web-frontend", Branch: "feat-1"},
	)
	h.SeedInstance("spike", workspace.Member{Repo: "acme/api", Branch: "x"})

	assert.NoError(t, h.run())

	lines := strings.Split(strings.TrimRight(h.Output(), "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %q", h.Output())
	}

	columns := make([]int, 0, 3)
	for _, i := range []int{1, 2, 4} {
		columns = append(columns, strings.Index(lines[i], "\x1b[36m"))
	}
	if columns[0] != columns[1] || columns[1] != columns[2] {
		t.Fatalf("branch column is ragged at %v:\n%s", columns, h.Output())
	}
}

func TestList_terminal_output_omits_a_symbol_for_unknown_state(t *testing.T) {
	h := newCommandTest(t)
	h.WithTTY()
	h.SeedInstance("auth", workspace.Member{Repo: "acme/api", Branch: "feat-1"})

	assert.NoError(t, h.run())

	if strings.Contains(h.Output(), "· —") {
		t.Fatalf("unknown state rendered two placeholders: %q", h.Output())
	}
}

// NO_COLOR strips color without downgrading the terminal layout, which is what
// the convention actually asks for.
func TestList_no_color_keeps_the_tree_without_escapes(t *testing.T) {
	h := newCommandTest(t)
	t.Setenv("NO_COLOR", "1")
	h.WithTTY()
	h.SeedInstance("auth", workspace.Member{Repo: "acme/api", Branch: "feat-1"})

	assert.NoError(t, h.run())

	out := h.Output()
	if strings.ContainsRune(out, '\x1b') {
		t.Fatalf("NO_COLOR output contains escapes: %q", out)
	}
	if !strings.Contains(out, "└─") {
		t.Fatalf("NO_COLOR output lost the tree: %q", out)
	}
}

func TestList_empty_root_prints_nothing(t *testing.T) {
	h := newCommandTest(t)

	assert.NoError(t, h.run())

	if h.Output() != "" {
		t.Fatalf("expected no output, got %q", h.Output())
	}
}
