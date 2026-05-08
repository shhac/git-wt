package cli

import (
	"errors"
	"reflect"
	"testing"

	"github.com/shhac/git-wt/internal/wt"
)

func TestParseGoneBranches(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			"plain branch with no upstream",
			"main\t\n",
			nil,
		},
		{
			"upstream-tracking, behind, ahead",
			"main\t[ahead 1]\nfeat\t[behind 2]\n",
			nil,
		},
		{
			"single gone",
			"feat\t[gone]\n",
			[]string{"feat"},
		},
		{
			"mixed: gone + alive",
			"main\t\nold\t[gone]\nfeat\t[ahead 1]\nstale\t[gone]\n",
			[]string{"old", "stale"},
		},
		{
			"gone embedded in compound track",
			"feat\t[behind 1, gone]\n",
			[]string{"feat"},
		},
		{
			"empty input",
			"",
			nil,
		},
		{
			"blank lines tolerated",
			"\n\nfeat\t[gone]\n\n",
			[]string{"feat"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseGoneBranches(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("parseGoneBranches(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestFindOrphanedWorktrees_Classification(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/p/main", Branch: "main"},                  // skipped: main repo
		{Path: "/p/prune", Branch: "p", Prunable: true},    // prunable
		{Path: "/p/det", Branch: "", Detached: true},       // detached (no branch)
		{Path: "/p/alive", Branch: "alive"},                // branch exists → kept
		{Path: "/p/dead", Branch: "dead"},                  // branch gone → orphan
	}
	repo := &wt.RepoInfo{MainRoot: "/p/main"}
	branchExists := func(b string) (bool, error) {
		return b == "alive", nil
	}

	got, err := findOrphanedWorktrees(wts, repo, branchExists)
	if err != nil {
		t.Fatal(err)
	}
	wantPaths := map[string]string{
		"/p/prune": "prunable",
		"/p/det":   "detached",
		"/p/dead":  "branch deleted",
	}
	if len(got) != len(wantPaths) {
		t.Fatalf("len = %d, want %d (got: %+v)", len(got), len(wantPaths), got)
	}
	for _, tg := range got {
		want, ok := wantPaths[tg.wt.Path]
		if !ok {
			t.Errorf("unexpected path in result: %s", tg.wt.Path)
			continue
		}
		if tg.reason != want {
			t.Errorf("reason for %s = %q, want %q", tg.wt.Path, tg.reason, want)
		}
	}
}

func TestFindOrphanedWorktrees_BranchExistsErrorPropagates(t *testing.T) {
	wts := []wt.Worktree{{Path: "/p/feat", Branch: "feat"}}
	repo := &wt.RepoInfo{MainRoot: "/p/main"}
	wantErr := errors.New("git boom")
	branchExists := func(b string) (bool, error) { return false, wantErr }

	_, err := findOrphanedWorktrees(wts, repo, branchExists)
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wraps %v", err, wantErr)
	}
}

func TestFindUpstreamGoneWorktrees(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/p/main", Branch: "main"},   // skipped: main repo
		{Path: "/p/det", Branch: ""},        // skipped: no branch
		{Path: "/p/alive", Branch: "alive"}, // not in gone set
		{Path: "/p/dead", Branch: "dead"},   // in gone set
	}
	repo := &wt.RepoInfo{MainRoot: "/p/main"}
	goneSet := goneSetFromList([]string{"dead", "ghost"})

	got := findUpstreamGoneWorktrees(wts, repo, goneSet)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (got: %+v)", len(got), got)
	}
	if got[0].wt.Path != "/p/dead" || got[0].reason != "upstream gone" {
		t.Errorf("got %+v, want /p/dead with reason 'upstream gone'", got[0])
	}
}

func TestFindUpstreamGoneWorktrees_EmptyGoneSet(t *testing.T) {
	wts := []wt.Worktree{{Path: "/p/feat", Branch: "feat"}}
	repo := &wt.RepoInfo{MainRoot: "/p/main"}
	got := findUpstreamGoneWorktrees(wts, repo, map[string]struct{}{})
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestPathSet_Dedup(t *testing.T) {
	in := []taggedTarget{
		{wt: wt.Worktree{Path: "/a"}},
		{wt: wt.Worktree{Path: "/b"}},
		{wt: wt.Worktree{Path: "/a"}}, // duplicate path
	}
	got := pathSet(in)
	if len(got) != 2 {
		t.Errorf("len = %d, want 2 (paths: %v)", len(got), got)
	}
	if _, ok := got["/a"]; !ok {
		t.Errorf("expected /a in set")
	}
	if _, ok := got["/b"]; !ok {
		t.Errorf("expected /b in set")
	}
}
