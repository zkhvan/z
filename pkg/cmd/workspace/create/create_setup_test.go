package create_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/zkhvan/z/pkg/cmd/workspace/create"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/iolib"
)

// harness owns the test environment and provides setup + assertions.
type harness struct {
	t      *testing.T
	cfgDir string
	wsRoot string
	cfg    cmdutil.Config
	out    bytes.Buffer
}

type cmdRunner struct{ h *harness }

// newCommandTest returns a command runner, the harness, and a cleanup func.
func newCommandTest(t *testing.T) (*cmdRunner, *harness, func()) {
	t.Helper()
	h := &harness{t: t, cfgDir: t.TempDir()}

	cfg, err := config.NewWithDir(h.cfgDir)
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	h.cfg = cfg

	return &cmdRunner{h: h}, h, func() {} // t.TempDir auto-cleans
}

// --- config builder ---

type configSpec struct {
	workspaceRoot  string
	createTempRoot bool
}

type configOption func(*configSpec)

// withWorkspaceRoot creates and uses a fresh temp workspaces root.
func withWorkspaceRoot() configOption {
	return func(s *configSpec) { s.createTempRoot = true }
}

func (h *harness) config(opts ...configOption) {
	h.t.Helper()

	var spec configSpec
	for _, o := range opts {
		o(&spec)
	}
	if spec.createTempRoot {
		spec.workspaceRoot = h.t.TempDir()
	}
	h.wsRoot = spec.workspaceRoot

	var b strings.Builder
	fmt.Fprint(&b, "workspaces:\n")
	if spec.workspaceRoot != "" {
		fmt.Fprintf(&b, "  root: %s\n", spec.workspaceRoot)
	}

	path := filepath.Join(h.cfgDir, "config.yaml")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		h.t.Fatalf("write config: %v", err)
	}

	cfg, err := config.NewWithDir(h.cfgDir)
	if err != nil {
		h.t.Fatalf("load config: %v", err)
	}
	h.cfg = cfg
}

// --- cmd namespace: run options ---

type runSpec struct {
	name    string
	nameSet bool
	members []string
}

type runOption func(*runSpec)

func (*cmdRunner) withName(name string) runOption {
	return func(s *runSpec) { s.name = name; s.nameSet = true }
}

func (*cmdRunner) withMember(member string) runOption {
	return func(s *runSpec) { s.members = append(s.members, member) }
}

func (c *cmdRunner) run(opts ...runOption) error {
	c.h.t.Helper()

	var spec runSpec
	for _, o := range opts {
		o(&spec)
	}

	f := &cmdutil.Factory{
		IOStreams: &iolib.IOStreams{In: strings.NewReader(""), Out: &c.h.out, ErrOut: io.Discard},
		Config:    c.h.cfg,
	}

	cmd := create.NewCmdCreate(f)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	var args []string
	if spec.nameSet {
		args = append(args, spec.name)
	}
	for _, m := range spec.members {
		args = append(args, "--member", m)
	}
	cmd.SetArgs(args)
	cmd.SetOut(&c.h.out)
	cmd.SetErr(io.Discard)

	return cmd.Execute()
}

// --- assertions ---

func (c *cmdRunner) outputContains(want string) {
	c.h.t.Helper()
	if !strings.Contains(c.h.out.String(), want) {
		c.h.t.Fatalf("output does not contain %q\ngot: %q", want, c.h.out.String())
	}
}

// --- wsDir namespace: seed options ---

type wsDirSpec struct {
	name             string
	instanceManifest bool
}

type wsDirOption func(*wsDirSpec)

// wsDir is a stateless namespace for seedWorkspaceDir options.
type wsDirNS struct{}

var wsDir wsDirNS

func (wsDirNS) withName(name string) wsDirOption {
	return func(s *wsDirSpec) { s.name = name }
}

func (wsDirNS) withEmptyInstanceManifest() wsDirOption {
	return func(s *wsDirSpec) { s.instanceManifest = true }
}

// seedWorkspaceDir creates an instance directory, optionally with an empty
// .z/instance.yaml, without going through the command.
func (h *harness) seedWorkspaceDir(opts ...wsDirOption) {
	h.t.Helper()

	var spec wsDirSpec
	for _, o := range opts {
		o(&spec)
	}

	dir := filepath.Join(h.wsRoot, spec.name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		h.t.Fatalf("seed dir %q: %v", spec.name, err)
	}

	if spec.instanceManifest {
		dotZ := filepath.Join(dir, ".z")
		if err := os.MkdirAll(dotZ, 0o700); err != nil {
			h.t.Fatalf("seed .z %q: %v", spec.name, err)
		}
		if err := os.WriteFile(filepath.Join(dotZ, "instance.yaml"), nil, 0o600); err != nil {
			h.t.Fatalf("seed manifest %q: %v", spec.name, err)
		}
	}
}

// noWorkspace asserts that no instance directory exists for name.
func (h *harness) noWorkspace(name string) {
	h.t.Helper()
	path := filepath.Join(h.wsRoot, name, ".z", "instance.yaml")
	if _, err := os.Stat(path); err == nil {
		h.t.Fatalf("workspace %q unexpectedly exists", name)
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}

type instanceManifest struct {
	Version int `yaml:"version"`
	Members []struct {
		Repo    string `yaml:"repo"`
		Branch  string `yaml:"branch"`
		BaseRef string `yaml:"base_ref"`
	} `yaml:"members"`
}

type workspaceAssert struct {
	t        *testing.T
	name     string
	manifest instanceManifest
}

// workspace reads and parses the manifest for name, ready for assertions.
func (h *harness) workspace(name string) *workspaceAssert {
	h.t.Helper()

	path := filepath.Join(h.wsRoot, name, ".z", "instance.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		h.t.Fatalf("workspace %q: read manifest: %v", name, err)
	}

	var m instanceManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		h.t.Fatalf("workspace %q: parse manifest: %v", name, err)
	}

	return &workspaceAssert{t: h.t, name: name, manifest: m}
}

func (w *workspaceAssert) hasVersion(want int) *workspaceAssert {
	w.t.Helper()
	if w.manifest.Version != want {
		w.t.Fatalf("workspace %q: version = %d, want %d", w.name, w.manifest.Version, want)
	}
	return w
}

func (w *workspaceAssert) memberCount(want int) *workspaceAssert {
	w.t.Helper()
	if len(w.manifest.Members) != want {
		w.t.Fatalf("workspace %q: member count = %d, want %d", w.name, len(w.manifest.Members), want)
	}
	return w
}

func (w *workspaceAssert) hasMember(repo, branch, baseRef string) *workspaceAssert {
	w.t.Helper()
	for _, m := range w.manifest.Members {
		if m.Repo == repo && m.Branch == branch && m.BaseRef == baseRef {
			return w
		}
	}
	w.t.Fatalf("workspace %q: member {repo:%q branch:%q base_ref:%q} not found in %+v",
		w.name, repo, branch, baseRef, w.manifest.Members)
	return w
}
