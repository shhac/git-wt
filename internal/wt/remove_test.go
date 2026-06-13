package wt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkTree(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(root, filepath.FromSlash(p))
		if strings.HasSuffix(p, "/") {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDeleteTree_PlainTree(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tree")
	mkTree(t, root, "a.txt", "sub/b.txt", "sub/deep/c.txt", "empty/")
	if err := DeleteTree(root, nil); err != nil {
		t.Fatalf("DeleteTree: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("expected %s to be gone, stat err = %v", root, err)
	}
}

// TestDeleteTree_ReadOnlyDir pins the fix for the worst `git worktree
// remove` failure mode: a directory without owner write/exec permission
// kills git's recursive delete mid-way, stranding a half-removed worktree.
func TestDeleteTree_ReadOnlyDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tree")
	mkTree(t, root, "ignored/rodir/locked.txt", "ignored/other.txt")
	roDir := filepath.Join(root, "ignored", "rodir")
	if err := os.Chmod(filepath.Join(roDir, "locked.txt"), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(roDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(roDir, 0o755) }) // in case the delete fails

	if err := DeleteTree(root, nil); err != nil {
		t.Fatalf("DeleteTree: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("expected %s to be gone, stat err = %v", root, err)
	}
}

func TestDeleteTree_DoesNotFollowSymlinks(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside")
	mkTree(t, outside, "precious.txt")
	root := filepath.Join(dir, "tree")
	mkTree(t, root, "a.txt")
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}

	if err := DeleteTree(root, nil); err != nil {
		t.Fatalf("DeleteTree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "precious.txt")); err != nil {
		t.Errorf("symlink target must survive: %v", err)
	}
}

func TestDeleteTree_MissingRootIsNoop(t *testing.T) {
	if err := DeleteTree(filepath.Join(t.TempDir(), "nope"), nil); err != nil {
		t.Fatalf("missing root should be a no-op, got %v", err)
	}
}

func TestDeleteTree_ReportsProgress(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tree")
	var paths []string
	for i := 0; i < 300; i++ {
		paths = append(paths, fmt.Sprintf("sub%d/f%d.txt", i%10, i))
	}
	mkTree(t, root, paths...)

	final := 0
	calls := 0
	err := DeleteTree(root, func(n int) { final = n; calls++ })
	if err != nil {
		t.Fatalf("DeleteTree: %v", err)
	}
	if calls == 0 {
		t.Fatalf("expected at least the final progress call")
	}
	if final == 0 {
		t.Errorf("final progress count should be > 0")
	}
}

func TestRenameAside(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "wt")
	mkTree(t, dir, "f.txt")
	aside, err := RenameAside(dir)
	if err != nil {
		t.Fatalf("RenameAside: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("original path should be free")
	}
	if _, err := os.Stat(filepath.Join(aside, "f.txt")); err != nil {
		t.Errorf("contents should have moved with the rename: %v", err)
	}
	if !strings.Contains(filepath.Base(aside), ".removing-") {
		t.Errorf("aside name %q should mark the dir as mid-removal", aside)
	}
}

func TestRenameAside_MissingDir(t *testing.T) {
	if _, err := RenameAside(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatalf("expected error for missing dir")
	}
}
