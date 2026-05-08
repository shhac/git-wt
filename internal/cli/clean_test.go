package cli

import (
	"errors"
	"reflect"
	"strings"
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

func TestPrintCleanTargets(t *testing.T) {
	var buf strings.Builder
	targets := []taggedTarget{
		{wt: wt.Worktree{Path: "/p/a", Branch: "a"}, reason: "branch deleted"},
		{wt: wt.Worktree{Path: "/p/b", Branch: "b"}, reason: "upstream gone"},
	}
	printCleanTargets(&buf, targets)
	got := buf.String()
	for _, want := range []string{"worktrees to remove:", "[branch deleted]", "[upstream gone]", "/p/a", "/p/b"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n--- got ---\n%s", want, got)
		}
	}
}

// collectCleanTargets composes findOrphanedWorktrees + goneBranches +
// findUpstreamGoneWorktrees, with dedup to prevent counting the same path
// twice. The orphaned scan calls into git via wt.BranchExists, and the
// upstream-gone scan calls git for-each-ref. We can't unit-test the full
// function without a git seam, but we can test the dedup-and-merge slice of
// it by directly exercising the building blocks the same way the function
// does — which is what these helpers (findOrphaned, findUpstreamGone,
// pathSet) already do — and asserting the dedup semantics that the
// integration of the two checks is supposed to enforce.

func TestCollectCleanTargets_DedupSimulation(t *testing.T) {
	// A worktree on a branch that is BOTH locally-deleted and has gone
	// upstream tracking should appear only once in the merged target list.
	wts := []wt.Worktree{
		{Path: "/p/main", Branch: "main"},
		{Path: "/p/dual", Branch: "dual"}, // both orphaned + upstream-gone
		{Path: "/p/orph", Branch: "orph"}, // only orphaned
		{Path: "/p/gone", Branch: "gone"}, // only upstream-gone
	}
	repo := &wt.RepoInfo{MainRoot: "/p/main"}

	// "alive" branches (everything else is orphaned)
	branchExists := func(b string) (bool, error) { return b == "gone", nil }

	orph, err := findOrphanedWorktrees(wts, repo, branchExists)
	if err != nil {
		t.Fatal(err)
	}
	gone := findUpstreamGoneWorktrees(wts, repo, goneSetFromList([]string{"dual", "gone"}))

	// Replicate collectCleanTargets' merge-with-dedup logic.
	targets := append([]taggedTarget(nil), orph...)
	seen := pathSet(targets)
	for _, t := range gone {
		if _, dup := seen[t.wt.Path]; !dup {
			targets = append(targets, t)
		}
	}

	// Expect three distinct paths: /p/dual (orphaned wins, no dup),
	// /p/orph (orphaned), /p/gone (upstream-gone).
	wantPaths := map[string]string{
		"/p/dual": "branch deleted", // orphaned check ran first
		"/p/orph": "branch deleted",
		"/p/gone": "upstream gone",
	}
	if len(targets) != len(wantPaths) {
		t.Fatalf("len = %d, want %d (got: %+v)", len(targets), len(wantPaths), targets)
	}
	for _, tg := range targets {
		want, ok := wantPaths[tg.wt.Path]
		if !ok {
			t.Errorf("unexpected path: %s", tg.wt.Path)
			continue
		}
		if tg.reason != want {
			t.Errorf("reason for %s: got %q, want %q (the orphaned check should win)", tg.wt.Path, tg.reason, want)
		}
	}
}

