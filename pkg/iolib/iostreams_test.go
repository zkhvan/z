package iolib_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"github.com/zkhvan/z/pkg/iolib"
)

func newStreams() (*iolib.IOStreams, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return &iolib.IOStreams{In: strings.NewReader(""), Out: out, ErrOut: &bytes.Buffer{}}, out
}

func TestIsTerminal_buffer_is_not_a_terminal(t *testing.T) {
	s, _ := newStreams()
	if s.IsTerminal() {
		t.Fatal("a buffer must not be reported as a terminal")
	}
}

// clearNoColor keeps a NO_COLOR in the developer's own environment from
// silently passing tests that assert color is on.
func clearNoColor(t *testing.T) {
	t.Helper()
	previous, ok := os.LookupEnv("NO_COLOR")
	if !ok {
		return
	}
	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatalf("unset NO_COLOR: %v", err)
	}
	t.Cleanup(func() { _ = os.Setenv("NO_COLOR", previous) })
}

func TestColorEnabled_requires_a_capable_terminal(t *testing.T) {
	clearNoColor(t)
	s, _ := newStreams()
	if s.ColorEnabled() {
		t.Fatal("color must be off when not a terminal")
	}

	s.SetColorProfile(colorprofile.TrueColor)
	if !s.ColorEnabled() {
		t.Fatal("color must be on for a capable terminal")
	}
}

// no-color.org requires the variable to be present and non-empty.
func TestColorEnabled_ignores_an_empty_no_color(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	s, _ := newStreams()
	s.SetTerminal(true)
	s.SetColorProfile(colorprofile.TrueColor)

	if !s.ColorEnabled() {
		t.Fatal("an empty NO_COLOR must not disable color")
	}
}

func TestColorEnabled_respects_no_color_without_losing_terminal(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	s, _ := newStreams()
	s.SetTerminal(true)
	s.SetColorProfile(colorprofile.TrueColor)

	if s.ColorEnabled() {
		t.Fatal("NO_COLOR must disable color")
	}
	if !s.IsTerminal() {
		t.Fatal("NO_COLOR must not change terminal capability")
	}
}

func TestColorEnabled_respects_the_opt_out(t *testing.T) {
	s, _ := newStreams()
	s.SetTerminal(true)
	s.SetColorProfile(colorprofile.TrueColor)
	s.SetColorDisabled(true)

	if s.ColorEnabled() {
		t.Fatal("--no-color must disable color")
	}
	if !s.IsTerminal() {
		t.Fatal("--no-color must not change terminal capability")
	}
}

// Styles always render escapes; StyledOut is what removes them, so a caller
// never has to branch on capability.
func TestStyledOut_strips_styling_for_a_plain_destination(t *testing.T) {
	s, out := newStreams()
	st := iolib.NewStyles()

	fmt.Fprint(s.StyledOut(), st.Bold.Render("auth"))

	if got := out.String(); got != "auth" {
		t.Fatalf("StyledOut = %q, want plain text", got)
	}
}

func TestStyledOut_keeps_styling_for_a_capable_terminal(t *testing.T) {
	clearNoColor(t)
	s, out := newStreams()
	s.SetColorProfile(colorprofile.TrueColor)
	st := iolib.NewStyles()

	fmt.Fprint(s.StyledOut(), st.Bold.Render("auth"))

	if got := out.String(); !strings.Contains(got, "\x1b[1m") {
		t.Fatalf("StyledOut = %q, want bold escapes", got)
	}
}

func TestWidth_counts_terminal_cells_not_bytes(t *testing.T) {
	if got := iolib.Width("日本語"); got != 6 {
		t.Fatalf("Width(CJK) = %d, want 6", got)
	}
	if got := iolib.Width("abc"); got != 3 {
		t.Fatalf("Width(ascii) = %d, want 3", got)
	}
}
