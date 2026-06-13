//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package wt

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// TestDeleteTree_ImmutableFlag pins the uchg case: a user-immutable file
// fails unlink with EPERM until the flag is cleared.
func TestDeleteTree_ImmutableFlag(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tree")
	mkTree(t, root, "ignored/pinned.txt")
	pinned := filepath.Join(root, "ignored", "pinned.txt")
	if err := unix.Chflags(pinned, unix.UF_IMMUTABLE); err != nil {
		t.Skipf("cannot set uchg here: %v", err)
	}
	t.Cleanup(func() { _ = unix.Chflags(pinned, 0) }) // in case the delete fails

	if err := DeleteTree(root, nil); err != nil {
		t.Fatalf("DeleteTree: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("expected %s to be gone, stat err = %v", root, err)
	}
}
