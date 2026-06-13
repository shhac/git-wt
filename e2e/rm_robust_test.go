package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gitignoreDir commits a .gitignore covering dir/ in the given worktree.
func gitignoreDir(t *testing.T, repo, dir string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, ".gitignore"), dir+"/\n")
	mustGit(t, repo, "add", ".gitignore")
	mustGit(t, repo, "commit", "-q", "-m", "ignore "+dir)
}

// TestRm_ReadOnlyDirInIgnoredPath pins the headline fix: `git worktree
// remove` dies mid-delete on a read-only directory (build caches and
// package managers create these), unregistering the worktree but stranding
// a half-deleted directory. gwt rm must remove it completely.
func TestRm_ReadOnlyDirInIgnoredPath(t *testing.T) {
	repo := newRepo(t)
	gitignoreDir(t, repo, "junk")
	if r := runWT(t, repo, "new", "rm-ro", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "rm-ro")
	roDir := filepath.Join(wtPath, "junk", "rodir")
	if err := os.MkdirAll(roDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(roDir, "locked.txt"), "x\n")
	if err := os.Chmod(filepath.Join(roDir, "locked.txt"), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(roDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(roDir, 0o755) }) // in case removal fails

	res := runWT(t, repo, "rm", "rm-ro", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("rm exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, wtPath)
	if entries, err := os.ReadDir(filepath.Join(repo, ".worktrees")); err == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "rm-ro") {
				t.Errorf("leftover entry in trees dir: %s", e.Name())
			}
		}
	}
	if list := mustGit(t, repo, "worktree", "list"); strings.Contains(list, "rm-ro") {
		t.Errorf("worktree still registered:\n%s", list)
	}
}

// TestRm_IgnoredFilesDoNotBlock matches git's own semantics: gitignored
// files never require --force.
func TestRm_IgnoredFilesDoNotBlock(t *testing.T) {
	repo := newRepo(t)
	gitignoreDir(t, repo, "junk")
	if r := runWT(t, repo, "new", "rm-ign", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "rm-ign")
	if err := os.MkdirAll(filepath.Join(wtPath, "junk"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(wtPath, "junk", "cache.bin"), "data\n")

	res := runWT(t, repo, "rm", "rm-ign", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("rm exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, wtPath)
}

// TestRm_RefusesDirtyWithoutForce preserves git's safety check on the fast
// path: untracked (non-ignored) files must block removal, leaving both the
// directory and the registration untouched.
func TestRm_RefusesDirtyWithoutForce(t *testing.T) {
	repo := newRepo(t)
	if r := runWT(t, repo, "new", "rm-dirty", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "rm-dirty")
	mustWrite(t, filepath.Join(wtPath, "untracked.txt"), "wip\n")

	res := runWT(t, repo, "rm", "rm-dirty", "--non-interactive")
	if res.ExitCode == 0 {
		t.Fatalf("expected refusal for dirty worktree")
	}
	if !strings.Contains(res.Stderr, "--force") {
		t.Errorf("error should point at --force, got: %s", res.Stderr)
	}
	mustExist(t, wtPath)
	mustExist(t, filepath.Join(wtPath, "untracked.txt"))
	if list := mustGit(t, repo, "worktree", "list"); !strings.Contains(list, "rm-dirty") {
		t.Errorf("worktree must stay registered after refusal:\n%s", list)
	}
}

func TestRm_ForceRemovesDirty(t *testing.T) {
	repo := newRepo(t)
	if r := runWT(t, repo, "new", "rm-forced", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "rm-forced")
	mustWrite(t, filepath.Join(wtPath, "untracked.txt"), "wip\n")

	res := runWT(t, repo, "rm", "rm-forced", "--non-interactive", "--force")
	if res.ExitCode != 0 {
		t.Fatalf("rm --force exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, wtPath)
}

// TestRm_RescuesOrphanDir covers the leftover-directory rescue: a dir under
// .worktrees/ that no longer has a worktree registration (the post-failure
// state older versions could strand). Non-interactive rm requires --force.
func TestRm_RescuesOrphanDir(t *testing.T) {
	repo := newRepo(t)
	ghost := filepath.Join(repo, ".worktrees", "ghost")
	if err := os.MkdirAll(filepath.Join(ghost, "junk"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(ghost, "junk", "f.txt"), "x\n")

	res := runWT(t, repo, "rm", "ghost", "--non-interactive")
	if res.ExitCode == 0 {
		t.Fatalf("expected refusal without --force, stderr: %s", res.Stderr)
	}
	if !strings.Contains(res.Stderr, "leftover") {
		t.Errorf("error should explain the leftover state, got: %s", res.Stderr)
	}
	mustExist(t, ghost)

	res = runWT(t, repo, "rm", "ghost", "--non-interactive", "--force")
	if res.ExitCode != 0 {
		t.Fatalf("rm --force exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, ghost)
}
