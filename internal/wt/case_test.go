package wt

import (
	"os"
	"path/filepath"
	"testing"
)

// walkCaseCollision is the OS-gate-free inner logic. We test it directly so
// the Linux CI can exercise the case-collision logic that, on macOS/Windows,
// is the user-visible behavior.

func TestWalkCaseCollision_LeafCollision(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "MyBranch"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := walkCaseCollision(root, "mybranch")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, "MyBranch") {
		t.Errorf("got %q, want %q", got, filepath.Join(root, "MyBranch"))
	}
}

func TestWalkCaseCollision_NoCollision(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "feature-a"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := walkCaseCollision(root, "feature-b")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got %q, want empty (no collision)", got)
	}
}

func TestWalkCaseCollision_NamespaceParentCollision(t *testing.T) {
	// Existing: paul/Foo
	// New:      Paul/bar
	// Expected: collision on the `Paul` parent dir against existing `paul`
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "paul", "Foo"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := walkCaseCollision(root, "Paul/bar")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, "paul") {
		t.Errorf("got %q, want %q", got, filepath.Join(root, "paul"))
	}
}

func TestWalkCaseCollision_DeepNamespace(t *testing.T) {
	// Existing: a/b/c
	// New:      A/B/D
	// Expected: collision on the first component (we stop at the first hit)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b", "c"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := walkCaseCollision(root, "A/B/D")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, "a") {
		t.Errorf("got %q, want %q", got, filepath.Join(root, "a"))
	}
}

func TestWalkCaseCollision_ParentMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")
	got, err := walkCaseCollision(root, "feat")
	if err != nil {
		t.Errorf("unexpected error for missing parent: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestWalkCaseCollision_ExactNameNotConsideredCollision(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "feat"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Same case = same name, not a collision (the caller's "path already
	// exists" check is what handles this case).
	got, err := walkCaseCollision(root, "feat")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got %q, want empty (exact match isn't a case collision)", got)
	}
}
