// Package wstest provides the shared world for workspace command tests: a
// temporary config, workspaces root, projects root, and definitions root, plus seeding and
// assertion helpers. Command packages compose it with a thin local harness that
// binds their own constructor.
package wstest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/workspace"
)

type Harness struct {
	t               *testing.T
	cfg             cmdutil.Config
	root            string
	projectsRoot    string
	definitionsRoot string
	out             bytes.Buffer
	errOut          bytes.Buffer
	tty             bool
}

// WithTTY makes the command believe it is writing to a terminal. Terminal
// capability is one of the few inputs argv cannot express.
func (h *Harness) WithTTY() *Harness {
	h.tty = true
	return h
}

// New writes a config pointing at fresh temporary workspaces, projects, and
// definitions roots. All three are always set, so no test can accidentally
// reach the real ones — definition init writes, so an unset root would create
// directories in the developer's own config dir.
func New(t *testing.T) *Harness {
	t.Helper()

	clearNoColor(t)

	cfgDir := t.TempDir()
	h := &Harness{
		t:               t,
		root:            filepath.Join(t.TempDir(), "workspaces"),
		projectsRoot:    filepath.Join(t.TempDir(), "projects"),
		definitionsRoot: filepath.Join(t.TempDir(), "definitions"),
	}

	contents := fmt.Sprintf(
		"workspaces:\n  root: %s\n  definitions_root: %s\nprojects:\n  root: %s\n",
		h.root, h.definitionsRoot, h.projectsRoot)
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

	streams := &iolib.IOStreams{In: strings.NewReader(""), Out: &h.out, ErrOut: &h.errOut}
	streams.SetTerminal(h.tty)
	if h.tty {
		streams.SetColorProfile(colorprofile.TrueColor)
	}

	f := &cmdutil.Factory{
		IOStreams: streams,
		Config:    h.cfg,
	}

	cmd := newCmd(f)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	cmd.SetOut(&h.out)
	cmd.SetErr(&h.errOut)

	return cmd.Execute()
}

// clearNoColor keeps a developer's own NO_COLOR out of the world, so a test
// asserting colored output cannot pass for the wrong reason. Tests that want
// NO_COLOR set it themselves, after New.
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

func (h *Harness) Root() string {
	return h.root
}

func (h *Harness) DefinitionsRoot() string {
	return h.definitionsRoot
}

// DefinitionDir returns a definition's on-disk path, which hooks receive as
// Z_DEFINITION_PATH.
func (h *Harness) DefinitionDir(name string) string {
	return filepath.Join(h.definitionsRoot, name)
}

// InDir runs the rest of the test from a directory under the workspaces root,
// with no arguments meaning the root itself. The current directory is one of
// the few inputs argv cannot express.
func (h *Harness) InDir(relPath ...string) *Harness {
	h.t.Helper()
	h.t.Chdir(filepath.Join(append([]string{h.root}, relPath...)...))
	return h
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

// SeedDefinition writes a definition manifest without running the command under
// test. An empty pattern omits the field, exercising the default.
func (h *Harness) SeedDefinition(name, pattern string, members ...workspace.Member) {
	h.t.Helper()

	var b strings.Builder
	fmt.Fprint(&b, "version: 1\n")
	if pattern != "" {
		fmt.Fprintf(&b, "branch_pattern: %q\n", pattern)
	}
	fmt.Fprint(&b, "members:\n")
	for _, m := range members {
		fmt.Fprintf(&b, "  - repo: %s\n", m.Repo)
		if m.BaseRef != "" {
			fmt.Fprintf(&b, "    base_ref: %s\n", m.BaseRef)
		}
	}
	h.SeedDefinitionManifest(name, b.String())
}

// SeedDefinitionManifest writes raw definition content, for malformed manifests.
func (h *Harness) SeedDefinitionManifest(name, raw string) {
	h.t.Helper()
	h.seedUnder(h.definitionsRoot, filepath.Join(name, ".z", "definition.yaml"), raw)
}

func (h *Harness) SeedFile(relPath, content string) {
	h.t.Helper()
	h.seedUnder(h.root, relPath, content)
}

// SeedDefinitionHook writes an executable hook script into a definition. The
// executable bit matters: a non-executable hook is a hard error, not a no-op.
func (h *Harness) SeedDefinitionHook(definition string, phase workspace.HookPhase, script string) {
	h.t.Helper()
	h.SeedDefinitionExecFile(definition, filepath.Join("hooks", string(phase)), script)
}

// SeedInstanceFromDefinition writes both a definition and an instance manifest
// that names it, without running any command, so hook behavior can be arranged
// directly on an instance that remembers its source definition.
func (h *Harness) SeedInstanceFromDefinition(instance, definition string, members ...workspace.Member) {
	h.t.Helper()
	h.SeedDefinition(definition, "{instance}", members...)

	var b strings.Builder
	fmt.Fprintf(&b, "version: 1\ndefinition: %s\nmembers:\n", definition)
	for _, m := range members {
		fmt.Fprintf(&b, "  - repo: %s\n    branch: %s\n", m.Repo, m.Branch)
		if m.BaseRef != "" {
			fmt.Fprintf(&b, "    base_ref: %s\n", m.BaseRef)
		}
	}
	h.SeedManifest(instance, b.String())
}

func (h *Harness) seedUnder(base, relPath, content string) {
	h.t.Helper()
	path := filepath.Join(base, relPath)
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

func (h *Harness) DefinitionFileExists(relPath string) {
	h.t.Helper()
	if _, err := os.Stat(filepath.Join(h.definitionsRoot, relPath)); err != nil {
		h.t.Fatalf("expected %s to exist under the definitions root: %v", relPath, err)
	}
}

func (h *Harness) NoDefinition(name string) {
	h.t.Helper()
	path := filepath.Join(h.definitionsRoot, name, ".z", "definition.yaml")
	if _, err := os.Stat(path); err == nil {
		h.t.Fatalf("definition %q unexpectedly exists", name)
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

func (h *Harness) Output() string {
	return h.out.String()
}

func (h *Harness) OutputContains(want string) {
	h.t.Helper()
	if !strings.Contains(h.out.String(), want) {
		h.t.Fatalf("output does not contain %q\ngot: %q", want, h.out.String())
	}
}

// ErrOutput is kept separate from Output so a test can assert which stream a
// message went to: diagnostics belong on stderr, results on stdout.
func (h *Harness) ErrOutput() string {
	return h.errOut.String()
}

func (h *Harness) ErrOutputContains(want string) {
	h.t.Helper()
	if !strings.Contains(h.errOut.String(), want) {
		h.t.Fatalf("stderr does not contain %q\ngot: %q", want, h.errOut.String())
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
