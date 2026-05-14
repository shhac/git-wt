package wt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveParentDir_EmptyOverrideUsesDefault(t *testing.T) {
	got, err := ResolveParentDir("/repo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "/repo/.worktrees"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveParentDir_AbsoluteOverrideUnchanged(t *testing.T) {
	got, err := ResolveParentDir("/repo", "/elsewhere/trees")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "/elsewhere/trees"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveParentDir_RelativeOverrideResolvedAgainstMainRoot(t *testing.T) {
	got, err := ResolveParentDir("/repo", "trees")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "/repo/trees"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveParentDir_TemplateExpansion(t *testing.T) {
	cases := []struct {
		name, mainRoot, override, want string
	}{
		{
			name:     "sibling pattern",
			mainRoot: "/u/p/myrepo",
			override: "${repoParent}/${repo}.worktrees",
			want:     "/u/p/myrepo.worktrees",
		},
		{
			name:     "absolute via repoPath",
			mainRoot: "/u/p/myrepo",
			override: "${repoPath}/subdir/wt",
			want:     "/u/p/myrepo/subdir/wt",
		},
		{
			name:     "relative post-expansion joins mainRoot",
			mainRoot: "/u/p/myrepo",
			override: "${repo}-trees",
			want:     "/u/p/myrepo/myrepo-trees",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveParentDir(c.mainRoot, c.override)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestResolveParentDir_UnknownTemplateVarErrors(t *testing.T) {
	_, err := ResolveParentDir("/repo", "${typo}/trees")
	if err == nil {
		t.Fatal("expected error for unknown template var")
	}
	if !strings.Contains(err.Error(), "typo") {
		t.Errorf("error should name the bad var: %v", err)
	}
}

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
