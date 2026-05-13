package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAdd_OneArgLocalBranch(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "feat-local")

	res := runWT(t, repo, "add", "feat-local", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}

	wtPath := filepath.Join(repo, ".gwt", "feat-local")
	mustExist(t, wtPath)

	if !strings.Contains(res.Stdout, wtPath) {
		t.Errorf("expected stdout to include %s, got: %s", wtPath, res.Stdout)
	}
}

func TestAdd_OneArgLocalSlashBranchNestsUnderGwt(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "paul/auth-bug")

	res := runWT(t, repo, "add", "paul/auth-bug", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}

	wtPath := filepath.Join(repo, ".gwt", "paul", "auth-bug")
	mustExist(t, wtPath)
}

func TestAdd_OneArgRemoteBranchCreatesTrackingLocal(t *testing.T) {
	repo, _ := newRepoWithRemote(t)
	// Set up origin/feature-x with no local counterpart.
	mustGit(t, repo, "checkout", "-q", "-b", "feature-x")
	mustGit(t, repo, "push", "-q", "-u", "origin", "feature-x")
	mustGit(t, repo, "checkout", "-q", "main")
	mustGit(t, repo, "branch", "-D", "feature-x")

	res := runWT(t, repo, "add", "origin/feature-x", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}

	// Leaf derived from remote rest: feature-x (NOT origin/feature-x).
	wtPath := filepath.Join(repo, ".gwt", "feature-x")
	mustExist(t, wtPath)
	mustNotExist(t, filepath.Join(repo, ".gwt", "origin"))

	// Local branch feature-x exists and tracks origin/feature-x.
	upstream := mustGit(t, repo, "for-each-ref", "--format=%(upstream:short)", "refs/heads/feature-x")
	if upstream != "origin/feature-x" {
		t.Errorf("upstream = %q, want origin/feature-x", upstream)
	}
}

func TestAdd_TwoArgLeafOverrideLocal(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "my-branch")

	res := runWT(t, repo, "add", "custom-leaf", "my-branch", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}

	mustExist(t, filepath.Join(repo, ".gwt", "custom-leaf"))
	mustNotExist(t, filepath.Join(repo, ".gwt", "my-branch"))
}

func TestAdd_TwoArgLeafOverrideRemote(t *testing.T) {
	repo, _ := newRepoWithRemote(t)
	mustGit(t, repo, "checkout", "-q", "-b", "feature-x")
	mustGit(t, repo, "push", "-q", "-u", "origin", "feature-x")
	mustGit(t, repo, "checkout", "-q", "main")
	mustGit(t, repo, "branch", "-D", "feature-x")

	res := runWT(t, repo, "add", "review", "origin/feature-x", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}

	mustExist(t, filepath.Join(repo, ".gwt", "review"))
	mustNotExist(t, filepath.Join(repo, ".gwt", "feature-x"))

	// Local feature-x branch should still be created with tracking.
	branches := mustGit(t, repo, "branch", "--list", "feature-x")
	if !strings.Contains(branches, "feature-x") {
		t.Errorf("expected local feature-x branch to be created; got: %q", branches)
	}
}

func TestAdd_MissingRefErrors(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "add", "nope", "--non-interactive")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit for missing ref")
	}
	if !strings.Contains(res.Stderr, "no such branch") {
		t.Errorf("expected `no such branch` in error; got: %s", res.Stderr)
	}
}

func TestAdd_DoesNotCreateNewBranch(t *testing.T) {
	repo := newRepo(t)
	// `new-branch` does not exist; add should refuse, not create one.
	res := runWT(t, repo, "add", "new-branch", "--non-interactive")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit (add should not create new branches)")
	}
	if got := mustGit(t, repo, "branch", "--list", "new-branch"); got != "" {
		t.Errorf("expected no branch created; got: %q", got)
	}
}

func TestAdd_CopySpecRunsByDefault(t *testing.T) {
	repo := newRepo(t)
	mustWrite(t, filepath.Join(repo, ".env"), "FOO=1\n")
	mustGit(t, repo, "branch", "feat-copy")

	res := runWT(t, repo, "add", "feat-copy", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}

	mustExist(t, filepath.Join(repo, ".gwt", "feat-copy", ".env"))
}

func TestAdd_NoCopyFlagSkipsCopy(t *testing.T) {
	repo := newRepo(t)
	mustWrite(t, filepath.Join(repo, ".env"), "FOO=1\n")
	mustGit(t, repo, "branch", "feat-nocopy")

	res := runWT(t, repo, "add", "feat-nocopy", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}

	mustNotExist(t, filepath.Join(repo, ".gwt", "feat-nocopy", ".env"))
}

func TestAdd_HintsWhenParentDirNotIgnored(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "feat-hint")
	res := runWT(t, repo, "add", "feat-hint", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stderr, ".gwt/") || !strings.Contains(res.Stderr, ".gitignore") {
		t.Errorf("expected gitignore hint mentioning `.gwt/` and `.gitignore`; got:\n%s", res.Stderr)
	}
}

func TestAdd_EmitsPathOnFD(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "feat-fd")

	res := runWTFD(t, repo, "add", "feat-fd", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d: %s", res.ExitCode, res.Stderr)
	}
	got := strings.TrimSpace(res.FD3)
	if !strings.HasSuffix(got, "/demo/.gwt/feat-fd") {
		t.Errorf("fd3 = %q, want path ending in /demo/.gwt/feat-fd", got)
	}
}

func TestAdd_ThenGoFindsWorktree(t *testing.T) {
	// End-to-end: add creates the worktree, go finds it. The worktree
	// produced by `add` should be a fully usable, listable worktree.
	repo := newRepo(t)
	mustGit(t, repo, "branch", "feat-go")

	if r := runWT(t, repo, "add", "feat-go", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("add: %s", r.Stderr)
	}

	res := runWTFD(t, repo, "go", "feat-go")
	if res.ExitCode != 0 {
		t.Fatalf("go exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.HasSuffix(strings.TrimSpace(res.FD3), "/demo/.gwt/feat-go") {
		t.Errorf("fd3 = %q, want path ending in /demo/.gwt/feat-go", res.FD3)
	}
}

func TestAdd_CustomParentDir(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "feat-pd")
	custom := filepath.Join(repo, "my-trees")

	res := runWT(t, repo, "add", "feat-pd", "--parent-dir", custom, "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustExist(t, filepath.Join(custom, "feat-pd"))
	mustNotExist(t, filepath.Join(repo, ".gwt", "feat-pd"))
}

func TestAdd_InvalidLeafRejected(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "real-branch")

	// Leaf with `..` should be rejected by ValidateBranchName before any
	// filesystem mutation.
	res := runWT(t, repo, "add", "../escape", "real-branch", "--non-interactive", "--no-copy")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit for leaf containing `..`")
	}
	if !strings.Contains(res.Stderr, "invalid leaf") {
		t.Errorf("expected `invalid leaf` error, got: %s", res.Stderr)
	}
}

func TestAdd_BranchAlreadyCheckedOutElsewhereErrors(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "dup")
	// First add: succeeds.
	res := runWT(t, repo, "add", "dup", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("first add exit %d: %s", res.ExitCode, res.Stderr)
	}
	// Second add with different leaf: git refuses since `dup` is checked out.
	res = runWT(t, repo, "add", "dup-2", "dup", "--non-interactive", "--no-copy")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit when branch is already checked out elsewhere")
	}
}
