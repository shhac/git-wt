package e2e

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// wrapperShells are the shells the generated alias is exercised under: zsh,
// where most people run it (the macOS default), and bash as the POSIX
// baseline. A shell missing from PATH is skipped, not failed.
var wrapperShells = []string{"bash", "zsh"}

// forEachShell runs test once per wrapper shell, as subtests.
func forEachShell(t *testing.T, test func(t *testing.T, shell string)) {
	t.Helper()
	for _, sh := range wrapperShells {
		t.Run(sh, func(t *testing.T) {
			if _, err := exec.LookPath(sh); err != nil {
				t.Skipf("%s not on PATH", sh)
			}
			test(t, sh)
		})
	}
}

// runUnderWrapper evals the generated `gwt` alias in shell, runs script from
// dir, and returns trimmed stdout plus stderr. The wrapper bakes in
// os.Executable() at generation time, so the evaled function drives the
// freshly-built test binary through the real fd-capture path. zsh runs with
// -f so the developer's startup files stay out of it.
func runUnderWrapper(t *testing.T, shell, repo, dir, script string) (string, string) {
	t.Helper()
	aliasRes := runWT(t, repo, "alias", "gwt")
	if aliasRes.ExitCode != 0 {
		t.Fatalf("alias: exit %d, stderr: %s", aliasRes.ExitCode, aliasRes.Stderr)
	}

	args := []string{"-c", aliasRes.Stdout + "\n" + script}
	if shell == "zsh" {
		args = append([]string{"-f"}, args...)
	}
	cmd := exec.Command(shell, args...)
	cmd.Dir = dir
	cmd.Env = hermeticEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s run: %v\nstderr: %s", shell, err, stderr.String())
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

func TestWrapper_GoCdsIntoWorktree(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		repo := newRepo(t)
		paths := mkTrees(t, repo, "wrap-go")
		pwd, _ := runUnderWrapper(t, shell, repo, repo, "gwt go wrap-go\npwd\n")
		assertSamePath(t, "pwd after go", pwd, paths[0])
	})
}

// lock and unlock never move the shell: they emit no path, so the wrapper
// must pass them through and leave pwd alone, success or failure.
func TestWrapper_LockAndUnlockLeaveTheShellInPlace(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		repo := newRepo(t)
		paths := mkTrees(t, repo, "wrap-lock")
		script := "cd '" + paths[0] + "'\n" +
			"gwt lock wrap-lock --reason 'from the wrapper'; echo lock=$?\n" +
			"gwt lock main; echo main=$?\n" +
			"gwt unlock wrap-lock; echo unlock=$?\n" +
			"pwd\n"
		out, stderr := runUnderWrapper(t, shell, repo, repo, script)
		lines := strings.Split(out, "\n")
		if got := strings.Join(lines[:len(lines)-1], " "); got != "lock=0 main=1 unlock=0" {
			t.Errorf("exit codes = %q, want lock=0 main=1 unlock=0\n--- stderr ---\n%s", got, stderr)
		}
		assertSamePath(t, "pwd after lock/unlock", lines[len(lines)-1], paths[0])
		if !strings.Contains(stderr, "unlocked wrap-lock (was locked") || !strings.Contains(stderr, "from the wrapper") {
			t.Errorf("unlock should report the released lock\n--- stderr ---\n%s", stderr)
		}
	})
}

func TestWrapper_NewCdsIntoWorktree(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		repo := newRepo(t)
		pwd, _ := runUnderWrapper(t, shell, repo, repo, "gwt new wrap-new --no-copy\npwd\n")
		assertSamePath(t, "pwd after new", pwd, filepath.Join(repo, ".worktrees", "wrap-new"))
	})
}

func TestWrapper_AddCdsIntoWorktree(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		repo := newRepo(t)
		mustGit(t, repo, "branch", "wrap-add")
		pwd, _ := runUnderWrapper(t, shell, repo, repo, "gwt add wrap-add\npwd\n")
		assertSamePath(t, "pwd after add", pwd, filepath.Join(repo, ".worktrees", "wrap-add"))
	})
}

func TestWrapper_EjectCdsIntoWorktree(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		repo := newRepo(t)
		mustGit(t, repo, "checkout", "-q", "-b", "wrap-eject")
		pwd, _ := runUnderWrapper(t, shell, repo, repo, "gwt eject\npwd\n")
		assertSamePath(t, "pwd after eject", pwd, filepath.Join(repo, ".worktrees", "wrap-eject"))
		if br := mustGit(t, repo, "branch", "--show-current"); br != "main" {
			t.Errorf("main tree on %q after eject, want main", br)
		}
	})
}

func TestWrapper_RmCurrentWorktreeBouncesToMain(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		repo := newRepo(t)
		if r := runWT(t, repo, "new", "wrap-rm", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
			t.Fatalf("setup: %s", r.Stderr)
		}
		wtPath := filepath.Join(repo, ".worktrees", "wrap-rm")
		script := "cd '" + wtPath + "'\ngwt rm wrap-rm\npwd\n"
		pwd, _ := runUnderWrapper(t, shell, repo, repo, script)
		assertSamePath(t, "pwd after rm bounce", pwd, repo)
		mustNotExist(t, wtPath)
	})
}

// TestWrapper_RmOtherWorktreeStaysPut pins the no-bounce case: removing a
// worktree you are not inside must not emit a path, so the shell stays where
// it is.
func TestWrapper_RmOtherWorktreeStaysPut(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		repo := newRepo(t)
		if r := runWT(t, repo, "new", "wrap-rm-other", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
			t.Fatalf("setup: %s", r.Stderr)
		}
		pwd, _ := runUnderWrapper(t, shell, repo, repo, "gwt rm wrap-rm-other\npwd\n")
		assertSamePath(t, "pwd after rm of other worktree", pwd, repo)
		mustNotExist(t, filepath.Join(repo, ".worktrees", "wrap-rm-other"))
	})
}

// TestWrapper_RmBouncesEvenWhenALaterTargetFails pins the fix in 28ff394.
// Removing the worktree you are standing in chdirs the process to the main
// repo up front, but the path only reaches the shell at the end of the run.
// With three targets and a failure on the second, the shell used to be left
// sitting in the first one's deleted directory.
func TestWrapper_RmBouncesEvenWhenALaterTargetFails(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		repo := newRepo(t)
		for _, n := range []string{"wrap-cur", "wrap-broken"} {
			if r := runWT(t, repo, "new", n, "--non-interactive", "--no-copy"); r.ExitCode != 0 {
				t.Fatalf("setup %s: %s", n, r.Stderr)
			}
		}
		curPath := filepath.Join(repo, ".worktrees", "wrap-cur")
		brokenPath := filepath.Join(repo, ".worktrees", "wrap-broken")
		breakWorktreeLink(t, brokenPath)

		script := "cd '" + curPath + "'\ngwt rm wrap-cur wrap-broken || true\npwd\n"
		pwd, stderr := runUnderWrapper(t, shell, repo, repo, script)

		assertSamePath(t, "pwd after a failed multi-target rm", pwd, repo)
		mustNotExist(t, curPath)
		mustExist(t, brokenPath)
		if !strings.Contains(stderr, "wrap-broken") {
			t.Errorf("expected the failure to name the broken worktree\n--- got ---\n%s", stderr)
		}
	})
}

// The mirror of the case above: the current worktree is queued behind a
// target that fails, so the run never reaches it. Its directory is still
// there, and teleporting the shell to the main repo would be wrong.
func TestWrapper_RmDoesNotBounceWhenTheCurrentWorktreeSurvives(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		repo := newRepo(t)
		for _, n := range []string{"wrap-block", "wrap-stay"} {
			if r := runWT(t, repo, "new", n, "--non-interactive", "--no-copy"); r.ExitCode != 0 {
				t.Fatalf("setup %s: %s", n, r.Stderr)
			}
		}
		blockedPath := filepath.Join(repo, ".worktrees", "wrap-block")
		stayPath := filepath.Join(repo, ".worktrees", "wrap-stay")
		breakWorktreeLink(t, blockedPath)

		// wrap-block is first, so it fails before wrap-stay is ever touched.
		script := "cd '" + stayPath + "'\ngwt rm wrap-block wrap-stay || true\npwd\n"
		pwd, _ := runUnderWrapper(t, shell, repo, repo, script)

		assertSamePath(t, "pwd after a rm that never reached the current worktree", pwd, stayPath)
		mustExist(t, stayPath)
		mustExist(t, blockedPath)
	})
}

// breakWorktreeLink points a worktree's .git file somewhere bogus, so it
// fails at removal time rather than in any preflight: the dirty scan can't
// read it and treats it as clean, the fast path can't establish safety and
// defers to git, and `git worktree remove` refuses a .git that doesn't
// point back at its admin dir. (A lock used to serve here, but rm now
// refuses locked targets before deleting anything.)
func breakWorktreeLink(t *testing.T, path string) {
	t.Helper()
	mustWrite(t, filepath.Join(path, ".git"), "gitdir: "+filepath.Join(t.TempDir(), "nowhere")+"\n")
}
