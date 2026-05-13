package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

// Happy path: clean tree, eject default branch name → worktree at .gwt/<branch>/.
func TestEject_CleanBranch(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "checkout", "-q", "-b", "feat-clean")

	res := runWT(t, repo, "eject", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("eject exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustExist(t, filepath.Join(repo, ".gwt", "feat-clean"))

	// Main tree HEAD should be on main now.
	head := mustGit(t, repo, "rev-parse", "--abbrev-ref", "HEAD")
	if head != "main" {
		t.Errorf("main tree HEAD = %q, want main", head)
	}

	// Path emitted to stdout (bare mode).
	if !strings.Contains(res.Stdout, filepath.Join(repo, ".gwt", "feat-clean")) {
		t.Errorf("expected emitted path in stdout, got: %s", res.Stdout)
	}
}

// Dirty tree with tracked-modified + staged: stash captures both,
// apply --index restores staged-as-staged and modified-as-modified.
func TestEject_DirtyTrackedRestoresStagedAndModified(t *testing.T) {
	repo := newRepo(t)
	// Add and commit a file so we have something to modify.
	mustWrite(t, filepath.Join(repo, "tracked.txt"), "original\n")
	mustGit(t, repo, "add", "tracked.txt")
	mustGit(t, repo, "commit", "-q", "-m", "add tracked.txt")
	mustGit(t, repo, "checkout", "-q", "-b", "feat-dirty")

	// Stage a modification + leave another modification unstaged.
	mustWrite(t, filepath.Join(repo, "tracked.txt"), "staged\n")
	mustGit(t, repo, "add", "tracked.txt")
	mustWrite(t, filepath.Join(repo, "tracked.txt"), "staged+more\n")

	res := runWT(t, repo, "eject", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("eject exit %d: %s", res.ExitCode, res.Stderr)
	}

	wtPath := filepath.Join(repo, ".gwt", "feat-dirty")
	mustExist(t, wtPath)

	// In the new worktree: staged content is in the index, working tree has the further-modified content.
	stagedContent := mustGit(t, wtPath, "show", ":tracked.txt")
	if stagedContent != "staged" {
		t.Errorf("staged content = %q, want %q", stagedContent, "staged")
	}
	// Working tree has the unstaged-on-top modification.
	wtContent, err := readFile(filepath.Join(wtPath, "tracked.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(wtContent) != "staged+more" {
		t.Errorf("worktree content = %q, want %q", strings.TrimSpace(wtContent), "staged+more")
	}

	// Main tree: tracked.txt back to original.
	mainContent, err := readFile(filepath.Join(repo, "tracked.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(mainContent) != "original" {
		t.Errorf("main tracked.txt = %q, want %q", strings.TrimSpace(mainContent), "original")
	}
}

// Untracked files travel with the eject; restored as untracked (not in index).
func TestEject_UntrackedRestoresAsUntracked(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "checkout", "-q", "-b", "feat-untracked")

	// Create an untracked file (and gitignore-anchor a config-style file).
	mustWrite(t, filepath.Join(repo, ".env"), "FOO=1\n")

	res := runWT(t, repo, "eject", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("eject exit %d: %s", res.ExitCode, res.Stderr)
	}

	wtPath := filepath.Join(repo, ".gwt", "feat-untracked")
	mustExist(t, filepath.Join(wtPath, ".env"))

	// .env should NOT be in the index in the new worktree.
	status := mustGit(t, wtPath, "status", "--porcelain", ".env")
	// "??" means untracked in porcelain format.
	if !strings.HasPrefix(status, "??") {
		t.Errorf(".env status = %q, want untracked (??...)", status)
	}

	// And it should be gone from the main tree (stash --include-untracked moved it).
	mustNotExist(t, filepath.Join(repo, ".env"))
}

func TestEject_LeafOverride(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "checkout", "-q", "-b", "feat-leaf")

	res := runWT(t, repo, "eject", "custom-leaf", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("eject exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustExist(t, filepath.Join(repo, ".gwt", "custom-leaf"))
	mustNotExist(t, filepath.Join(repo, ".gwt", "feat-leaf"))
}

func TestEject_BaseOverride(t *testing.T) {
	repo := newRepo(t)
	// Create a non-default base branch.
	mustGit(t, repo, "checkout", "-q", "-b", "trunk")
	mustGit(t, repo, "checkout", "-q", "main")
	mustGit(t, repo, "checkout", "-q", "-b", "feat-base")

	res := runWT(t, repo, "eject", "--base", "trunk", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("eject exit %d: %s", res.ExitCode, res.Stderr)
	}
	head := mustGit(t, repo, "rev-parse", "--abbrev-ref", "HEAD")
	if head != "trunk" {
		t.Errorf("main tree HEAD = %q, want trunk", head)
	}
}

func TestEject_RefusesOnMainBranch(t *testing.T) {
	repo := newRepo(t)
	// Already on main (newRepo's default).
	res := runWT(t, repo, "eject", "--non-interactive")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit when ejecting from main")
	}
	if !strings.Contains(res.Stderr, "base branch") && !strings.Contains(res.Stderr, "main") {
		t.Errorf("expected base/main mention in error, got: %s", res.Stderr)
	}
}

func TestEject_RefusesOnDetachedHEAD(t *testing.T) {
	repo := newRepo(t)
	// Commit a second commit so we can detach onto an SHA.
	mustGit(t, repo, "commit", "-q", "--allow-empty", "-m", "second")
	sha := mustGit(t, repo, "rev-parse", "HEAD")
	mustGit(t, repo, "checkout", "-q", sha)

	res := runWT(t, repo, "eject", "--non-interactive")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit on detached HEAD")
	}
	if !strings.Contains(res.Stderr, "detached") && !strings.Contains(res.Stderr, "not on a branch") {
		t.Errorf("expected detached-HEAD mention, got: %s", res.Stderr)
	}
}

func TestEject_RefusesInsideWorktree(t *testing.T) {
	repo := newRepo(t)
	// Create a worktree via `gwt new`, then try to `gwt eject` from inside it.
	if r := runWT(t, repo, "new", "feat-inside", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("new: %s", r.Stderr)
	}
	wtPath := filepath.Join(repo, ".gwt", "feat-inside")

	res := runWT(t, wtPath, "eject", "--non-interactive")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit from inside a worktree")
	}
	if !strings.Contains(res.Stderr, "main working tree") && !strings.Contains(res.Stderr, "worktree") {
		t.Errorf("expected `main working tree` mention, got: %s", res.Stderr)
	}
}

func TestEject_RefusesWhenNoBaseExists(t *testing.T) {
	repo := newRepo(t)
	// Rename main → something-else, then create feat-x.
	mustGit(t, repo, "branch", "-m", "main", "trunk-only")
	mustGit(t, repo, "checkout", "-q", "-b", "feat-no-base")

	res := runWT(t, repo, "eject", "--non-interactive")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit when no main/master exists")
	}
	if !strings.Contains(res.Stderr, "main") && !strings.Contains(res.Stderr, "master") && !strings.Contains(res.Stderr, "base") {
		t.Errorf("expected base-branch mention, got: %s", res.Stderr)
	}
}

func TestEject_RefusesWhenCurrentBranchIsBaseOverride(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "checkout", "-q", "-b", "trunk")

	res := runWT(t, repo, "eject", "--base", "trunk", "--non-interactive")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit when current branch is the base override")
	}
}

func TestEject_EmitsPathOnFD(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "checkout", "-q", "-b", "feat-fd-eject")

	res := runWTFD(t, repo, "eject", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("eject exit %d: %s", res.ExitCode, res.Stderr)
	}
	got := strings.TrimSpace(res.FD3)
	if !strings.HasSuffix(got, "/demo/.gwt/feat-fd-eject") {
		t.Errorf("fd3 = %q, want path ending in /demo/.gwt/feat-fd-eject", got)
	}
}

func TestEject_DropsStashOnSuccess(t *testing.T) {
	repo := newRepo(t)
	mustWrite(t, filepath.Join(repo, "tracked.txt"), "v1\n")
	mustGit(t, repo, "add", "tracked.txt")
	mustGit(t, repo, "commit", "-q", "-m", "v1")
	mustGit(t, repo, "checkout", "-q", "-b", "feat-stash-drop")
	mustWrite(t, filepath.Join(repo, "tracked.txt"), "v2\n")

	// Pre-condition: no stash entries.
	if got := mustGit(t, repo, "stash", "list"); got != "" {
		t.Fatalf("expected empty stash list pre-eject, got: %q", got)
	}

	res := runWT(t, repo, "eject", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("eject exit %d: %s", res.ExitCode, res.Stderr)
	}

	// Post-condition: stash entry was dropped after a clean apply.
	if got := mustGit(t, repo, "stash", "list"); got != "" {
		t.Errorf("expected empty stash list post-eject, got: %q", got)
	}
}

func TestEject_CustomParentDir(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "checkout", "-q", "-b", "feat-pd-eject")
	custom := filepath.Join(repo, "my-trees")

	res := runWT(t, repo, "eject", "--parent-dir", custom, "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("eject exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustExist(t, filepath.Join(custom, "feat-pd-eject"))
	mustNotExist(t, filepath.Join(repo, ".gwt", "feat-pd-eject"))
}
