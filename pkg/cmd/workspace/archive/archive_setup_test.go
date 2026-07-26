package archive_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/archive"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/iolib"
)

type harness struct {
	t      *testing.T
	cfgDir string
	wsRoot string
	cfg    cmdutil.Config
	out    bytes.Buffer
}

type cmdRunner struct{ h *harness }

func newCommandTest(t *testing.T) (*cmdRunner, *harness) {
	t.Helper()
	h := &harness{t: t, cfgDir: t.TempDir()}
	cfg, err := config.NewWithDir(h.cfgDir)
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	h.cfg = cfg
	return &cmdRunner{h: h}, h
}

type configSpec struct {
	workspaceRoot  string
	createTempRoot bool
}

type configOption func(*configSpec)

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
	// A projects root under the temp dir keeps canonical clone lookups away from
	// the real one.
	fmt.Fprintf(&b, "projects:\n  root: %s\n", filepath.Join(h.t.TempDir(), "projects"))

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

func (h *harness) seedInstance(name, repo, branch string) {
	h.t.Helper()
	dotZ := filepath.Join(h.wsRoot, name, ".z")
	if err := os.MkdirAll(dotZ, 0o700); err != nil {
		h.t.Fatalf("seed .z for %q: %v", name, err)
	}
	manifest := fmt.Sprintf("version: 1\nmembers:\n  - repo: %s\n    branch: %s\n", repo, branch)
	if err := os.WriteFile(filepath.Join(dotZ, "instance.yaml"), []byte(manifest), 0o600); err != nil {
		h.t.Fatalf("seed manifest for %q: %v", name, err)
	}
}

func (h *harness) seedMemberFile(name, member, file string) {
	h.t.Helper()
	dir := filepath.Join(h.wsRoot, name, member)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		h.t.Fatalf("seed member dir %q: %v", member, err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte("content\n"), 0o600); err != nil {
		h.t.Fatalf("seed member file %q: %v", file, err)
	}
}

func (h *harness) fileExists(relPath string) {
	h.t.Helper()
	if _, err := os.Stat(filepath.Join(h.wsRoot, relPath)); err != nil {
		h.t.Fatalf("expected %s to exist: %v", relPath, err)
	}
}

func (h *harness) pathMissing(relPath string) {
	h.t.Helper()
	if _, err := os.Stat(filepath.Join(h.wsRoot, relPath)); !os.IsNotExist(err) {
		h.t.Fatalf("expected %s to be missing, stat error: %v", relPath, err)
	}
}

type runSpec struct {
	name    string
	nameSet bool
	force   bool
}

type runOption func(*runSpec)

func (*cmdRunner) withName(name string) runOption {
	return func(s *runSpec) { s.name = name; s.nameSet = true }
}

func (*cmdRunner) withForce() runOption {
	return func(s *runSpec) { s.force = true }
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

	cmd := archive.NewCmdArchive(f)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	var args []string
	if spec.nameSet {
		args = append(args, spec.name)
	}
	if spec.force {
		args = append(args, "--force")
	}
	cmd.SetArgs(args)
	cmd.SetOut(&c.h.out)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}

func (c *cmdRunner) outputContains(want string) {
	c.h.t.Helper()
	if !strings.Contains(c.h.out.String(), want) {
		c.h.t.Fatalf("output does not contain %q\ngot: %q", want, c.h.out.String())
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
