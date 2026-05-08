package wt

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CaseInsensitiveFilesystem is a best-effort guess at whether the OS-default
// filesystem at parent treats names case-insensitively. We use this to gate
// the collision check; the cost of a false positive is a confusing error, and
// the cost of a false negative is silent path collisions on macOS/Windows.
//
// We assume macOS (APFS default) and Windows (NTFS default) are
// case-insensitive; Linux (ext4 etc.) case-sensitive.
func CaseInsensitiveFilesystem() bool {
	switch runtime.GOOS {
	case "darwin", "windows":
		return true
	default:
		return false
	}
}

// FindCaseCollision walks the directory chain from parent down through each
// component of branch, looking for an existing entry whose name matches a
// component case-insensitively but not exactly. Returns the offending path,
// or "" if no collision.
//
// Returns "" without scanning on case-sensitive filesystems (Linux), where
// "Paul" and "paul" can legitimately coexist.
//
// Branch names with slashes are walked component-by-component so we catch
// "Paul/feat" colliding with an existing "paul/" directory.
func FindCaseCollision(parent, branch string) (string, error) {
	if !CaseInsensitiveFilesystem() {
		return "", nil
	}
	return walkCaseCollision(parent, branch)
}

// walkCaseCollision is the OS-independent inner loop, exposed so tests can
// exercise the logic directly without the OS gate.
func walkCaseCollision(parent, branch string) (string, error) {
	parts := strings.Split(filepath.FromSlash(branch), string(filepath.Separator))
	cur := parent
	for _, part := range parts {
		entries, err := os.ReadDir(cur)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", nil
			}
			return "", err
		}
		for _, entry := range entries {
			if entry.Name() != part && strings.EqualFold(entry.Name(), part) {
				return filepath.Join(cur, entry.Name()), nil
			}
		}
		cur = filepath.Join(cur, part)
		if !pathExists(cur) {
			return "", nil
		}
	}
	return "", nil
}
