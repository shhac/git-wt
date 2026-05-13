package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newRepo initialises a fresh git repo with one initial commit and returns
// its absolute path. The repo is created under t.TempDir(), so cleanup is
// automatic.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repo := filepath.Join(dir, "demo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	mustGit(t, repo, "init", "-q", "-b", "main", ".")
	mustWrite(t, filepath.Join(repo, "README"), "init\n")
	mustGit(t, repo, "add", "README")
	mustGit(t, repo, "commit", "-q", "-m", "init")
	return repo
}

// newRepoWithRemote sets up a fresh repo + a bare origin remote, with main pushed.
// Returns (repo, originBare).
func newRepoWithRemote(t *testing.T) (string, string) {
	t.Helper()
	repo := newRepo(t)
	origin := filepath.Join(filepath.Dir(repo), "origin.git")
	mustGit(t, "", "init", "-q", "--bare", origin)
	mustGit(t, repo, "remote", "add", "origin", origin)
	mustGit(t, repo, "push", "-q", "-u", "origin", "main")
	return repo, origin
}

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = hermeticEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readFile reads a file or fails the test.
func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func mustNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected %s to not exist", path)
	}
}

// orphanBranch deletes the local branch's ref file directly, simulating the
// "branch deleted but worktree still tracks it" scenario that `clean
// --orphaned-only` is meant to find. Git itself refuses `branch -D` while a
// worktree uses the branch; the ref-file shortcut lets us set up the test.
func orphanBranch(repo, branch string) error {
	refPath := filepath.Join(repo, ".git", "refs", "heads", branch)
	if err := os.Remove(refPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
