package wt

import (
	"os"
	"path/filepath"
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

func TestSortByModTime_AllZero(t *testing.T) {
	wts := []Worktree{
		{Path: "/a"},
		{Path: "/b"},
		{Path: "/c"},
	}
	SortByModTime(wts)
	// stable order preserved
	for i, want := range []string{"/a", "/b", "/c"} {
		if wts[i].Path != want {
			t.Errorf("position %d: got %s, want %s", i, wts[i].Path, want)
		}
	}
}

func TestDisplayPath(t *testing.T) {
	const main = "/Users/paul/projects/backend"
	const trees = "/Users/paul/projects/backend/.worktrees"
	cases := []struct {
		name, path, want string
	}{
		{"main worktree → repo basename", main, "backend"},
		{"inside trees dir, flat → #abuja", trees + "/abuja", "#abuja"},
		{"inside trees dir, namespaced → #paul/feat/aaa", trees + "/paul/feat/aaa", "#paul/feat/aaa"},
		{"inside repo, outside trees → rel-to-repo", main + "/.conductor/abuja", ".conductor/abuja"},
		{"inside repo, deeper outside trees → rel-to-repo", main + "/sub/dir/x", "sub/dir/x"},
		{"outside repo → absolute", "/tmp/dddd", "/tmp/dddd"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Worktree{Path: c.path}.DisplayPath(main, trees)
			if got != c.want {
				t.Errorf("got %q, want %q (path=%s)", got, c.want, c.path)
			}
		})
	}
}

func TestDisplayPath_NoTreesDirGate(t *testing.T) {
	// If treesDir is empty, the # rule is skipped — falls through to repo-rel.
	const main = "/repo"
	got := Worktree{Path: main + "/sub"}.DisplayPath(main, "")
	if got != "sub" {
		t.Errorf("got %q, want sub", got)
	}
}

func TestCurrent_ExactMatch(t *testing.T) {
	tmp := t.TempDir()
	wts := []Worktree{
		{Path: tmp},
		{Path: t.TempDir()},
	}
	got := Current(wts, tmp)
	if got == nil || got.Path != tmp {
		t.Errorf("expected exact match for %s, got %+v", tmp, got)
	}
}

func TestCurrent_SubdirMatch(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "src", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	wts := []Worktree{{Path: tmp}}
	got := Current(wts, sub)
	if got == nil || got.Path != tmp {
		t.Errorf("expected match for subdir %s under worktree %s, got %+v", sub, tmp, got)
	}
}

func TestCurrent_DeepestNestedWins(t *testing.T) {
	// Bug #36 / v0.6.2: when one worktree is nested inside another, the
	// inner one should be reported as current (deepest match wins).
	outer := t.TempDir()
	inner := filepath.Join(outer, "inner")
	deepest := filepath.Join(inner, "subdir")
	if err := os.MkdirAll(deepest, 0o755); err != nil {
		t.Fatal(err)
	}
	wts := []Worktree{
		{Path: outer},
		{Path: inner},
	}
	got := Current(wts, deepest)
	if got == nil || got.Path != inner {
		t.Errorf("expected deepest match %q, got %+v", inner, got)
	}
}

func TestCurrent_NoMatchReturnsNil(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	wts := []Worktree{{Path: a}}
	got := Current(wts, b)
	if got != nil {
		t.Errorf("expected nil for non-matching dir, got %+v", got)
	}
}

func TestCurrent_EmptyDirReturnsNil(t *testing.T) {
	got := Current([]Worktree{{Path: "/anywhere"}}, "")
	if got != nil {
		t.Errorf("expected nil for empty dir, got %+v", got)
	}
}

func TestConstructPath(t *testing.T) {
	cases := []struct {
		parent string
		branch string
		want   string
	}{
		{"/parent", "feat", filepath.Join("/parent", "feat")},
		{"/parent", "paul/auth", filepath.Join("/parent", "paul", "auth")},
		{"/parent", "deep/nest/branch", filepath.Join("/parent", "deep", "nest", "branch")},
	}
	for _, c := range cases {
		got := ConstructPath(c.parent, c.branch)
		if got != c.want {
			t.Errorf("ConstructPath(%q, %q) = %q, want %q", c.parent, c.branch, got, c.want)
		}
	}
}

func TestTreesDirFor(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/home/user/repo", "/home/user/repo/.worktrees"},
		{"/repo", "/repo/.worktrees"},
	}
	for _, c := range cases {
		got := TreesDirFor(c.in)
		if got != c.want {
			t.Errorf("TreesDirFor(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
