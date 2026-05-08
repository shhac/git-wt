package wt

import (
	"testing"
	"time"
)

func TestParsePorcelain(t *testing.T) {
	in := "worktree /path/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
		"worktree /path/feat\nHEAD def456\nbranch refs/heads/feature/x\n\n" +
		"worktree /path/det\nHEAD 789abc\ndetached\n\n" +
		"worktree /path/lock\nHEAD 111\nbranch refs/heads/locked\nlocked\n\n"

	got := parsePorcelain(in)
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}

	if got[0].Branch != "main" || got[0].Path != "/path/main" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[1].Branch != "feature/x" {
		t.Errorf("got[1].Branch = %q, want feature/x", got[1].Branch)
	}
	if !got[2].Detached || got[2].Branch != "" {
		t.Errorf("got[2] = %+v, want detached", got[2])
	}
	if !got[3].Locked {
		t.Errorf("got[3] should be locked")
	}
}

func TestSortByModTime(t *testing.T) {
	now := time.Now()
	wts := []Worktree{
		{Path: "/a", ModTime: now.Add(-3 * time.Hour)},
		{Path: "/b", ModTime: now.Add(-1 * time.Hour)},
		{Path: "/c", ModTime: time.Time{}}, // zero — should sort last
		{Path: "/d", ModTime: now.Add(-2 * time.Hour)},
	}
	SortByModTime(wts)
	wantOrder := []string{"/b", "/d", "/a", "/c"}
	for i, w := range wts {
		if w.Path != wantOrder[i] {
			t.Errorf("position %d: got %s, want %s", i, w.Path, wantOrder[i])
		}
	}
}

func TestParentDirName(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/Users/paul/projects/repo", "projects"},
		{"/Users/paul/projects/repo-trees/feat", "repo-trees"},
		{"/Users/paul/projects/repo-trees/paul/feat", "paul"},
		{"/foo", "/"},
	}
	for _, c := range cases {
		got := Worktree{Path: c.path}.ParentDirName()
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.path, got, c.want)
		}
	}
}

func TestValidateBranchName(t *testing.T) {
	good := []string{"main", "feature/auth", "paul/wip", "fix-123", "feat_foo"}
	for _, s := range good {
		if err := ValidateBranchName(s); err != nil {
			t.Errorf("ValidateBranchName(%q) returned error: %v", s, err)
		}
	}
	bad := []string{
		"",
		"-startswithdash",
		"/leading-slash",
		"trailing-slash/",
		"double//slash",
		"has space",
		"has..dotdot",
		"name.lock",
		"control\x01char",
		"caret^",
		"colon:foo",
		"backslash\\",
		"@{badref}",
	}
	for _, s := range bad {
		if err := ValidateBranchName(s); err == nil {
			t.Errorf("ValidateBranchName(%q) should have errored", s)
		}
	}
}
