package iolib

import "charm.land/lipgloss/v2"

// Styles is the shared vocabulary for terminal output. Styles always render
// full-fidelity ANSI; StyledOut downsamples or strips it to match the
// destination, so nothing here needs to know whether color is available.
type Styles struct {
	Bold   lipgloss.Style
	Dim    lipgloss.Style
	Red    lipgloss.Style
	Green  lipgloss.Style
	Yellow lipgloss.Style
	Cyan   lipgloss.Style
}

func NewStyles() Styles {
	return Styles{
		Bold:   lipgloss.NewStyle().Bold(true),
		Dim:    lipgloss.NewStyle().Faint(true),
		Red:    lipgloss.NewStyle().Foreground(lipgloss.Red),
		Green:  lipgloss.NewStyle().Foreground(lipgloss.Green),
		Yellow: lipgloss.NewStyle().Foreground(lipgloss.Yellow),
		Cyan:   lipgloss.NewStyle().Foreground(lipgloss.Cyan),
	}
}

// Width is the rendered width of s in terminal cells, which is what column
// alignment needs: len() counts bytes, and a CJK or emoji character is neither
// one byte nor one cell.
func Width(s string) int {
	return lipgloss.Width(s)
}
