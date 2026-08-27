package e2e

import (
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
