// Package testutil holds helpers shared across multiple test files. It
// is import-safe in non-test code too, but the functions only make
// sense inside a *testing.T flow.
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shhac/git-wt/internal/git"
)

// SetupTempGitconfig initialises an isolated git environment for a
// test: a temp HOME (so `git config --global` writes land somewhere
// disposable), a temp repo with `git init`, GIT_CONFIG_NOSYSTEM set so
// real /etc/gitconfig can't leak in, and a chdir into the repo so
// `git config --local` works without extra plumbing.
//
// All env mutations use t.Setenv and t.Cleanup so the test framework
// restores the original values automatically. Returns the absolute
// path of the new repo.
//
// Not safe for t.Parallel(): the chdir mutates process-global state.
func SetupTempGitconfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	repoDir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := git.RunIn(ctx, repoDir, "init", "--quiet"); err != nil {
		t.Fatalf("git init: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(repoDir); err != nil {
		t.Fatal(err)
	}
	return repoDir
}
