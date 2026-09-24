package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLock_ListShowsAgeAndReason(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "held", "free")

	res := runWT(t, repo, "lock", "held", "--reason", "agent x (pid 1)")
	if res.ExitCode != 0 {
		t.Fatalf("lock exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stderr, "locked held") {
		t.Errorf("expected confirmation, got: %s", res.Stderr)
	}
	if porcelain := mustGit(t, repo, "worktree", "list", "--porcelain"); !strings.Contains(porcelain, "locked agent x (pid 1)") {
		t.Fatalf("git doesn't see the lock:\n%s", porcelain)
	}

	// Backdate the lock so the age column has something to show.
	lockFile := filepath.Join(repo, ".git", "worktrees", filepath.Base(paths[0]), "locked")
	past := time.Now().Add(-50 * time.Hour)
	if err := os.Chtimes(lockFile, past, past); err != nil {
		t.Fatal(err)
	}

	list := runWT(t, repo, "--plain", "list")
	for _, line := range strings.Split(list.Stdout, "\n") {
		switch {
		case strings.Contains(line, "held"):
			if !strings.Contains(line, "locked  2d   2h  agent x (pid 1)") {
				t.Errorf("locked row missing age + reason: %q", line)
			}
		case strings.Contains(line, "locked"):
			t.Errorf("only the locked worktree should be tagged: %q", line)
		}
	}
}

func TestLock_AlreadyLockedKeepsTheExistingLock(t *testing.T) {
	repo := newRepo(t)
	mkTrees(t, repo, "held")
	runWT(t, repo, "lock", "held", "--reason", "first")

	res := runWT(t, repo, "lock", "held", "--reason", "second")
	if res.ExitCode != 0 {
		t.Fatalf("relock exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stderr, "held is locked") || !strings.Contains(res.Stderr, "first") {
		t.Errorf("expected a note naming the existing lock, got: %s", res.Stderr)
	}
	if porcelain := mustGit(t, repo, "worktree", "list", "--porcelain"); !strings.Contains(porcelain, "locked first") {
		t.Errorf("existing reason should survive:\n%s", porcelain)
	}
}

func TestLock_MainWorktreeRefused(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "lock", "main")
	if res.ExitCode == 0 || !strings.Contains(res.Stderr, "cannot lock the main worktree") {
		t.Errorf("expected refusal, exit %d: %s", res.ExitCode, res.Stderr)
	}
}

func TestLock_NoArgsNonInteractive(t *testing.T) {
	repo := newRepo(t)
	mkTrees(t, repo, "free")
	res := runWT(t, repo, "lock", "--non-interactive")
	if res.ExitCode == 0 || !strings.Contains(res.Stderr, "no branches specified") {
		t.Errorf("expected a request for branch args, exit %d: %s", res.ExitCode, res.Stderr)
	}
}

func TestUnlock_ReportsTheLockItReleased(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "held")
	mustGit(t, repo, "worktree", "lock", "--reason", "claude agent a1 (pid 70277)", paths[0])

	res := runWT(t, repo, "unlock", "held")
	if res.ExitCode != 0 {
		t.Fatalf("unlock exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stderr, "unlocked held (was locked") || !strings.Contains(res.Stderr, "claude agent a1 (pid 70277)") {
		t.Errorf("expected the released lock's reason, got: %s", res.Stderr)
	}
	if porcelain := mustGit(t, repo, "worktree", "list", "--porcelain"); strings.Contains(porcelain, "locked") {
		t.Errorf("still locked:\n%s", porcelain)
	}

	again := runWT(t, repo, "unlock", "held")
	if again.ExitCode != 0 || !strings.Contains(again.Stderr, "held is not locked") {
		t.Errorf("unlocking twice should be a no-op note, exit %d: %s", again.ExitCode, again.Stderr)
	}
}

func TestUnlock_NothingLocked(t *testing.T) {
	repo := newRepo(t)
	mkTrees(t, repo, "free")
	res := runWT(t, repo, "unlock", "--non-interactive")
	if res.ExitCode != 0 || !strings.Contains(res.Stderr, "no locked worktrees") {
		t.Errorf("exit %d: %s", res.ExitCode, res.Stderr)
	}
}

// A locked target must stop rm before anything goes, even when an unlocked
// target is listed ahead of it — git would otherwise refuse the locked one
// only after the first was deleted.
func TestRm_LockedTargetRefusedBeforeDeletingAnything(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "rl-free", "rl-held")
	mustGit(t, repo, "worktree", "lock", "--reason", "agent z", paths[1])

	res := runWT(t, repo, "rm", "rl-free", "rl-held", "--force", "--non-interactive")
	if res.ExitCode == 0 {
		t.Fatalf("expected refusal, got success: %s", res.Stderr)
	}
	for _, p := range paths {
		mustExist(t, p)
	}
	for _, want := range []string{"rl-held", "agent z", "unlock"} {
		if !strings.Contains(res.Stderr, want) {
			t.Errorf("stderr missing %q\n--- got ---\n%s", want, res.Stderr)
		}
	}
}

func TestClean_SkipsLockedWorktrees(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "cl-held", "cl-free")
	mustGit(t, repo, "worktree", "lock", "--reason", "keep me", paths[0])
	for _, b := range []string{"cl-held", "cl-free"} {
		if err := orphanBranch(repo, b); err != nil {
			t.Fatal(err)
		}
	}

	res := runWT(t, repo, "clean", "--non-interactive", "--orphaned-only", "--no-fetch")
	if res.ExitCode != 0 {
		t.Fatalf("clean exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustExist(t, paths[0])
	mustNotExist(t, paths[1])
	if !strings.Contains(res.Stderr, "skipping cl-held: locked") || !strings.Contains(res.Stderr, "keep me") {
		t.Errorf("expected the locked worktree reported as skipped, got: %s", res.Stderr)
	}
}
