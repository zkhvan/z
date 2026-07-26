package iolib

import (
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
)

type IOStreams struct {
	In     io.Reader // think os.Stdin
	Out    io.Writer // think os.Stdout
	ErrOut io.Writer // think os.Stderr

	terminal      *bool
	profile       *colorprofile.Profile
	colorDisabled bool
}

func System() *IOStreams {
	return &IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
}

// IsTerminal gates rich rendering — symbols and alignment as well as color — so
// that anything reading z's output through a pipe keeps seeing stable plain
// text.
func (s *IOStreams) IsTerminal() bool {
	if s.terminal != nil {
		return *s.terminal
	}

	f, ok := s.Out.(term.File)
	return ok && term.IsTerminal(f.Fd())
}

// IsInputTerminal reports whether In is a terminal. Prompting needs this as
// well as IsTerminal: `z ... < answers.txt` has a terminal to draw on but
// nobody to answer.
func (s *IOStreams) IsInputTerminal() bool {
	if s.terminal != nil {
		return *s.terminal
	}

	f, ok := s.In.(term.File)
	return ok && term.IsTerminal(f.Fd())
}

// IsInteractive reports whether z may prompt: someone is watching and someone
// can answer.
func (s *IOStreams) IsInteractive() bool {
	return s.IsInputTerminal() && s.IsTerminal()
}

// SetTerminal overrides detection for both streams, for tests and for callers
// that already know.
func (s *IOStreams) SetTerminal(terminal bool) {
	s.terminal = &terminal
}

// SetColorProfile overrides the detected color capability. Detection reads the
// environment, so tests writing to a buffer must state the capability instead.
func (s *IOStreams) SetColorProfile(profile colorprofile.Profile) {
	s.profile = &profile
}

func (s *IOStreams) SetColorDisabled(disabled bool) {
	s.colorDisabled = disabled
}

// ColorProfile resolves capability first, then policy: NO_COLOR and --no-color
// always win over whatever the terminal can do. They strip every escape, not
// just color, because the usual reason for setting them is a reader that
// mangles escapes — layout survives regardless, since it is plain text.
func (s *IOStreams) ColorProfile() colorprofile.Profile {
	if s.colorDisabled || noColorRequested() {
		return colorprofile.NoTTY
	}
	return s.detectProfile()
}

func (s *IOStreams) detectProfile() colorprofile.Profile {
	if s.profile != nil {
		return *s.profile
	}
	if s.terminal != nil && !*s.terminal {
		return colorprofile.NoTTY
	}
	return colorprofile.Detect(s.Out, os.Environ())
}

func (s *IOStreams) ColorEnabled() bool {
	return s.ColorProfile() > colorprofile.ASCII
}

// StyledOut downsamples or strips styling to match the destination, so callers
// render styles unconditionally and never branch on capability themselves.
func (s *IOStreams) StyledOut() io.Writer {
	return &colorprofile.Writer{Forward: s.Out, Profile: s.ColorProfile()}
}

// noColorRequested follows no-color.org: the variable must be present and
// non-empty, so NO_COLOR= does not disable color.
func noColorRequested() bool {
	return os.Getenv("NO_COLOR") != ""
}
