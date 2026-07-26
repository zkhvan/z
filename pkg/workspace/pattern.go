package workspace

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// DefaultBranchPattern applies when a definition omits one. There is no
// meaningful "no pattern" behavior: every member needs a branch.
const DefaultBranchPattern = "{instance}"

const (
	placeholderInstance = "instance"
	placeholderRepo     = "repo"
)

// A dangling brace is left alone; only a well-formed {word} is a placeholder,
// and braces are legal in a branch name.
var placeholderRe = regexp.MustCompile(`\{([^{}]*)\}`)

// ValidateBranchPattern rejects placeholders outside the closed set. Widening
// the set is a deliberate act, so a typo cannot reach materialize as a branch
// literally named "feat/{user}".
func ValidateBranchPattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("branch pattern is empty")
	}

	var unknown []string
	seen := make(map[string]bool)
	for _, match := range placeholderRe.FindAllStringSubmatch(pattern, -1) {
		name := match[1]
		switch name {
		case placeholderInstance, placeholderRepo:
			continue
		}
		if !seen[name] {
			seen[name] = true
			unknown = append(unknown, match[0])
		}
	}

	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf(
			"branch pattern %q has unknown placeholder %s (known: {instance}, {repo})",
			pattern, strings.Join(unknown, ", "))
	}

	return nil
}

// ResolveBranch substitutes the pattern for one member. The instance name is
// never sanitized: a result that is not a valid branch is an error the caller
// reports, not a silent rename.
func ResolveBranch(pattern, instance, repo string) (string, error) {
	if err := ValidateBranchPattern(pattern); err != nil {
		return "", err
	}

	branch := placeholderRe.ReplaceAllStringFunc(pattern, func(match string) string {
		switch placeholderRe.FindStringSubmatch(match)[1] {
		case placeholderInstance:
			return instance
		case placeholderRepo:
			return baseName(repo)
		}
		return match
	})

	if err := validateBranchName(branch); err != nil {
		return "", fmt.Errorf(
			"member %q: branch pattern %q produced %q: %w",
			repo, pattern, branch, err)
	}

	return branch, nil
}
