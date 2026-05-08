// Package e2e drives the built git-wt binary against fresh temp repos
// under t.TempDir() (which lives under the OS temp dir, never inside this
// project). Each test gets a clean repo of its own; nothing mutates the
// developer's real worktrees.
package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// binPath is set in TestMain to the freshly-built git-wt binary.
var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "git-wt-e2e-bin-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "mktemp:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binPath = filepath.Join(tmpDir, "git-wt")
	cmd := exec.Command("go", "build", "-o", binPath, "../cmd/git-wt")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "build failed:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// runResult bundles the outputs of a single git-wt invocation.
type runResult struct {
	Stdout   string
	Stderr   string
	FD3      string // captured if RunWithFD is used; "" otherwise
	ExitCode int
}

// runWT invokes the test binary in cwd with args. Stdin is /dev/null.
func runWT(t *testing.T, cwd string, args ...string) runResult {
	t.Helper()
	return doRun(t, cwd, false, args...)
}

// runWTFD invokes the binary with fd 3 wired to a pipe. Use this to test the
// wrapper protocol: pass `--fd 3` (or rely on the default) and read FD3 from
// the result.
func runWTFD(t *testing.T, cwd string, args ...string) runResult {
	t.Helper()
	return doRun(t, cwd, true, args...)
}

func doRun(t *testing.T, cwd string, withFD bool, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = cwd
	cmd.Env = hermeticEnv()
	cmd.Stdin = nil // ensures non-interactive auto-detection

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	var captured bytes.Buffer
	var wg sync.WaitGroup
	var fdReader, fdWriter *os.File
	if withFD {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("pipe: %v", err)
		}
		fdReader, fdWriter = r, w
		cmd.ExtraFiles = []*os.File{w} // becomes fd 3 in the child
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = io.Copy(&captured, r)
		}()
	}

	err := cmd.Run()
	if withFD {
		// Close our copy of the write end so the reader goroutine sees EOF
		// (the child has its own clone via ExtraFiles, already closed by Run).
		_ = fdWriter.Close()
		wg.Wait()
		_ = fdReader.Close()
	}

	res := runResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		FD3:    captured.String(),
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.ExitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run failed (not an exit error): %v", err)
	}
	return res
}

// hermeticEnv returns a minimal env that lets git run without depending on
// the developer's user.name/user.email and without picking up the global or
// system gitignore (which on developer machines often includes things like
// `.gwt/` — exactly the path our tests want to assert is *not* covered).
// PATH is preserved (we still need git).
func hermeticEnv() []string {
	keep := []string{"PATH", "HOME", "USER", "TMPDIR", "LANG", "LC_ALL"}
	env := []string{
		"GIT_AUTHOR_NAME=git-wt-test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=git-wt-test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		// Quiet down git's hint output that would otherwise flood stderr
		"GIT_TERMINAL_PROMPT=0",
		// Ignore developer-side gitconfig + gitignore_global so check-ignore
		// behaviour is reproducible.
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
	}
	for _, k := range keep {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}

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
