package cli

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/wt"
)

func TestLockTag(t *testing.T) {
	cases := []struct {
		name string
		in   wt.Worktree
		want string
	}{
		{"unlocked", wt.Worktree{}, ""},
		{"lock time unknown", wt.Worktree{Locked: true}, "locked"},
		{"lock time known", wt.Worktree{Locked: true, LockedAt: time.Now().Add(-50 * time.Hour)}, "locked  2d   2h"},
	}
	for _, c := range cases {
		if got := lockTag(c.in); got != c.want {
			t.Errorf("%s: lockTag = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestWithLockTag_ReasonIsOneLineAndTruncated(t *testing.T) {
	long := "line one\nline\ttwo \x1b[31m" + strings.Repeat("x", 60)
	got := withLockTag("row", wt.Worktree{Locked: true, LockReason: long}, noStyle, noStyle)

	if strings.ContainsAny(got, "\n\t\x1b") {
		t.Errorf("row carries control characters: %q", got)
	}
	if !strings.HasPrefix(got, "row  locked  line one line two [31m") {
		t.Errorf("unexpected row: %q", got)
	}
	reason := strings.TrimPrefix(got, "row  locked  ")
	if n := len([]rune(reason)); n != lockReasonWidth || !strings.HasSuffix(reason, "…") {
		t.Errorf("reason = %q (%d runes), want %d runes ending in …", reason, n, lockReasonWidth)
	}
}

func TestWithLockTag_Unlocked(t *testing.T) {
	if got := withLockTag("row", wt.Worktree{}, noStyle, noStyle); got != "row" {
		t.Errorf("got %q, want the row unchanged", got)
	}
}

func TestLockStateDetail(t *testing.T) {
	at := time.Now().Add(-26 * time.Hour)
	cases := []struct {
		name string
		in   wt.Worktree
		want string
	}{
		{"unlocked", wt.Worktree{}, "not locked"},
		{"bare lock", wt.Worktree{Locked: true}, "locked"},
		{"reason only", wt.Worktree{Locked: true, LockReason: "agent x"}, "locked: agent x"},
		{"age and reason", wt.Worktree{Locked: true, LockedAt: at, LockReason: "agent\nx"}, "locked 1d 2h ago: agent x"},
	}
	for _, c := range cases {
		if got := lockStateDetail(c.in); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPrintList_ShowsLock(t *testing.T) {
	ui.Plain = true
	defer func() { ui.Plain = false }()

	wts := []wt.Worktree{
		{Path: "/repo/.worktrees/a", Branch: "a", Locked: true, LockReason: "agent x"},
		{Path: "/repo/.worktrees/b", Branch: "b"},
	}
	var buf strings.Builder
	printList(&buf, wts, &wts[0], "/repo", "/repo/.worktrees")
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

	if !strings.HasSuffix(lines[0], "  locked  agent x") {
		t.Errorf("locked current row should end with the lock tag; got %q", lines[0])
	}
	if strings.Contains(lines[1], "locked") {
		t.Errorf("unlocked row should carry no tag; got %q", lines[1])
	}
}

func TestWorktreeBranchesForLock(t *testing.T) {
	wts := []wt.Worktree{
		{Branch: "main", Path: "/repo"},
		{Branch: "held", Path: "/repo/.worktrees/held", Locked: true},
		{Branch: "held-too", Path: "/repo/.worktrees/held-too", Locked: true},
		{Branch: "free", Path: "/repo/.worktrees/free"},
	}
	unlockable := branchesOnly(worktreeBranchesForLock(wts, "/repo", "/repo/.worktrees", []string{"held-too"}, true))
	if want := []string{"held"}; !reflect.DeepEqual(unlockable, want) {
		t.Errorf("unlock candidates = %v, want %v", unlockable, want)
	}
	lockable := branchesOnly(worktreeBranchesForLock(wts, "/repo", "/repo/.worktrees", nil, false))
	if want := []string{"free"}; !reflect.DeepEqual(lockable, want) {
		t.Errorf("lock candidates = %v, want %v", lockable, want)
	}
}

func TestResolveNamedWorktrees(t *testing.T) {
	repo := &wt.RepoInfo{MainRoot: "/repo"}
	treesDir := "/repo/.worktrees"
	wts := []wt.Worktree{
		{Branch: "main", Path: "/repo"},
		{Branch: "paul/auth", Path: "/repo/.worktrees/paul/auth"},
		{Detached: true, Path: "/repo/.worktrees/scratch"},
	}

	got, err := resolveNamedWorktrees(wts, repo, []string{"auth", "paul/auth", "scratch"}, treesDir, "lock")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, w := range got {
		paths = append(paths, w.Path)
	}
	if want := []string{"/repo/.worktrees/paul/auth", "/repo/.worktrees/scratch"}; !reflect.DeepEqual(paths, want) {
		t.Errorf("paths = %v, want %v (suffix + exact deduped, detached by leaf)", paths, want)
	}

	if _, err := resolveNamedWorktrees(wts, repo, []string{"main"}, treesDir, "lock"); err == nil || !strings.Contains(err.Error(), "cannot lock the main worktree") {
		t.Errorf("main: err = %v, want refusal", err)
	}
	if _, err := resolveNamedWorktrees(wts, repo, []string{"nope"}, treesDir, "unlock"); err == nil || !strings.Contains(err.Error(), `"nope"`) {
		t.Errorf("unknown: err = %v, want no-worktree error", err)
	}
}

func TestRefuseLocked(t *testing.T) {
	targets := []rmTarget{
		{Worktree: wt.Worktree{Branch: "free", Path: "/r/free"}},
		{Worktree: wt.Worktree{Branch: "a", Path: "/r/a", Locked: true, LockReason: "agent a"}},
		{Worktree: wt.Worktree{Branch: "b", Path: "/r/b", Locked: true}},
	}
	err := refuseLocked(targets)
	if err == nil {
		t.Fatal("expected an error naming the locked targets")
	}
	msg := err.Error()
	for _, want := range []string{"2 worktree(s) locked", "a (locked: agent a)", "b (locked)", "nothing was removed"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "free") {
		t.Errorf("error names an unlocked target:\n%s", msg)
	}

	if err := refuseLocked(targets[:1]); err != nil {
		t.Errorf("no locked targets: err = %v, want nil", err)
	}
}

func TestSkipLocked(t *testing.T) {
	targets := []rmTarget{
		{Worktree: wt.Worktree{Branch: "a", Path: "/r/a", Locked: true, LockReason: "agent a"}},
		{Worktree: wt.Worktree{Branch: "b", Path: "/r/b"}},
	}
	var buf strings.Builder
	kept := skipLocked(&buf, targets)
	if len(kept) != 1 || kept[0].Branch != "b" {
		t.Errorf("kept = %+v, want only b", kept)
	}
	if got := buf.String(); got != "skipping a: locked: agent a\n" {
		t.Errorf("report = %q", got)
	}
}
