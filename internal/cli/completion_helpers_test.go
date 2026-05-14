package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shhac/git-wt/internal/wt"
)

// branchesOnly strips the "\tdescription" suffix Cobra uses so the
// "did we pick the right branches?" assertions stay readable.
func branchesOnly(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i], _, _ = strings.Cut(s, "\t")
	}
	return out
}

func TestWorktreeBranchesForGo(t *testing.T) {
	mainRoot := "/repo"
	treesDir := "/repo/.worktrees"
	cases := []struct {
		name string
		wts  []wt.Worktree
		cur  *wt.Worktree
		want []string
	}{
		{
			name: "all branches when no current",
			wts: []wt.Worktree{
				{Branch: "paul/auth", Path: "/repo/.worktrees/paul/auth"},
				{Branch: "feat-x", Path: "/repo/.worktrees/feat-x"},
				{Branch: "main", Path: "/repo"},
			},
			want: []string{"feat-x", "main", "paul/auth"},
		},
		{
			name: "excludes current worktree by path",
			wts: []wt.Worktree{
				{Branch: "paul/auth", Path: "/repo/.worktrees/paul/auth"},
				{Branch: "feat-x", Path: "/repo/.worktrees/feat-x"},
			},
			cur:  &wt.Worktree{Branch: "feat-x", Path: "/repo/.worktrees/feat-x"},
			want: []string{"paul/auth"},
		},
		{
			name: "skips entries with no branch (detached HEAD)",
			wts: []wt.Worktree{
				{Branch: "", Path: "/repo/.worktrees/detached"},
				{Branch: "feat-x", Path: "/repo/.worktrees/feat-x"},
			},
			want: []string{"feat-x"},
		},
		{
			name: "empty input returns empty",
			wts:  nil,
			want: []string{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := branchesOnly(worktreeBranchesForGo(c.wts, c.cur, mainRoot, treesDir))
			if got == nil {
				got = []string{}
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestWorktreeBranchesForGo_IncludesDescription(t *testing.T) {
	wts := []wt.Worktree{
		// Inside trees dir → `#feat-x`.
		{Branch: "feat-x", Path: "/repo/.worktrees/feat-x"},
		// Main worktree → just the repo basename.
		{Branch: "main", Path: "/repo"},
	}
	got := worktreeBranchesForGo(wts, nil, "/repo", "/repo/.worktrees")
	// Each entry must have a tab separating branch from description.
	for _, line := range got {
		if !strings.Contains(line, "\t") {
			t.Fatalf("expected branch\\tdesc format; got %q", line)
		}
	}
	// `feat-x` lives inside the trees dir, so its description starts with `#`.
	for _, line := range got {
		if strings.HasPrefix(line, "feat-x\t") {
			_, desc, _ := strings.Cut(line, "\t")
			if !strings.HasPrefix(desc, "#feat-x") {
				t.Errorf("expected feat-x description to start with `#feat-x`; got %q", desc)
			}
		}
	}
}

func TestWorktreeBranchesForRm(t *testing.T) {
	wts := []wt.Worktree{
		{Branch: "main", Path: "/repo"},
		{Branch: "paul/auth", Path: "/repo/.worktrees/paul/auth"},
		{Branch: "feat-x", Path: "/repo/.worktrees/feat-x"},
		{Branch: "feat-y", Path: "/repo/.worktrees/feat-y"},
	}
	mainRoot := "/repo"
	treesDir := "/repo/.worktrees"

	t.Run("excludes main worktree", func(t *testing.T) {
		got := branchesOnly(worktreeBranchesForRm(wts, mainRoot, treesDir, nil))
		want := []string{"feat-x", "feat-y", "paul/auth"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("excludes already-chosen args", func(t *testing.T) {
		got := branchesOnly(worktreeBranchesForRm(wts, mainRoot, treesDir, []string{"feat-x"}))
		want := []string{"feat-y", "paul/auth"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("multiple already-chosen", func(t *testing.T) {
		got := branchesOnly(worktreeBranchesForRm(wts, mainRoot, treesDir, []string{"feat-x", "paul/auth"}))
		want := []string{"feat-y"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("entries carry descriptions", func(t *testing.T) {
		got := worktreeBranchesForRm(wts, mainRoot, treesDir, nil)
		for _, line := range got {
			if !strings.Contains(line, "\t") {
				t.Errorf("expected tab-separated description; got %q", line)
			}
		}
	})
}

func TestAddCandidates(t *testing.T) {
	locals := []string{"main", "feat-x", "paul/auth"}
	remotes := []string{"origin/main", "origin/feat-y", "origin/paul/auth"}

	t.Run("excludes branches already in a worktree", func(t *testing.T) {
		wts := []wt.Worktree{
			{Branch: "main", Path: "/repo"},
			{Branch: "paul/auth", Path: "/repo/.worktrees/paul/auth"},
		}
		got := addCandidates(locals, remotes, wts)
		// `paul/auth` (local) excluded.
		// `origin/main` excluded because local `main` is in a worktree.
		// `origin/paul/auth` excluded because local `paul/auth` is in a
		// worktree (origin/<rest> would resolve to local <rest>).
		want := []string{"feat-x", "origin/feat-y"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("no worktrees: every ref offered", func(t *testing.T) {
		got := addCandidates(locals, remotes, nil)
		want := []string{
			"feat-x", "main",
			"origin/feat-y", "origin/main", "origin/paul/auth",
			"paul/auth",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("output sorted", func(t *testing.T) {
		got := addCandidates([]string{"zz", "aa", "mm"}, nil, nil)
		want := []string{"aa", "mm", "zz"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestSplitNonEmptyLines(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"foo", []string{"foo"}},
		{"foo\nbar", []string{"foo", "bar"}},
		{"foo\n\nbar\n", []string{"foo", "bar"}},
		{"  foo  \n bar ", []string{"foo", "bar"}},
	}
	for _, c := range cases {
		got := splitNonEmptyLines(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitNonEmptyLines(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
