package materialize_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/cmd/workspace/materialize"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/config"
	"github.com/zkhvan/z/pkg/iolib"
)

func TestMaterialize_missing_instance(t *testing.T) {
	cmd, harness := newCommandTest(t)
	harness.config(withWorkspaceRoot())

	err := cmd.run(cmd.withName("missing"))

	assertErrorContains(t, err, "has not been created")
}

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

type runSpec struct {
	name    string
	nameSet bool
}

type runOption func(*runSpec)

func (*cmdRunner) withName(name string) runOption {
	return func(s *runSpec) { s.name = name; s.nameSet = true }
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

	cmd := materialize.NewCmdMaterialize(f)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	var args []string
	if spec.nameSet {
		args = append(args, spec.name)
	}
	cmd.SetArgs(args)
	cmd.SetOut(&c.h.out)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
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
