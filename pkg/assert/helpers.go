package assert

import (
	"testing"
)

func Error(t *testing.T, got error, want error) {
	if got == nil {
		t.Fatalf("expected error, got nil")
	}
	if got.Error() != want.Error() {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func NoError(t *testing.T, got error) {
	if got != nil {
		t.Fatalf("expected no error, got %v", got)
	}
}

func EqualString(t *testing.T, got, want string) {
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
