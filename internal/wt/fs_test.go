package wt

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopyTree_SingleFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "out", "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CopyTree(src, dst); err != nil {
		t.Fatalf("CopyTree: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("dst contents = %q, want hello", got)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("dst mode = %v, want 0o600", info.Mode().Perm())
	}
}

func TestCopyTree_Directory(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")

	// src/
	//   a.txt
	//   nested/
	//     b.txt
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyTree(src, dst); err != nil {
		t.Fatalf("CopyTree: %v", err)
	}

	if got, _ := os.ReadFile(filepath.Join(dst, "a.txt")); string(got) != "a" {
		t.Errorf("dst/a.txt = %q, want a", got)
	}
	if got, _ := os.ReadFile(filepath.Join(dst, "nested", "b.txt")); string(got) != "b" {
		t.Errorf("dst/nested/b.txt = %q, want b", got)
	}
}

func TestCopyTree_Symlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("target-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "link")
	if err := os.Symlink("target.txt", src); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(root, "out", "link-dst")
	if err := CopyTree(src, dst); err != nil {
		t.Fatalf("CopyTree: %v", err)
	}

	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("dst should be a symlink (mode=%v)", info.Mode())
	}
	gotTarget, err := os.Readlink(dst)
	if err != nil {
		t.Fatal(err)
	}
	if gotTarget != "target.txt" {
		t.Errorf("readlink = %q, want target.txt", gotTarget)
	}
}

func TestCopyTree_MissingSource(t *testing.T) {
	dir := t.TempDir()
	err := CopyTree(filepath.Join(dir, "does-not-exist"), filepath.Join(dir, "dst"))
	if err == nil {
		t.Errorf("expected error for missing source, got nil")
	}
}

func TestPathExists(t *testing.T) {
	dir := t.TempDir()
	if !PathExists(dir) {
		t.Errorf("PathExists(tempdir) = false, want true")
	}
	if PathExists(filepath.Join(dir, "nope")) {
		t.Errorf("PathExists(missing) = true, want false")
	}
	// Broken symlink — should be treated as existing (Lstat succeeds).
	if runtime.GOOS != "windows" {
		link := filepath.Join(dir, "broken")
		if err := os.Symlink("/totally-missing", link); err != nil {
			t.Fatal(err)
		}
		if !PathExists(link) {
			t.Errorf("PathExists(broken-symlink) = false, want true")
		}
	}
}
