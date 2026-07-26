package cmd_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/cmd"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/iolib"
)

// The --no-color opt-out is a root-level flag, so its effect on the shared
// IOStreams is only observable through the root command.
func TestRoot_no_color_disables_color_for_every_command(t *testing.T) {
	streams := &iolib.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, ErrOut: io.Discard}
	streams.SetTerminal(true)
	streams.SetColorProfile(colorprofile.TrueColor)

	root, err := cmd.NewCmdRoot(&cmdutil.Factory{IOStreams: streams}, "1.0.0", "2026-07-26")
	assert.NoError(t, err)

	root.SetArgs([]string{"version", "--no-color"})
	assert.NoError(t, root.Execute())

	if streams.ColorEnabled() {
		t.Fatal("--no-color did not disable color")
	}
	if !streams.IsTerminal() {
		t.Fatal("--no-color must not change terminal capability")
	}
}

func TestRoot_color_stays_enabled_without_the_flag(t *testing.T) {
	if previous, ok := os.LookupEnv("NO_COLOR"); ok {
		if err := os.Unsetenv("NO_COLOR"); err != nil {
			t.Fatalf("unset NO_COLOR: %v", err)
		}
		t.Cleanup(func() { _ = os.Setenv("NO_COLOR", previous) })
	}

	streams := &iolib.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, ErrOut: io.Discard}
	streams.SetTerminal(true)
	streams.SetColorProfile(colorprofile.TrueColor)

	root, err := cmd.NewCmdRoot(&cmdutil.Factory{IOStreams: streams}, "1.0.0", "2026-07-26")
	assert.NoError(t, err)

	root.SetArgs([]string{"version"})
	assert.NoError(t, root.Execute())

	if !streams.ColorEnabled() {
		t.Fatal("color should stay enabled on a terminal")
	}
}
