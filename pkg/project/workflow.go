package project

import (
	"fmt"
	"path/filepath"
	"regexp"
)

// ValidateBranch checks whether name matches the given regex pattern.
// An empty pattern means no enforcement (always returns nil).
// Returns an error if the pattern is invalid or the name does not match.
func ValidateBranch(name, pattern string) error {
	if pattern == "" {
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid branch pattern %q: %w", pattern, err)
	}
	if !re.MatchString(name) {
		return fmt.Errorf("branch %q does not match pattern %q", name, pattern)
	}
	return nil
}

// ValidateCommit checks whether message matches the given regex pattern.
// An empty pattern means no enforcement (always returns nil).
// Returns an error if the pattern is invalid or the message does not match.
func ValidateCommit(message, pattern string) error {
	if pattern == "" {
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid commit pattern %q: %w", pattern, err)
	}
	if !re.MatchString(message) {
		return fmt.Errorf("commit message does not match pattern %q", pattern)
	}
	return nil
}

// IsProtected reports whether branch matches any entry in the protected list.
// Entries support glob patterns via filepath.Match (e.g., "release/*").
func IsProtected(branch string, protectedBranches []string) bool {
	for _, pattern := range protectedBranches {
		if pattern == branch {
			return true
		}
		if matched, _ := filepath.Match(pattern, branch); matched {
			return true
		}
	}
	return false
}
