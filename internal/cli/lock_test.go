package cli

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/wt"
)

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
	unlockable := branchesOnly(worktreeBranchesForLock(wts, "/repo", "/repo/.worktrees", []string{"held-too"}, false))
	if want := []string{"held"}; !reflect.DeepEqual(unlockable, want) {
		t.Errorf("unlock candidates = %v, want %v", unlockable, want)
	}
	lockable := branchesOnly(worktreeBranchesForLock(wts, "/repo", "/repo/.worktrees", nil, true))
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

func TestApplyLockChange(t *testing.T) {
	held := wt.Worktree{Branch: "held", Path: "/r/held", Locked: true, LockReason: "agent a"}
	free := wt.Worktree{Branch: "free", Path: "/r/free"}
	boom := wt.Worktree{Branch: "boom", Path: "/r/boom"}
	after := wt.Worktree{Branch: "after", Path: "/r/after"}

	t.Run("lock skips the already-locked and reports the rest", func(t *testing.T) {
		var changed []string
		var buf strings.Builder
		err := applyLockChange(&buf, []wt.Worktree{held, free}, true, func(w wt.Worktree) error {
			changed = append(changed, w.Branch)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"free"}; !reflect.DeepEqual(changed, want) {
			t.Errorf("changed = %v, want %v", changed, want)
		}
		if want := "held is locked: agent a\nlocked free\n"; buf.String() != want {
			t.Errorf("report = %q, want %q", buf.String(), want)
		}
	})

	t.Run("unlock names the lock it released", func(t *testing.T) {
		var buf strings.Builder
		err := applyLockChange(&buf, []wt.Worktree{held, free}, false, func(wt.Worktree) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		if want := "unlocked held (was locked: agent a)\nfree is not locked\n"; buf.String() != want {
			t.Errorf("report = %q, want %q", buf.String(), want)
		}
	})

	t.Run("stops at the first failure, naming it", func(t *testing.T) {
		var changed []string
		err := applyLockChange(io.Discard, []wt.Worktree{free, boom, after}, true, func(w wt.Worktree) error {
			changed = append(changed, w.Branch)
			if w.Branch == "boom" {
				return errors.New("git said no")
			}
			return nil
		})
		if err == nil || err.Error() != "lock boom: git said no" {
			t.Errorf("err = %v, want `lock boom: git said no`", err)
		}
		if want := []string{"free", "boom"}; !reflect.DeepEqual(changed, want) {
			t.Errorf("changed = %v, want %v (nothing after the failure)", changed, want)
		}
	})
}
