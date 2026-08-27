package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkTrees creates one worktree per name and returns their paths, in order.
func mkTrees(t *testing.T, repo string, names ...string) []string {
	t.Helper()
	paths := make([]string, len(names))
	for i, n := range names {
		if r := runWT(t, repo, "new", n, "--non-interactive", "--no-copy"); r.ExitCode != 0 {
			t.Fatalf("setup %s: %s", n, r.Stderr)
		}
		paths[i] = filepath.Join(repo, ".worktrees", n)
	}
	return paths
}

// TestRm_DirtyMiddleTargetBailsBeforeDeletingAnything is the headline of the
// preflight: the dirty worktree sits second in the list, and the first one
// must survive. Before the preflight, `a` was already gone by the time `b`
// was inspected.
func TestRm_DirtyMiddleTargetBailsBeforeDeletingAnything(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "dm-a", "dm-b", "dm-c")
	mustWrite(t, filepath.Join(paths[1], "scratch.txt"), "work in progress\n")

	res := runWT(t, repo, "rm", "dm-a", "dm-b", "dm-c", "--non-interactive")
	if res.ExitCode == 0 {
		t.Fatalf("expected refusal, got success: %s", res.Stderr)
	}
	for _, p := range paths {
		mustExist(t, p)
	}
	for _, want := range []string{"dm-b", "1 untracked", "--force"} {
		if !strings.Contains(res.Stderr, want) {
			t.Errorf("stderr missing %q\n--- got ---\n%s", want, res.Stderr)
		}
	}
	if strings.Contains(res.Stderr, "removed ") {
		t.Errorf("nothing should have been removed\n--- got ---\n%s", res.Stderr)
	}
}

// The non-interactive error names every dirty target, not just the first,
// so one run tells the user the whole story.
func TestRm_DirtyErrorNamesEveryDirtyTarget(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "dn-a", "dn-b", "dn-c")
	mustWrite(t, filepath.Join(paths[0], "one.txt"), "x\n")
	mustWrite(t, filepath.Join(paths[2], "two.txt"), "y\n")

	res := runWT(t, repo, "rm", "dn-a", "dn-b", "dn-c", "--non-interactive")
	if res.ExitCode == 0 {
		t.Fatal("expected refusal")
	}
	for _, want := range []string{"dn-a", "dn-c"} {
		if !strings.Contains(res.Stderr, want) {
			t.Errorf("stderr missing %q\n--- got ---\n%s", want, res.Stderr)
		}
	}
}

// --force still removes, but says what it destroyed — the only record that
// the work ever existed, since an untracked file leaves no reflog entry.
func TestRm_ForceReportsDiscardedWork(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "df-a", "df-b")
	mustWrite(t, filepath.Join(paths[1], "scratch.txt"), "work\n")
	mustWrite(t, filepath.Join(paths[1], "README"), "edited\n")

	res := runWT(t, repo, "rm", "df-a", "df-b", "--non-interactive", "--force")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	for _, p := range paths {
		mustNotExist(t, p)
	}
	if !strings.Contains(res.Stderr, "1 modified, 1 untracked") {
		t.Errorf("expected the discarded counts\n--- got ---\n%s", res.Stderr)
	}
	if !strings.Contains(res.Stderr, "df-b") || !strings.Contains(res.Stderr, "warning") {
		t.Errorf("expected a warning naming df-b\n--- got ---\n%s", res.Stderr)
	}
	if strings.Contains(res.Stderr, "df-a has uncommitted") {
		t.Errorf("clean worktree should draw no warning\n--- got ---\n%s", res.Stderr)
	}
}

// A clean run stays quiet: the preflight must not editorialise when there
// is nothing to report.
func TestRm_CleanTargetsProduceNoDirtyNoise(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "dq-a", "dq-b")

	res := runWT(t, repo, "rm", "dq-a", "dq-b", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	for _, p := range paths {
		mustNotExist(t, p)
	}
	for _, unwanted := range []string{"uncommitted", "skipping", "warning"} {
		if strings.Contains(res.Stderr, unwanted) {
			t.Errorf("unexpected %q in a clean run\n--- got ---\n%s", unwanted, res.Stderr)
		}
	}
}

// clean is a bulk sweep over worktrees the user never named, so a dirty one
// is skipped and reported rather than destroyed — and the rest still go.
func TestClean_SkipsDirtyWorktree(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "co-keep", "co-drop")
	mustWrite(t, filepath.Join(paths[0], "wip.txt"), "unfinished\n")

	mustGit(t, repo, "checkout", "-q", "main")
	for _, b := range []string{"co-keep", "co-drop"} {
		if err := orphanBranch(repo, b); err != nil {
			t.Fatalf("orphan %s: %v", b, err)
		}
	}

	res := runWT(t, repo, "clean", "--non-interactive", "--orphaned-only", "--no-fetch")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustExist(t, paths[0])
	mustNotExist(t, paths[1])
	if !strings.Contains(res.Stderr, "skipping") || !strings.Contains(res.Stderr, "co-keep") {
		t.Errorf("expected a skip line for co-keep\n--- got ---\n%s", res.Stderr)
	}
}

// An orphaned worktree with no local edits must not read as dirty: with its
// branch ref gone there is no HEAD to diff against, and git reports every
// tracked file as a staged addition.
func TestClean_OrphanedCleanWorktreeIsNotDirty(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "cn-a")
	mustGit(t, repo, "checkout", "-q", "main")
	if err := orphanBranch(repo, "cn-a"); err != nil {
		t.Fatalf("orphan: %v", err)
	}

	res := runWT(t, repo, "clean", "--non-interactive", "--orphaned-only", "--no-fetch")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, paths[0])
	if strings.Contains(res.Stderr, "skipping") {
		t.Errorf("clean orphan should not be skipped\n--- got ---\n%s", res.Stderr)
	}
}

// --force is about uncommitted work in the worktree, not about branch
// deletion: an unmerged branch still survives unless --force-branch says
// otherwise.
func TestRm_ForceDoesNotForceBranchDeletion(t *testing.T) {
	repo := newRepo(t)
	mkTrees(t, repo, "fb-a")
	wtPath := filepath.Join(repo, ".worktrees", "fb-a")
	mustWrite(t, filepath.Join(wtPath, "committed.txt"), "unmerged work\n")
	mustGit(t, wtPath, "add", "committed.txt")
	mustGit(t, wtPath, "commit", "-q", "-m", "unmerged")
	mustWrite(t, filepath.Join(wtPath, "scratch.txt"), "wip\n")

	res := runWT(t, repo, "rm", "fb-a", "--non-interactive", "--force", "--delete-branch")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, wtPath)
	if !strings.Contains(mustGit(t, repo, "branch", "--list", "--format=%(refname:short)"), "fb-a") {
		t.Error("unmerged branch should survive --force alone")
	}
}

func TestRm_ForceBranchDropsUnmergedBranch(t *testing.T) {
	repo := newRepo(t)
	mkTrees(t, repo, "fx-a")
	wtPath := filepath.Join(repo, ".worktrees", "fx-a")
	mustWrite(t, filepath.Join(wtPath, "committed.txt"), "unmerged work\n")
	mustGit(t, wtPath, "add", "committed.txt")
	mustGit(t, wtPath, "commit", "-q", "-m", "unmerged")

	res := runWT(t, repo, "rm", "fx-a", "--non-interactive", "--delete-branch", "--force-branch")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, wtPath)
	if strings.Contains(mustGit(t, repo, "branch", "--list", "--format=%(refname:short)"), "fx-a") {
		t.Error("--force-branch should have dropped the unmerged branch")
	}
}

// The trees dir can hold a directory that is a repository in its own right
// rather than the debris of a crashed removal. It is not a registered
// worktree, so rm treats it as a leftover — but it can still hold real
// uncommitted work, and --force must say what it destroyed.
func TestRm_ForeignRepoUnderTreesDirReportsItsDirt(t *testing.T) {
	repo := newRepo(t)
	foreign := filepath.Join(repo, ".worktrees", "foreign")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	mustGit(t, foreign, "init", "-q", "-b", "main", ".")
	mustWrite(t, filepath.Join(foreign, "a.txt"), "committed\n")
	mustGit(t, foreign, "add", "a.txt")
	mustGit(t, foreign, "commit", "-q", "-m", "init")
	mustWrite(t, filepath.Join(foreign, "a.txt"), "edited\n")
	mustWrite(t, filepath.Join(foreign, "b.txt"), "new\n")

	res := runWT(t, repo, "rm", "foreign", "--non-interactive", "--force")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, foreign)
	if !strings.Contains(res.Stderr, "1 modified, 1 untracked") {
		t.Errorf("expected the discarded counts\n--- got ---\n%s", res.Stderr)
	}
}

// A leftover from a crashed removal has a .git file pointing at records git
// has already pruned, so `git status` fails there. That must read as clean
// rather than erroring the run.
func TestRm_CrashedRemovalLeftoverStillRemovable(t *testing.T) {
	repo := newRepo(t)
	if r := runWT(t, repo, "new", "lo-a", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "lo-a")
	aside := filepath.Join(repo, ".worktrees", ".aside")
	if err := os.Rename(wtPath, aside); err != nil {
		t.Fatal(err)
	}
	mustGit(t, repo, "worktree", "prune")
	if err := os.Rename(aside, wtPath); err != nil {
		t.Fatal(err)
	}

	res := runWT(t, repo, "rm", "lo-a", "--non-interactive", "--force")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, wtPath)
}

// Naming the same worktree twice used to remove it and then fail the second
// pass with git complaining the path is not a working tree.
func TestRm_DuplicateArgsRemoveOnceAndSucceed(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "dup-a")

	res := runWT(t, repo, "rm", "dup-a", "dup-a", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, paths[0])
	if n := strings.Count(res.Stderr, "removed "); n != 1 {
		t.Errorf("removed reported %d times, want 1\n--- got ---\n%s", n, res.Stderr)
	}
}

// A detached worktree has no branch, but `list` shows it as #leaf and that
// is the only handle the user has for naming it.
func TestRm_DetachedWorktreeRemovableByLeafName(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "det-a")
	mustGit(t, paths[0], "checkout", "-q", "--detach", "HEAD")

	res := runWT(t, repo, "rm", "det-a", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, paths[0])
}

// breakStatus corrupts a worktree's own .git file so `git status` fails
// inside it while the main repo still lists it as a registered worktree.
// That is the one state where the dirty scan can't establish anything.
func breakStatus(t *testing.T, wtPath string) {
	t.Helper()
	mustWrite(t, filepath.Join(wtPath, ".git"), "gitdir: /nonexistent/path\n")
}

// scanDirty treats a failed `git status` as clean, so such a target never
// reaches dirtyTargets. Deciding the force list from that set left it
// unforced, and removeWorktree's independent re-check then refused it — on a
// run that passed --force. Verified against the pre-fix binary: it exits 1
// and the worktree survives.
func TestRm_ForceRemovesAWorktreeWhoseStatusCheckFails(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "sf-a")
	breakStatus(t, paths[0])

	res := runWT(t, repo, "rm", "sf-a", "--non-interactive", "--force")
	if res.ExitCode != 0 {
		t.Fatalf("--force must not be refused for an unscannable worktree: exit %d\n%s", res.ExitCode, res.Stderr)
	}
	mustNotExist(t, paths[0])
}

// Without --force the same worktree defers to git, which refuses it. The
// point is that the refusal is git's own and nothing is half-removed.
func TestRm_UnscannableWorktreeWithoutForceLeavesItIntact(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "sf-b")
	breakStatus(t, paths[0])

	res := runWT(t, repo, "rm", "sf-b", "--non-interactive")
	if res.ExitCode == 0 {
		t.Fatal("expected git to refuse a worktree it cannot validate")
	}
	mustExist(t, paths[0])
	if !strings.Contains(res.Stderr, "sf-b") {
		t.Errorf("error should name the worktree\n--- got ---\n%s", res.Stderr)
	}
}

// A batch must not be stalled by one unscannable member: the others still go.
func TestRm_UnscannableTargetDoesNotBlockItsBatchUnderForce(t *testing.T) {
	repo := newRepo(t)
	paths := mkTrees(t, repo, "sf-x", "sf-y", "sf-z")
	breakStatus(t, paths[1])
	mustWrite(t, filepath.Join(paths[2], "wip.txt"), "work\n")

	res := runWT(t, repo, "rm", "sf-x", "sf-y", "sf-z", "--non-interactive", "--force")
	if res.ExitCode != 0 {
		t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
	}
	for _, p := range paths {
		mustNotExist(t, p)
	}
	// sf-z's dirt was scannable, so it is still reported; sf-y's was not.
	if !strings.Contains(res.Stderr, "1 untracked") {
		t.Errorf("expected the scannable target's counts\n--- got ---\n%s", res.Stderr)
	}
}
