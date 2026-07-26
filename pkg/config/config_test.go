package config_test

import (
	"os"
	"path/filepath"
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
