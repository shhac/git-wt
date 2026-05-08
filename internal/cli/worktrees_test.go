package cli

import (
	"testing"

	"github.com/shhac/git-wt/internal/wt"
)

func TestFindByBranch_Exact(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/p/main", Branch: "main"},
		{Path: "/p/feat", Branch: "feat"},
	}
	got := findByBranch(wts, "feat")
	if got == nil || got.Branch != "feat" {
		t.Errorf("expected feat, got %+v", got)
	}
}

func TestFindByBranch_UniqueSuffix(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/p/main", Branch: "main"},
		{Path: "/p/auth", Branch: "paul/auth"},
	}
	got := findByBranch(wts, "auth")
	if got == nil || got.Branch != "paul/auth" {
		t.Errorf("expected paul/auth, got %+v", got)
	}
}

func TestFindByBranch_AmbiguousSuffix(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/p/a", Branch: "paul/auth"},
		{Path: "/p/b", Branch: "jane/auth"},
	}
	got := findByBranch(wts, "auth")
	if got != nil {
		t.Errorf("expected nil for ambiguous suffix, got %+v", got)
	}
}

func TestFindByBranch_ExactBeatsSuffix(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/p/auth", Branch: "auth"},
		{Path: "/p/paul-auth", Branch: "paul/auth"},
	}
	got := findByBranch(wts, "auth")
	if got == nil || got.Path != "/p/auth" {
		t.Errorf("expected exact match /p/auth, got %+v", got)
	}
}

func TestFindByBranch_NotFound(t *testing.T) {
	wts := []wt.Worktree{{Path: "/p/main", Branch: "main"}}
	got := findByBranch(wts, "nope")
	if got != nil {
		t.Errorf("expected nil for unknown branch, got %+v", got)
	}
}

func TestFindByBranch_DetachedSkipped(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/p/det", Branch: "", Detached: true},
		{Path: "/p/feat", Branch: "feat"},
	}
	got := findByBranch(wts, "feat")
	if got == nil || got.Branch != "feat" {
		t.Errorf("expected feat, got %+v", got)
	}
}

func TestFilterOutCurrent(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/p/main", Branch: "main"},
		{Path: "/p/feat", Branch: "feat"},
		{Path: "/p/other", Branch: "other"},
	}
	cur := &wts[1]
	got := filterOutCurrent(wts, cur)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	for _, t2 := range got {
		if t2.Path == cur.Path {
			t.Errorf("current %q should have been removed", cur.Path)
		}
	}
}

func TestFilterOutCurrent_NilCurrentReturnsAll(t *testing.T) {
	wts := []wt.Worktree{{Path: "/p/main"}, {Path: "/p/feat"}}
	got := filterOutCurrent(wts, nil)
	if len(got) != 2 {
		t.Errorf("len = %d, want 2 (no filter)", len(got))
	}
}

func TestFilterRemovable(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/p/main", Branch: "main"},
		{Path: "/p/feat", Branch: "feat"},
		{Path: "/p/other", Branch: "other"},
	}
	got := filterRemovable(wts, &wt.RepoInfo{MainRoot: "/p/main"})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	for _, t2 := range got {
		if t2.Path == "/p/main" {
			t.Errorf("main worktree must be filtered out, got %+v", t2)
		}
	}
}

func TestFilterRemovable_OnlyMain(t *testing.T) {
	wts := []wt.Worktree{{Path: "/p/main"}}
	got := filterRemovable(wts, &wt.RepoInfo{MainRoot: "/p/main"})
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}
