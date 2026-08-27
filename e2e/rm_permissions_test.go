package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// requireUnprivileged skips a test that relies on filesystem permissions
// actually denying something. root bypasses them, so the test would pass
// vacuously or, worse, fail because the operation succeeded.
func requireUnprivileged(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits are not enforced")
	}
}

// chmodFor restores the mode at test end so t.TempDir() cleanup can run.
func chmodFor(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	prev, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, prev.Mode()) })
}

// registered reports whether git still lists path as a worktree. Used where
// the failure happens early enough that the worktree should be left whole:
// present on disk and still in git's records.
func registered(t *testing.T, repo, path string) bool {
	t.Helper()
	return strings.Contains(mustGit(t, repo, "worktree", "list", "--porcelain"), path)
}

// The fast path renames the worktree aside before unregistering it, and
// that rename needs write permission on the parent. Denied it, gwt falls
// back to `git worktree remove` — which on this filesystem state
// unregisters the worktree and *then* fails to delete the directory,
// leaving an unregistered leftover. That is git's own behaviour, not
// something gwt chooses; what matters is that nothing is silently
// destroyed and the leftover stays recoverable.
func TestRm_ReadOnlyTreesDirFailsAndLeavesTheDirectoryIntact(t *testing.T) {
	requireUnprivileged(t)
	repo := newRepo(t)
	paths := mkTrees(t, repo, "ro-parent")
	treesDir := filepath.Join(repo, ".worktrees")
	chmodFor(t, treesDir, 0o555)

	res := runWT(t, repo, "rm", "ro-parent", "--non-interactive")
	if res.ExitCode == 0 {
		t.Fatal("expected the removal to fail on a read-only parent")
	}
	mustExist(t, paths[0])
	if !strings.Contains(res.Stderr, "Permission denied") {
		t.Errorf("the permission failure should reach the user\n--- got ---\n%s", res.Stderr)
	}
	if strings.Contains(res.Stderr, ".removing-") {
		t.Errorf("a renamed-aside directory was left behind\n--- got ---\n%s", res.Stderr)
	}
}

// The recovery half of the case above: once the parent is writable again,
// the leftover is exactly what rm's rescue path is for, so a second run
// finishes the job by leaf name.
func TestRm_LeftoverFromAPermissionFailureIsRescuable(t *testing.T) {
	requireUnprivileged(t)
	repo := newRepo(t)
	paths := mkTrees(t, repo, "ro-rescue")
	treesDir := filepath.Join(repo, ".worktrees")

	if err := os.Chmod(treesDir, 0o555); err != nil {
		t.Fatal(err)
	}
	if res := runWT(t, repo, "rm", "ro-rescue", "--non-interactive"); res.ExitCode == 0 {
		t.Fatal("expected the first removal to fail")
	}
	if err := os.Chmod(treesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	res := runWT(t, repo, "rm", "ro-rescue", "--non-interactive", "--force")
	if res.ExitCode != 0 {
		t.Fatalf("rescue run failed: exit %d\n%s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, paths[0])
	if !strings.Contains(res.Stderr, "unregistered leftover") {
		t.Errorf("expected the rescue path to be used\n--- got ---\n%s", res.Stderr)
	}
}

// An unsearchable worktree root defeats the dirty scan and git's own
// validation. It must fail whole, like the read-only parent.
func TestRm_UnsearchableWorktreeRootFailsWhole(t *testing.T) {
	requireUnprivileged(t)
	repo := newRepo(t)
	paths := mkTrees(t, repo, "no-x")
	chmodFor(t, paths[0], 0o000)

	res := runWT(t, repo, "rm", "no-x", "--non-interactive")
	if res.ExitCode == 0 {
		t.Fatal("expected the removal to fail on an unsearchable worktree")
	}
	mustExist(t, paths[0])
	if !registered(t, repo, paths[0]) {
		t.Error("worktree was unregistered despite the failure: half-removed")
	}
}

// The counterpart that must still succeed: unreadable files *inside* the
// worktree are what wt.DeleteTree exists to fix, so removal goes through.
func TestRm_UnreadableFileInsideWorktreeIsStillRemoved(t *testing.T) {
	requireUnprivileged(t)
	repo := newRepo(t)
	paths := mkTrees(t, repo, "inner-perm")
	sub := filepath.Join(paths[0], "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(sub, "locked.txt"), "x\n")
	if err := os.Chmod(filepath.Join(sub, "locked.txt"), 0o000); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sub, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })

	res := runWT(t, repo, "rm", "inner-perm", "--non-interactive", "--force")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, paths[0])
}
