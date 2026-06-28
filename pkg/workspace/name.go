package workspace

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidName wraps every error returned by ValidateName.
var ErrInvalidName = errors.New("invalid workspace name")

// ValidateName requires a non-empty single path segment with no separators,
// no leading dot, and not "." or "..".
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: must not be empty", ErrInvalidName)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("%w: must not be %q", ErrInvalidName, name)
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("%w: must not start with a dot: %q", ErrInvalidName, name)
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("%w: must be a single path segment (no separators): %q", ErrInvalidName, name)
	}
	return nil
}

// IsInvalidName reports whether err wraps ErrInvalidName.
func IsInvalidName(err error) bool {
	return errors.Is(err, ErrInvalidName)
}
