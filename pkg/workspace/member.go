package workspace

import (
	"fmt"
	"strings"
)

// Member is one repository in a workspace: owner/repo@branch[:base_ref].
type Member struct {
	Repo   string
	Branch string
	// BaseRef is empty to resolve the default branch at materialize time.
	BaseRef string
}

// BaseName is the worktree directory name (the part after the last "/").
func (m Member) BaseName() string {
	parts := strings.Split(m.Repo, "/")
	return parts[len(parts)-1]
}

// ValidateMembers requires at least one member and rejects members that
// resolve to the same worktree directory.
func ValidateMembers(members []Member) error {
	if len(members) == 0 {
		return fmt.Errorf("workspace must have at least one member")
	}

	seen := make(map[string]string) // baseName -> repo
	for _, m := range members {
		base := m.BaseName()
		if prev, ok := seen[base]; ok {
			return fmt.Errorf("members %q and %q share worktree directory name %q", prev, m.Repo, base)
		}
		seen[base] = m.Repo
	}

	return nil
}

// ParseMember parses a member string in the form owner/repo@branch[:base_ref].
func ParseMember(s string) (Member, error) {
	atIdx := strings.Index(s, "@")
	if atIdx < 0 {
		return Member{}, fmt.Errorf("invalid member %q: missing '@' (expected owner/repo@branch[:base])", s)
	}

	repo := s[:atIdx]
	rest := s[atIdx+1:]

	if repo == "" {
		return Member{}, fmt.Errorf("invalid member %q: repo part is empty", s)
	}
	if !strings.Contains(repo, "/") {
		return Member{}, fmt.Errorf("invalid member %q: repo must be owner/repo", s)
	}

	var branch, baseRef string
	if colonIdx := strings.Index(rest, ":"); colonIdx >= 0 {
		branch = rest[:colonIdx]
		baseRef = rest[colonIdx+1:]
	} else {
		branch = rest
	}

	if branch == "" {
		return Member{}, fmt.Errorf("invalid member %q: branch is empty", s)
	}

	return Member{
		Repo:    repo,
		Branch:  branch,
		BaseRef: baseRef,
	}, nil
}
