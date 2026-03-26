package tmux

import (
	"context"
	"errors"
	"strings"
)

const (
	PopupPrefix    = "_popup_"
	PopupSeparator = "__"
)

func ToPopupSessionName(parentSession, popupName string) string {
	return PopupPrefix + parentSession + PopupSeparator + popupName
}

func IsPopupSession(sessionName string) bool {
	return strings.HasPrefix(sessionName, PopupPrefix)
}

// ExtractPopupName extracts the popup name from a full popup session name,
// given the parent session name. Returns the popup name and true if the
// session belongs to the parent, or empty string and false otherwise.
func ExtractPopupName(sessionName, parentSession string) (string, bool) {
	prefix := PopupPrefix + parentSession + PopupSeparator
	if !strings.HasPrefix(sessionName, prefix) {
		return "", false
	}
	return strings.TrimPrefix(sessionName, prefix), true
}

// ExtractPopupParent extracts the parent session name from a popup session
// name by splitting on the double-underscore separator. Returns the parent
// name and true, or empty string and false if not a popup session.
func ExtractPopupParent(sessionName string) (string, bool) {
	if !IsPopupSession(sessionName) {
		return "", false
	}
	rest := strings.TrimPrefix(sessionName, PopupPrefix)
	idx := strings.LastIndex(rest, PopupSeparator)
	if idx < 0 {
		return "", false
	}
	return rest[:idx], true
}

// KillPopups kills all popup sessions associated with the given parent session.
func KillPopups(ctx context.Context, parentSessionName string) error {
	sessions, err := ListSessions(ctx, nil)
	if err != nil {
		return err
	}
	var errs []error
	for _, session := range sessions {
		if _, ok := ExtractPopupName(session.Name, parentSessionName); ok {
			errs = append(errs, KillSession(ctx, session))
		}
	}
	return errors.Join(errs...)
}
