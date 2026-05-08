package wt

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidateBranchName rejects names with shell metacharacters, path-traversal,
// or characters git itself disallows. Allows slashes (namespaces).
func ValidateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name is empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("branch name is too long (max 255)")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("branch name cannot start with '-'")
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("branch name cannot start or end with '/'")
	}
	if strings.Contains(name, "//") {
		return fmt.Errorf("branch name cannot contain consecutive '/'")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("branch name cannot contain '..'")
	}
	if strings.HasSuffix(name, ".lock") {
		return fmt.Errorf("branch name cannot end with '.lock'")
	}
	for _, r := range name {
		switch {
		case r < 0x20 || r == 0x7f:
			return fmt.Errorf("branch name contains a control character")
		case unicode.IsSpace(r):
			return fmt.Errorf("branch name cannot contain whitespace")
		}
		if strings.ContainsRune("~^:?*[\\@", r) {
			return fmt.Errorf("branch name cannot contain %q", r)
		}
	}
	if strings.Contains(name, "@{") {
		return fmt.Errorf("branch name cannot contain '@{'")
	}
	return nil
}
