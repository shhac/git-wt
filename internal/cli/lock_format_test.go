package cli

import (
	"strings"
	"testing"
	"time"

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

func TestLockLabel(t *testing.T) {
	cases := []struct {
		in   wt.Worktree
		want string
	}{
		{wt.Worktree{Locked: true}, "locked"},
		{wt.Worktree{Locked: true, LockedAt: time.Now().Add(-50 * time.Hour)}, "locked 2d 2h"},
	}
	for _, c := range cases {
		if got := lockLabel(c.in); got != c.want {
			t.Errorf("lockLabel = %q, want %q", got, c.want)
		}
	}
}
