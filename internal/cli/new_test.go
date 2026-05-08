package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hermeticGit configures the test environment so `git check-ignore` (and
// any other git invocation) doesn't pick up the developer's global or
// system gitignore. Required because some users include `.gwt/` in their
// global gitignore — which is the *correct* user-side fix this test is
// trying to suggest, but it would mask the warning we're testing.
func hermeticGit(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// captureStderr runs fn with os.Stderr replaced by a pipe, returns the
// captured bytes as a string.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	done := make(chan string)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				sb.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- sb.String()
	}()
	fn()
	w.Close()
	os.Stderr = old
	return <-done
}

// initRepo creates a fresh git repo at dir/repo with optional gitignore lines.
func initRepo(t *testing.T, gitignoreLines ...string) string {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	mustGitCmd := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	mustGitCmd("init", "-q", "-b", "main", ".")
	mustGitCmd("commit", "-q", "--allow-empty", "-m", "init")
	if len(gitignoreLines) > 0 {
		body := strings.Join(gitignoreLines, "\n") + "\n"
		if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// mkParent creates parentDir under repo so EvalSymlinks resolves consistently
// (real CLI flow always MkdirAll's the parent before warning).
func mkParent(t *testing.T, parent string) {
	t.Helper()
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestWarnIfParentNotIgnored_NotIgnored_Fires(t *testing.T) {
	hermeticGit(t)
	repo := initRepo(t) // no .gitignore
	parent := filepath.Join(repo, ".gwt")
	mkParent(t, parent)
	got := captureStderr(t, func() {
		warnIfParentNotIgnored(context.Background(), repo, parent)
	})
	if !strings.Contains(got, ".gwt/") || !strings.Contains(got, ".gitignore") {
		t.Errorf("expected hint mentioning .gwt/ and .gitignore; got:\n%s", got)
	}
}

func TestWarnIfParentNotIgnored_Ignored_Silent(t *testing.T) {
	repo := initRepo(t, ".gwt/")
	parent := filepath.Join(repo, ".gwt")
	mkParent(t, parent)
	got := captureStderr(t, func() {
		warnIfParentNotIgnored(context.Background(), repo, parent)
	})
	if got != "" {
		t.Errorf("expected silence when .gwt/ is ignored; got:\n%s", got)
	}
}

func TestWarnIfParentNotIgnored_OutsideRepo_Silent(t *testing.T) {
	repo := initRepo(t)
	outside := t.TempDir()
	got := captureStderr(t, func() {
		warnIfParentNotIgnored(context.Background(), repo, outside)
	})
	if got != "" {
		t.Errorf("expected silence when parent is outside repo; got:\n%s", got)
	}
}

func TestWarnIfParentNotIgnored_RepoRootItself_Silent(t *testing.T) {
	// We never want to suggest the user gitignore their entire repo.
	repo := initRepo(t)
	got := captureStderr(t, func() {
		warnIfParentNotIgnored(context.Background(), repo, repo)
	})
	if got != "" {
		t.Errorf("expected silence when parent == repo root; got:\n%s", got)
	}
}

func TestWarnIfParentNotIgnored_NestedPath_FiresWithRelative(t *testing.T) {
	hermeticGit(t)
	repo := initRepo(t)
	parent := filepath.Join(repo, "foo", "bar")
	mkParent(t, parent)
	got := captureStderr(t, func() {
		warnIfParentNotIgnored(context.Background(), repo, parent)
	})
	// Forward-slash output regardless of platform.
	if !strings.Contains(got, "foo/bar/") {
		t.Errorf("expected hint to mention foo/bar/; got:\n%s", got)
	}
}
