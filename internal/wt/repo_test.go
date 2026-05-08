package wt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInProgressOp_CleanRepo(t *testing.T) {
	dir := t.TempDir()
	if got := inProgressOp(dir); got != "" {
		t.Errorf("got %q, want empty (clean repo)", got)
	}
}

func TestInProgressOp_DetectsEachMarker(t *testing.T) {
	cases := []struct {
		marker string
		want   string
	}{
		{"MERGE_HEAD", "merge"},
		{"rebase-merge", "rebase"},
		{"rebase-apply", "rebase"},
		{"CHERRY_PICK_HEAD", "cherry-pick"},
		{"REVERT_HEAD", "revert"},
		{"BISECT_LOG", "bisect"},
	}
	for _, c := range cases {
		t.Run(c.marker, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, c.marker)
			// rebase-* are directories in real git; the others are files.
			// inProgressOp uses pathExists which handles both.
			if c.marker == "rebase-merge" || c.marker == "rebase-apply" {
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := inProgressOp(dir); got != c.want {
				t.Errorf("inProgressOp with %s present = %q, want %q", c.marker, got, c.want)
			}
		})
	}
}

func TestInProgressOp_FirstWinsWhenMultiple(t *testing.T) {
	// Multiple markers shouldn't happen in real life, but if they do the
	// fixed iteration order means MERGE_HEAD beats CHERRY_PICK_HEAD.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MERGE_HEAD"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CHERRY_PICK_HEAD"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := inProgressOp(dir); got != "merge" {
		t.Errorf("got %q, want merge (MERGE_HEAD comes first)", got)
	}
}

func TestInProgressOp_MissingDir(t *testing.T) {
	// pathExists returns false for non-existent paths — overall result is "clean".
	if got := inProgressOp(filepath.Join(t.TempDir(), "does-not-exist")); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
