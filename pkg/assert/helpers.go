package assert

import (
	"testing"
)

func Error(t *testing.T, got error, want error) {
	if got == nil {
		t.Errorf("expected error, got nil")
	}
	if got.Error() != want.Error() {
		t.Errorf("expected error %q, got %q", want, got)
	}
}

func NoError(t *testing.T, got error) {
	if got != nil {
		t.Errorf("expected no error, got %v", got)
	}
}

func EqualString(t *testing.T, got, want string) {
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}
