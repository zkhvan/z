package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zkhvan/z/pkg/config"
)

func TestDir_reports_the_loaded_directory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("workspaces:\n  root: /tmp/ws\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.NewWithDir(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Dir() != dir {
		t.Fatalf("Dir() = %q, want %q", cfg.Dir(), dir)
	}
}

// A fresh install has no config.yaml but still needs defaults derived from the
// directory, so the early return must record it too.
func TestDir_is_reported_without_a_config_file(t *testing.T) {
	dir := t.TempDir()

	cfg, err := config.NewWithDir(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Dir() != dir {
		t.Fatalf("Dir() = %q, want %q", cfg.Dir(), dir)
	}
}

func TestConfigDir_override_is_used_verbatim(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(config.EnvConfigDir, dir)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("projects:\n  root: /tmp/p\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	// Verbatim: no implicit z/ subdir is appended.
	if cfg.Dir() != dir {
		t.Fatalf("Dir() = %q, want %q", cfg.Dir(), dir)
	}
	if got := cfg.String("projects.root"); got != "/tmp/p" {
		t.Fatalf("projects.root = %q, want the overridden config", got)
	}
}

// The directory must exist; a missing config.yaml inside it is a fresh install.
func TestConfigDir_override_without_a_config_file(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(config.EnvConfigDir, dir)

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Dir() != dir {
		t.Fatalf("Dir() = %q, want %q", cfg.Dir(), dir)
	}
}

// A mistyped path must not resolve to the default roots, which are the user's
// real ~/Projects and ~/Workspaces.
func TestConfigDir_override_that_does_not_exist_fails(t *testing.T) {
	t.Setenv(config.EnvConfigDir, filepath.Join(t.TempDir(), "nope"))

	_, err := config.New()

	assertErrorContains(t, err, "does not exist")
	assertErrorContains(t, err, config.EnvConfigDir)
}

func TestConfigDir_relative_override_fails(t *testing.T) {
	t.Setenv(config.EnvConfigDir, "./scratch")

	_, err := config.New()

	assertErrorContains(t, err, "absolute path")
}

func TestConfigDir_override_pointing_at_a_file_fails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("projects:\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv(config.EnvConfigDir, path)

	_, err := config.New()

	assertErrorContains(t, err, "not a directory")
}

func TestConfigDir_empty_override_falls_back_to_the_default(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvConfigDir, "")
	t.Setenv("XDG_CONFIG_HOME", home)

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	// The default resolution appends the z/ subdir; the override does not.
	if want := filepath.Join(home, "z"); cfg.Dir() != want {
		t.Fatalf("Dir() = %q, want %q", cfg.Dir(), want)
	}
}

// An exported override must not be able to redirect a caller that named its
// own directory — the whole test suite passes explicit dirs.
func TestConfigDir_explicit_dir_beats_the_override(t *testing.T) {
	explicit := t.TempDir()
	t.Setenv(config.EnvConfigDir, t.TempDir())

	cfg, err := config.NewWithDir(explicit)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Dir() != explicit {
		t.Fatalf("Dir() = %q, want the explicit dir %q", cfg.Dir(), explicit)
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err, want)
	}
}
