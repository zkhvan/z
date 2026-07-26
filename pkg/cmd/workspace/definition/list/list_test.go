package list_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/workspace"
)

func TestList_empty_root(t *testing.T) {
	h := newCommandTest(t)

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	if h.Output() != "" {
		t.Fatalf("expected no output, got %q", h.Output())
	}
}

func TestList_definition_with_members(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}",
		workspace.Member{Repo: "acme/api", BaseRef: "develop"},
		workspace.Member{Repo: "acme/ui"},
	)

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	lines := outputLines(t, h)
	assertFields(t, lines[0], "api-feature", "feat/{instance}")
	assertFields(t, lines[1], "acme/api", "develop")
	assertFields(t, lines[2], "acme/ui", "—")
}

func TestList_sorted_alphabetically(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("zebra", "", workspace.Member{Repo: "acme/z"})
	h.SeedDefinition("alpha", "", workspace.Member{Repo: "acme/a"})

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	lines := outputLines(t, h)
	assertFields(t, lines[0], "alpha")
	assertFields(t, lines[2], "zebra")
}

// An omitted pattern is reported as the default that will actually be applied,
// not as an empty column.
func TestList_omitted_pattern_shows_the_default(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "", workspace.Member{Repo: "acme/api"})

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	assertFields(t, outputLines(t, h)[0], "api-feature", "{instance}")
}

func TestList_directory_without_a_manifest_is_skipped(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("real", "", workspace.Member{Repo: "acme/api"})
	h.SeedDefinitionManifest("decoy/nested", "version: 1\n")

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	if strings.Contains(h.Output(), "decoy") {
		t.Fatalf("a directory without a definition manifest was listed:\n%s", h.Output())
	}
	h.OutputContains("real")
}

func TestList_unparseable_manifest_is_reported_not_skipped(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinitionManifest("broken", "version: 1\nmembers: [oh: no: yes\n")

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	h.OutputContains("broken")
	assertFields(t, outputLines(t, h)[0], "broken:")
}

func TestList_unknown_placeholder_is_reported(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("typo", "feat/{user}", workspace.Member{Repo: "acme/api"})

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	h.OutputContains("broken:")
	h.OutputContains("{user}")
}

func TestList_colliding_member_base_names_are_reported(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("collide", "",
		workspace.Member{Repo: "acme/api"},
		workspace.Member{Repo: "other/api"},
	)

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	h.OutputContains("broken:")
	h.OutputContains("worktree directory name")
}

// A yaml error spans two lines; the plain rendering is one line per row.
func TestList_multiline_error_stays_on_one_row(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinitionManifest("broken", "version: 1\nmembers: [oh: no: yes\n")

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	if got := len(outputLines(t, h)); got != 1 {
		t.Fatalf("expected 1 output line, got %d:\n%s", got, h.Output())
	}
}

func TestList_zero_member_definition_is_not_broken(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("scaffold", "feat/{instance}")

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	assertNotBroken(t, outputLines(t, h)[0])
	assertFields(t, outputLines(t, h)[0], "scaffold", "feat/{instance}")
}

func outputLines(t *testing.T, h *harness) []string {
	t.Helper()
	return strings.Split(strings.TrimRight(h.Output(), "\n"), "\n")
}

// assertNotBroken inspects the summary column rather than the whole line: the
// directory column holds a temp path that embeds the test name.
func assertNotBroken(t *testing.T, line string) {
	t.Helper()
	fields := strings.Split(line, "\t")
	if len(fields) < 3 {
		t.Fatalf("expected a definition row with 3 fields, got %q", line)
	}
	if strings.HasPrefix(fields[2], "broken:") {
		t.Fatalf("definition reported as broken: %q", fields[2])
	}
}

func assertFields(t *testing.T, line string, want ...string) {
	t.Helper()
	fields := strings.Split(strings.TrimSpace(line), "\t")
	for _, w := range want {
		found := false
		for _, f := range fields {
			if strings.HasPrefix(f, w) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("line %q has no field starting with %q (fields: %q)", line, w, fields)
		}
	}
}

func TestList_terminal_output_is_a_tree(t *testing.T) {
	h := newCommandTest(t)
	h.WithTTY()
	h.SeedDefinition("api-feature", "feat/{instance}",
		workspace.Member{Repo: "acme/api", BaseRef: "develop"},
		workspace.Member{Repo: "acme/web"},
	)

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	out := h.Output()
	for _, want := range []string{"\x1b[1mapi-feature", "├─", "└─", "feat/{instance}", "develop"} {
		if !strings.Contains(out, want) {
			t.Fatalf("terminal output missing %q:\n%q", want, out)
		}
	}
	if strings.ContainsRune(out, '\t') {
		t.Fatalf("terminal output should be padded, not tabbed: %q", out)
	}
}

// Members align into one grid across definitions, not a ragged block each.
func TestList_terminal_output_aligns_members_across_definitions(t *testing.T) {
	h := newCommandTest(t)
	h.WithTTY()
	h.SeedDefinition("alpha", "", workspace.Member{Repo: "acme/a", BaseRef: "main"})
	h.SeedDefinition("zebra", "", workspace.Member{Repo: "acme/much-longer-name", BaseRef: "main"})

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	short, long := memberColumn(t, h, "acme/a"), memberColumn(t, h, "acme/much-longer-name")
	if short != long {
		t.Fatalf("base ref column starts at %d for the short repo and %d for the long one", short, long)
	}
}

func TestList_terminal_output_marks_a_broken_definition(t *testing.T) {
	h := newCommandTest(t)
	h.WithTTY()
	h.SeedDefinition("typo", "feat/{user}", workspace.Member{Repo: "acme/api"})

	if err := h.run(); err != nil {
		t.Fatalf("list: %v", err)
	}

	for _, want := range []string{"\u2717", "broken:", "{user}"} {
		if !strings.Contains(h.Output(), want) {
			t.Fatalf("terminal output missing %q:\n%q", want, h.Output())
		}
	}
}

// memberColumn is the cell offset at which the base ref begins on the member
// row for repo, measured with escapes stripped.
func memberColumn(t *testing.T, h *harness, repo string) int {
	t.Helper()
	for _, line := range outputLines(t, h) {
		plain := stripANSI(line)
		if idx := strings.Index(plain, repo); idx >= 0 {
			return strings.Index(plain, "main")
		}
	}
	t.Fatalf("no member row for %q in:\n%s", repo, h.Output())
	return 0
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
