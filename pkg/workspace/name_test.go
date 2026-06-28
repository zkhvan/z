package workspace_test

import (
	"errors"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestValidateName_Valid(t *testing.T) {
	cases := []string{
		"my-workspace",
		"foo123",
		"login-feature",
		"a",
	}

	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			assert.NoError(t, workspace.ValidateName(name))
		})
	}
}

func TestValidateName_Invalid(t *testing.T) {
	cases := []string{
		"",        // empty
		"foo/bar", // forward slash
		".hidden", // leading dot
		".",       // dot
		"..",      // double-dot
	}

	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			err := workspace.ValidateName(name)
			if err == nil {
				t.Fatalf("expected error for name %q, got nil", name)
			}
			if !errors.Is(err, workspace.ErrInvalidName) {
				t.Fatalf("expected error to wrap ErrInvalidName, got %v", err)
			}
			if !workspace.IsInvalidName(err) {
				t.Fatalf("expected IsInvalidName to report true for %v", err)
			}
		})
	}
}
