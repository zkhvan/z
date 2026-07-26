// Package wstest provides the shared world for workspace command tests: a
// temporary config, workspaces root, and projects root, plus seeding and
// assertion helpers. Command packages compose it with a thin local harness that
// binds their own constructor.
package wstest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/workspace"
)

type Harness struct {
	t            *testing.T
	cfg          cmdutil.Config
	root         string
	projectsRoot string
	out          bytes.Buffer
}

// New writes a config pointing at fresh temporary workspaces and projects
// roots. Both are always set, so no test can accidentally reach the real ones.
func New(t *testing.T) *Harness {
	t.Helper()

	cfgDir := t.TempDir()
	h := &Harness{
		t:            t,
		root:         filepath.Join(t.TempDir(), "workspaces"),
		projectsRoot: filepath.Join(t.TempDir(), "projects"),
	}

	contents := fmt.Sprintf("workspaces:\n  root: %s\nprojects:\n  root: %s\n", h.root, h.projectsRoot)
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(h.root, 0o700); err != nil {
		t.Fatalf("create workspaces root: %v", err)
	}

	cfg, err := config.NewWithDir(cfgDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	h.cfg = cfg

	return h
}

// Run executes a command built from the real factory, with output captured.
func (h *Harness) Run(newCmd func(*cmdutil.Factory) *cobra.Command, args ...string) error {
	h.t.Helper()

	f := &cmdutil.Factory{
		IOStreams: &iolib.IOStreams{In: strings.NewReader(""), Out: &h.out, ErrOut: io.Discard},
		Config:    h.cfg,
	}

	cmd := newCmd(f)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	cmd.SetOut(&h.out)
	cmd.SetErr(io.Discard)

	return cmd.Execute()
}

func (h *Harness) Root() string {
	return h.root
}

// SeedInstance writes a manifest without running the command under test.
func (h *Harness) SeedInstance(name string, members ...workspace.Member) {
	h.t.Helper()

	var b strings.Builder
	fmt.Fprint(&b, "version: 1\nmembers:\n")
	for _, m := range members {
		fmt.Fprintf(&b, "  - repo: %s\n    branch: %s\n", m.Repo, m.Branch)
		if m.BaseRef != "" {
			fmt.Fprintf(&b, "    base_ref: %s\n", m.BaseRef)
		}
	}
	h.SeedManifest(name, b.String())
}

// SeedManifest writes raw manifest content, for empty or malformed manifests.
func (h *Harness) SeedManifest(name, raw string) {
	h.t.Helper()
	h.SeedFile(filepath.Join(name, ".z", "instance.yaml"), raw)
}

func (h *Harness) SeedFile(relPath, content string) {
	h.t.Helper()
	path := filepath.Join(h.root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		h.t.Fatalf("seed parent of %q: %v", relPath, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		h.t.Fatalf("seed file %q: %v", relPath, err)
	}
}

func (h *Harness) SeedDir(relPath string) {
	h.t.Helper()
	if err := os.MkdirAll(filepath.Join(h.root, relPath), 0o700); err != nil {
		h.t.Fatalf("seed dir %q: %v", relPath, err)
	}
}

func (h *Harness) FileExists(relPath string) {
	h.t.Helper()
	if _, err := os.Stat(filepath.Join(h.root, relPath)); err != nil {
		h.t.Fatalf("expected %s to exist: %v", relPath, err)
	}
}

func (h *Harness) PathMissing(relPath string) {
	h.t.Helper()
	if _, err := os.Stat(filepath.Join(h.root, relPath)); !os.IsNotExist(err) {
		h.t.Fatalf("expected %s to be missing, stat error: %v", relPath, err)
	}
}

// NoWorkspace asserts that no instance manifest exists for name.
func (h *Harness) NoWorkspace(name string) {
	h.t.Helper()
	path := filepath.Join(h.root, name, ".z", "instance.yaml")
	if _, err := os.Stat(path); err == nil {
		h.t.Fatalf("workspace %q unexpectedly exists", name)
	}
}

func (h *Harness) OutputContains(want string) {
	h.t.Helper()
	if !strings.Contains(h.out.String(), want) {
		h.t.Fatalf("output does not contain %q\ngot: %q", want, h.out.String())
	}
}

func AssertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}
