package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTempRepo creates a fresh git repo under t.TempDir() with the
// hermetic env vars our other test suites use, and returns its path.
// Local to this package so we don't introduce a testutil → git cycle.
func initTempRepo(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := RunIn(context.Background(), repo, "init", "--quiet"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return repo
}

func TestExitError_ExitCode_MissingKey(t *testing.T) {
	repo := initTempRepo(t)
	_, err := RunIn(context.Background(), repo, "config", "--local", "--get", "wt.does-not-exist")
	if err == nil {
		t.Fatal("expected error from missing config key")
	}
	var ee *ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *git.ExitError in chain, got %T: %v", err, err)
	}
	// `git config --get` exits 1 when the key is missing.
	if got := ee.ExitCode(); got != 1 {
		t.Errorf("ExitCode() = %d, want 1", got)
	}
}

func TestExitError_ExitCode_UnsetMissingKey(t *testing.T) {
	repo := initTempRepo(t)
	_, err := RunIn(context.Background(), repo, "config", "--local", "--unset", "wt.does-not-exist")
	if err == nil {
		t.Fatal("expected error from unset of missing key")
	}
	var ee *ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *git.ExitError in chain, got %T", err)
	}
	// `git config --unset` exits 5 specifically when the key was unset.
	// This is the contract that config.Unset relies on for idempotency.
	if got := ee.ExitCode(); got != 5 {
		t.Errorf("ExitCode() = %d, want 5", got)
	}
}

func TestExitError_UnwrapToExecExitError(t *testing.T) {
	repo := initTempRepo(t)
	_, err := RunIn(context.Background(), repo, "config", "--local", "--get", "wt.missing")
	if err == nil {
		t.Fatal("expected error")
	}
	// Unwrap should expose the underlying *exec.ExitError so callers
	// can walk to it directly without knowing about *git.ExitError.
	var execErr *exec.ExitError
	if !errors.As(err, &execErr) {
		t.Fatalf("expected *exec.ExitError reachable via Unwrap, got chain: %v", err)
	}
	if execErr.ExitCode() != 1 {
		t.Errorf("exec.ExitError.ExitCode() = %d, want 1", execErr.ExitCode())
	}
}

func TestExitError_ExitCode_NilInnerReturnsSentinel(t *testing.T) {
	// Defensive branch in ExitCode(): if somehow inner is nil, return -1
	// rather than panicking. Exercised here by direct construction —
	// the production path always populates inner.
	ee := &ExitError{Args: []string{"foo"}, Stderr: "boom"}
	if got := ee.ExitCode(); got != -1 {
		t.Errorf("ExitCode() with nil inner = %d, want -1", got)
	}
}

func TestExitError_Error_FormatsArgsAndStderr(t *testing.T) {
	ee := &ExitError{Args: []string{"config", "--get", "x"}, Stderr: "key not found"}
	got := ee.Error()
	want := "git config --get x: key not found"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
