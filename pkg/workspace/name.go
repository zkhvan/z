package workspace

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidName wraps every error returned by ValidateName and
// ValidateDefinitionName; the kind of name follows it.
var ErrInvalidName = errors.New("invalid")

// ValidateName requires a non-empty single path segment with no separators,
// no leading dot, and not "." or "..".
func ValidateName(name string) error {
	return validateName("workspace name", name)
}

// ValidateDefinitionName applies the same rules — both names are dirnames — but
// says which kind of name was rejected.
func ValidateDefinitionName(name string) error {
	return validateName("definition name", name)
}

func validateName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("%w %s: must not be empty", ErrInvalidName, kind)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("%w %s: must not be %q", ErrInvalidName, kind, name)
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("%w %s: must not start with a dot: %q", ErrInvalidName, kind, name)
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("%w %s: must be a single path segment (no separators): %q", ErrInvalidName, kind, name)
	}
	return nil
}

// IsInvalidName reports whether err wraps ErrInvalidName.
func IsInvalidName(err error) bool {
	return errors.Is(err, ErrInvalidName)
}
