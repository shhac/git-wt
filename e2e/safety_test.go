package e2e

// Tests in this file pin shared invariants of the worktree-creation
// commands (`new`, `add`) so that future refactors of the shared pipeline
// can't silently drop a guard. They cover paths that the existing happy-
// path E2E tests don't reach: dirty-repo refusal, bare-repo refusal, and
// the contract that copy-config failures are non-fatal.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMergeMarker drops a MERGE_HEAD file so wt.IsClean reports an
// in-progress merge without actually starting one.
func writeMergeMarker(t *testing.T, repo string) {
	t.Helper()
	marker := filepath.Join(repo, ".git", "MERGE_HEAD")
	if err := os.WriteFile(marker, []byte("0000000000000000000000000000000000000000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNew_RefusesWhenMergeInProgress(t *testing.T) {
	repo := newRepo(t)
	writeMergeMarker(t, repo)

	res := runWT(t, repo, "new", "feat-x", "--non-interactive", "--no-copy")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit during merge in progress")
	}
	if !strings.Contains(res.Stderr, "merge in progress") {
		t.Errorf("expected `merge in progress` in stderr, got: %s", res.Stderr)
	}
	mustNotExist(t, filepath.Join(repo, ".gwt", "feat-x"))
}

func TestAdd_RefusesWhenMergeInProgress(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "feat-y")
	writeMergeMarker(t, repo)

	res := runWT(t, repo, "add", "feat-y", "--non-interactive", "--no-copy")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit during merge in progress")
	}
	if !strings.Contains(res.Stderr, "merge in progress") {
		t.Errorf("expected `merge in progress` in stderr, got: %s", res.Stderr)
	}
	mustNotExist(t, filepath.Join(repo, ".gwt", "feat-y"))
}

// TestNew_RefusesInBareRepo and TestAdd_RefusesInBareRepo pin that the
// commands error out in a bare repo via wt.Inspect's "not in a git
// repository" error path. The explicit Bare guard in requireMutableRepo
// is currently unreachable because rev-parse --show-toplevel errors out
// first; pinning the message means that if Inspect is ever reordered so
// the Bare guard starts firing (with the "cannot create worktrees in a
// bare repository" wording), these tests will fail loudly and force the
// developer to confirm the change.
func TestNew_RefusesInBareRepo(t *testing.T) {
	bare := filepath.Join(t.TempDir(), "bare.git")
	mustGit(t, "", "init", "-q", "--bare", bare)

	res := runWT(t, bare, "new", "feat-bare", "--non-interactive", "--no-copy")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit in bare repo")
	}
	if !strings.Contains(res.Stderr, "not in a git repository") {
		t.Errorf("expected bare-repo refusal via the `not in a git repository` path, got: %s", res.Stderr)
	}
}

func TestAdd_RefusesInBareRepo(t *testing.T) {
	bare := filepath.Join(t.TempDir(), "bare.git")
	mustGit(t, "", "init", "-q", "--bare", bare)

	res := runWT(t, bare, "add", "main", "--non-interactive", "--no-copy")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit in bare repo")
	}
	if !strings.Contains(res.Stderr, "not in a git repository") {
		t.Errorf("expected bare-repo refusal via the `not in a git repository` path, got: %s", res.Stderr)
	}
}

// TestNew_CopyFailureIsNonFatal and TestAdd_CopyFailureIsNonFatal pin the
// contract that a copy-config error doesn't abort the worktree creation —
// the worktree exists, the path is emitted, and the failure is reported as
// a warning. A refactor that consolidates the copy step but makes errors
// fatal must not survive these tests.
func TestNew_CopyFailureIsNonFatal(t *testing.T) {
	repo := newRepo(t)
	// `[` opens a character class with no `]`. filepath.Match treats this
	// as ErrBadPattern — copyConfigs surfaces it, the command warns.
	mustWrite(t, filepath.Join(repo, ".git-wt-copy-files"), "[unclosed\n")

	res := runWT(t, repo, "new", "feat-copyfail", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0 (copy failure should be non-fatal), got %d: %s", res.ExitCode, res.Stderr)
	}
	wtPath := filepath.Join(repo, ".gwt", "feat-copyfail")
	mustExist(t, wtPath)
	if !strings.Contains(res.Stderr, "warning: copy configs") {
		t.Errorf("expected `warning: copy configs` in stderr, got: %s", res.Stderr)
	}
	if !strings.Contains(res.Stdout, wtPath) {
		t.Errorf("expected emitted path on stdout, got: %s", res.Stdout)
	}
}

func TestAdd_CopyFailureIsNonFatal(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "feat-addcopyfail")
	mustWrite(t, filepath.Join(repo, ".git-wt-copy-files"), "[unclosed\n")

	res := runWT(t, repo, "add", "feat-addcopyfail", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0 (copy failure should be non-fatal), got %d: %s", res.ExitCode, res.Stderr)
	}
	wtPath := filepath.Join(repo, ".gwt", "feat-addcopyfail")
	mustExist(t, wtPath)
	if !strings.Contains(res.Stderr, "warning: copy configs") {
		t.Errorf("expected `warning: copy configs` in stderr, got: %s", res.Stderr)
	}
}

// TestNew_CustomParentDir mirrors TestAdd_CustomParentDir so both commands
// have end-to-end coverage of the --parent-dir flag.
func TestNew_CustomParentDir(t *testing.T) {
	repo := newRepo(t)
	custom := filepath.Join(repo, "my-trees")

	res := runWT(t, repo, "new", "feat-pd", "--parent-dir", custom, "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustExist(t, filepath.Join(custom, "feat-pd"))
	mustNotExist(t, filepath.Join(repo, ".gwt", "feat-pd"))
}
