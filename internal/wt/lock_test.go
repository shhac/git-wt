package wt

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParsePorcelain_LockReason(t *testing.T) {
	in := "worktree /p/bare\nHEAD 1\nbranch refs/heads/a\nlocked\n\n" +
		"worktree /p/why\nHEAD 2\nbranch refs/heads/b\nlocked claude agent x (pid 1)\n\n" +
		"worktree /p/quoted\nHEAD 3\nbranch refs/heads/c\nlocked \"multi\\nline\"\n\n" +
		"worktree /p/octal\nHEAD 5\nbranch refs/heads/e\nlocked \"a \\342\\200\\224 b\"\n\n" +
		"worktree /p/open\nHEAD 4\nbranch refs/heads/d\n\n"

	got := parsePorcelain(in)
	want := []struct {
		locked bool
		reason string
	}{
		{true, ""},
		{true, "claude agent x (pid 1)"},
		{true, "multi\nline"},
		{true, "a \u2014 b"},
		{false, ""},
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Locked != w.locked || got[i].LockReason != w.reason {
			t.Errorf("%s: Locked=%v LockReason=%q, want %v %q", got[i].Path, got[i].Locked, got[i].LockReason, w.locked, w.reason)
		}
	}
}

func TestLockTimes(t *testing.T) {
	root := t.TempDir()
	commonDir := filepath.Join(root, "repo", ".git")
	absTree := filepath.Join(root, "abs-tree")
	relTree := filepath.Join(root, "repo", ".worktrees", "rel-tree")

	lockedAt := time.Now().Add(-50 * time.Hour).Truncate(time.Second)
	writeAdmin(t, commonDir, "abs", filepath.Join(absTree, ".git"), &lockedAt)
	writeAdmin(t, commonDir, "rel", "../../../.worktrees/rel-tree/.git", &lockedAt)
	writeAdmin(t, commonDir, "open", filepath.Join(root, "open-tree", ".git"), nil)

	got := lockTimes(commonDir)
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2: %v", len(got), got)
	}
	for _, p := range []string{absTree, relTree} {
		at, ok := got[canonicalPath(p)]
		if !ok {
			t.Errorf("missing entry for %s in %v", p, got)
			continue
		}
		if !at.Equal(lockedAt) {
			t.Errorf("%s: LockedAt = %v, want %v", p, at, lockedAt)
		}
	}
}

// writeAdmin fakes the admin dir git keeps for a linked worktree. A nil
// lockedAt leaves the worktree unlocked.
func writeAdmin(t *testing.T, commonDir, id, gitdir string, lockedAt *time.Time) {
	t.Helper()
	admin := filepath.Join(commonDir, "worktrees", id)
	if err := os.MkdirAll(admin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(admin, "gitdir"), []byte(gitdir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if lockedAt == nil {
		return
	}
	lockFile := filepath.Join(admin, "locked")
	if err := os.WriteFile(lockFile, []byte("reason"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(lockFile, *lockedAt, *lockedAt); err != nil {
		t.Fatal(err)
	}
}
