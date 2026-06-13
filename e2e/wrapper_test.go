package e2e

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runUnderWrapper evals the generated `gwt` alias in bash, runs script from
// dir, and returns trimmed stdout plus stderr. The wrapper bakes in
// os.Executable() at generation time, so the evaled function drives the
// freshly-built test binary through the real fd-capture path.
func runUnderWrapper(t *testing.T, repo, dir, script string) (string, string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not on PATH; wrapper-shell test requires bash")
	}
	aliasRes := runWT(t, repo, "alias", "gwt")
	if aliasRes.ExitCode != 0 {
		t.Fatalf("alias: exit %d, stderr: %s", aliasRes.ExitCode, aliasRes.Stderr)
	}

	cmd := exec.Command("bash", "-c", aliasRes.Stdout+"\n"+script)
	cmd.Dir = dir
	cmd.Env = hermeticEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("bash run: %v\nstderr: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), stderr.String()
}

// assertSamePath compares two paths after resolving symlinks (macOS TempDirs
// live under /var → /private/var).
func assertSamePath(t *testing.T, label, got, want string) {
	t.Helper()
	g, _ := filepath.EvalSymlinks(got)
	w, _ := filepath.EvalSymlinks(want)
	if g == "" || w == "" {
		t.Fatalf("%s: evalSymlinks failed: got=%q want=%q", label, got, want)
	}
	if g != w {
		t.Errorf("%s = %q, want %q", label, got, want)
	}
}

func TestWrapper_NewCdsIntoWorktree(t *testing.T) {
	repo := newRepo(t)
	pwd, _ := runUnderWrapper(t, repo, repo, "gwt new wrap-new --no-copy\npwd\n")
	assertSamePath(t, "pwd after new", pwd, filepath.Join(repo, ".worktrees", "wrap-new"))
}

func TestWrapper_AddCdsIntoWorktree(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "branch", "wrap-add")
	pwd, _ := runUnderWrapper(t, repo, repo, "gwt add wrap-add\npwd\n")
	assertSamePath(t, "pwd after add", pwd, filepath.Join(repo, ".worktrees", "wrap-add"))
}

func TestWrapper_EjectCdsIntoWorktree(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "checkout", "-q", "-b", "wrap-eject")
	pwd, _ := runUnderWrapper(t, repo, repo, "gwt eject\npwd\n")
	assertSamePath(t, "pwd after eject", pwd, filepath.Join(repo, ".worktrees", "wrap-eject"))
	if br := mustGit(t, repo, "branch", "--show-current"); br != "main" {
		t.Errorf("main tree on %q after eject, want main", br)
	}
}

func TestWrapper_RmCurrentWorktreeBouncesToMain(t *testing.T) {
	repo := newRepo(t)
	if r := runWT(t, repo, "new", "wrap-rm", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "wrap-rm")
	script := "cd '" + wtPath + "'\ngwt rm wrap-rm\npwd\n"
	pwd, _ := runUnderWrapper(t, repo, repo, script)
	assertSamePath(t, "pwd after rm bounce", pwd, repo)
	mustNotExist(t, wtPath)
}

// TestWrapper_RmOtherWorktreeStaysPut pins the no-bounce case: removing a
// worktree you are not inside must not emit a path, so the shell stays where
// it is.
func TestWrapper_RmOtherWorktreeStaysPut(t *testing.T) {
	repo := newRepo(t)
	if r := runWT(t, repo, "new", "wrap-rm-other", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	pwd, _ := runUnderWrapper(t, repo, repo, "gwt rm wrap-rm-other\npwd\n")
	assertSamePath(t, "pwd after rm of other worktree", pwd, repo)
	mustNotExist(t, filepath.Join(repo, ".worktrees", "wrap-rm-other"))
}
