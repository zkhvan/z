package workspace

import (
	"fmt"
	"strings"
	"unicode"
)

// Member is one repository in a workspace: owner/repo@branch[:base_ref].
type Member struct {
	Repo   string
	Branch string
	// BaseRef is empty to resolve the default branch at materialize time.
	BaseRef string
	// State is derived for listed instances and ignored when writing manifests.
	State MemberState
}

// BaseName is the worktree directory name (the part after the last "/").
func (m Member) BaseName() string {
	parts := strings.Split(m.Repo, "/")
	return parts[len(parts)-1]
}

// ValidateBranch and ValidateBaseRef expose the rules ValidateMembers applies
// per field, so an interactive caller can reject a value as it is typed and
// never collect one the structural path would refuse.
func ValidateBranch(branch string) error {
	return validateBranchName(branch)
}

func ValidateBaseRef(baseRef string) error {
	return validateBaseRef(baseRef)
}

// ValidateMembers requires valid members with unique worktree directory names.
func ValidateMembers(members []Member) error {
	if len(members) == 0 {
		return fmt.Errorf("workspace must have at least one member")
	}

	seen := make(map[string]string) // baseName -> repo
	for _, m := range members {
		if err := validateRepoID(m.Repo); err != nil {
			return fmt.Errorf("invalid member repo %q: %w", m.Repo, err)
		}
		if err := validateBranchName(m.Branch); err != nil {
			return fmt.Errorf("invalid member branch %q: %w", m.Branch, err)
		}
		if err := validateBaseRef(m.BaseRef); err != nil {
			return fmt.Errorf("invalid member base ref %q: %w", m.BaseRef, err)
		}

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
	if err := validateRepoID(repo); err != nil {
		return Member{}, fmt.Errorf("invalid member %q: %w", s, err)
	}

	var branch, baseRef string
	if colonIdx := strings.Index(rest, ":"); colonIdx >= 0 {
		branch = rest[:colonIdx]
		baseRef = rest[colonIdx+1:]
	} else {
		branch = rest
	}

	if err := validateBranchName(branch); err != nil {
		return Member{}, fmt.Errorf("invalid member %q: %w", s, err)
	}
	if err := validateBaseRef(baseRef); err != nil {
		return Member{}, fmt.Errorf("invalid member %q: %w", s, err)
	}

	return Member{
		Repo:    repo,
		Branch:  branch,
		BaseRef: baseRef,
	}, nil
}

func validateBaseRef(baseRef string) error {
	if strings.HasPrefix(baseRef, "-") || strings.IndexFunc(baseRef, unicode.IsControl) >= 0 {
		return fmt.Errorf("invalid base ref %q", baseRef)
	}
	return nil
}

func validateBranchName(branch string) error {
	if branch == "" {
		return fmt.Errorf("branch is empty")
	}
	if strings.HasPrefix(branch, "-") || branch == "HEAD" {
		return fmt.Errorf("invalid branch %q", branch)
	}
	if strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") || strings.Contains(branch, "//") {
		return fmt.Errorf("invalid branch %q", branch)
	}
	if strings.HasSuffix(branch, ".") || strings.Contains(branch, "..") || strings.Contains(branch, "@{") {
		return fmt.Errorf("invalid branch %q", branch)
	}
	for _, part := range strings.Split(branch, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return fmt.Errorf("invalid branch %q", branch)
		}
	}
	if strings.IndexFunc(branch, invalidBranchRune) >= 0 {
		return fmt.Errorf("invalid branch %q", branch)
	}
	return nil
}

func invalidBranchRune(r rune) bool {
	return r <= ' ' || r == 0x7f || strings.ContainsRune("~^:?*[\\", r)
}

func validateRepoID(repo string) error {
	if strings.IndexFunc(repo, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 {
		return fmt.Errorf("repo contains whitespace or control characters")
	}

	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("repo must be owner/repo")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("repo contains invalid path segment %q", part)
		}
		if strings.ContainsRune(part, '\\') {
			return fmt.Errorf("repo contains invalid path separator")
		}
	}
	return nil
}
