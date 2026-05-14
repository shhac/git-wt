package e2e

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestList_OnlyMain(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "list")
	if res.ExitCode != 0 {
		t.Fatalf("list exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "main") {
		t.Errorf("expected `main` in list output, got: %s", res.Stdout)
	}
	if !strings.HasPrefix(res.Stdout, "* ") {
		t.Errorf("expected current marker `* `, got: %s", res.Stdout)
	}
}

func TestList_LsAlias(t *testing.T) {
	repo := newRepo(t)
	a := runWT(t, repo, "list")
	b := runWT(t, repo, "ls")
	if a.Stdout != b.Stdout {
		t.Errorf("`list` and `ls` produced different output:\n list:\n%s\n ls:\n%s", a.Stdout, b.Stdout)
	}
}

func TestNew_CreatesWorktree(t *testing.T) {
	repo := newRepo(t)
	mustWrite(t, filepath.Join(repo, ".env"), "FOO=1\n")

	res := runWT(t, repo, "new", "feat-a", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}

	// default trees dir is <repo>/.worktrees
	wtPath := filepath.Join(repo, ".worktrees", "feat-a")
	mustExist(t, wtPath)
	mustExist(t, filepath.Join(wtPath, ".env"))

	// path printed on stdout (bare mode)
	if !strings.Contains(res.Stdout, wtPath) {
		t.Errorf("expected stdout to include %s, got: %s", wtPath, res.Stdout)
	}
}

func TestNew_HintsWhenParentDirNotIgnored(t *testing.T) {
	repo := newRepo(t)
	// No .gitignore — default .worktrees/ parent is unignored.
	res := runWT(t, repo, "new", "feat-hint", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stderr, ".worktrees/") || !strings.Contains(res.Stderr, ".gitignore") {
		t.Errorf("expected gitignore hint mentioning `.worktrees/` and `.gitignore`; got:\n%s", res.Stderr)
	}
}

func TestNew_NoHintWhenParentIgnored(t *testing.T) {
	repo := newRepo(t)
	mustWrite(t, filepath.Join(repo, ".gitignore"), ".worktrees/\n")

	res := runWT(t, repo, "new", "feat-noh", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d: %s", res.ExitCode, res.Stderr)
	}
	if strings.Contains(res.Stderr, "not in .gitignore") {
		t.Errorf("expected silence when .worktrees/ is ignored; got:\n%s", res.Stderr)
	}
}

func TestNew_NoCopySkipsConfig(t *testing.T) {
	repo := newRepo(t)
	mustWrite(t, filepath.Join(repo, ".env"), "FOO=1\n")

	res := runWT(t, repo, "new", "feat-b", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d: %s", res.ExitCode, res.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "feat-b")
	mustNotExist(t, filepath.Join(wtPath, ".env"))
}

func TestNew_CopySpecExcludes(t *testing.T) {
	repo := newRepo(t)
	mustWrite(t, filepath.Join(repo, ".env"), "x")
	mustWrite(t, filepath.Join(repo, ".env.production"), "secret")
	mustWrite(t, filepath.Join(repo, ".git-wt-copy-files"), ".env\n.env.*\n!.env.production\n")

	res := runWT(t, repo, "new", "feat-c", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d: %s", res.ExitCode, res.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "feat-c")
	mustExist(t, filepath.Join(wtPath, ".env"))
	mustNotExist(t, filepath.Join(wtPath, ".env.production"))
}

func TestNew_RejectsExistingBranch(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "new", "feat-x", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("first new failed: %s", res.Stderr)
	}
	res = runWT(t, repo, "new", "feat-x", "--non-interactive")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit re-creating existing branch; got %d", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "already exists") {
		t.Errorf("expected `already exists` error, got: %s", res.Stderr)
	}
}

func TestNew_CaseCollisionOnInsensitiveFS(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("case-collision check is gated to macOS/Windows")
	}
	repo := newRepo(t)
	if r := runWT(t, repo, "new", "paul/Foo", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("first new failed: %s", r.Stderr)
	}
	res := runWT(t, repo, "new", "Paul/bar", "--non-interactive", "--no-copy")
	if res.ExitCode == 0 {
		t.Errorf("expected case-collision error, got success")
	}
	if !strings.Contains(res.Stderr, "case-insensitive conflict") {
		t.Errorf("expected `case-insensitive conflict` in stderr, got: %s", res.Stderr)
	}
}

func TestGo_DirectViaFD(t *testing.T) {
	repo := newRepo(t)
	r := runWT(t, repo, "new", "feat-go", "--non-interactive", "--no-copy")
	if r.ExitCode != 0 {
		t.Fatalf("setup new failed: %s", r.Stderr)
	}
	res := runWTFD(t, repo, "go", "feat-go")
	if res.ExitCode != 0 {
		t.Fatalf("go exit %d: %s", res.ExitCode, res.Stderr)
	}
	got := strings.TrimSpace(res.FD3)
	// On macOS /var → /private/var, so git canonicalizes paths. Compare the suffix.
	if !strings.HasSuffix(got, "/demo/.worktrees/feat-go") {
		t.Errorf("fd3 = %q, want path ending in /demo/.worktrees/feat-go", got)
	}
	if got == "" {
		t.Errorf("fd3 was empty (binary did not write to fd 3)")
	}
}

func TestGo_BareModePrintsToStdout(t *testing.T) {
	repo := newRepo(t)
	r := runWT(t, repo, "new", "feat-bare", "--non-interactive", "--no-copy")
	if r.ExitCode != 0 {
		t.Fatalf("setup new failed: %s", r.Stderr)
	}
	res := runWT(t, repo, "go", "feat-bare")
	got := strings.TrimSpace(res.Stdout)
	if !strings.HasSuffix(got, "/demo/.worktrees/feat-bare") {
		t.Errorf("bare-mode stdout = %q, want path ending in /demo/.worktrees/feat-bare", got)
	}
}

func TestGo_MissingBranch(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "go", "nonexistent")
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit, got 0")
	}
	if !strings.Contains(res.Stderr, "no worktree for branch") {
		t.Errorf("expected `no worktree for branch` error, got: %s", res.Stderr)
	}
}

func TestRm_KeepsBranchByDefault(t *testing.T) {
	repo := newRepo(t)
	if r := runWT(t, repo, "new", "rm-keep", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	res := runWT(t, repo, "rm", "rm-keep", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("rm exit %d: %s", res.ExitCode, res.Stderr)
	}
	branches := mustGit(t, repo, "branch", "--list", "--format=%(refname:short)")
	if !strings.Contains(branches, "rm-keep") {
		t.Errorf("expected branch `rm-keep` to still exist (default keep-branch), got: %s", branches)
	}
}

func TestRm_DeleteBranch(t *testing.T) {
	repo := newRepo(t)
	if r := runWT(t, repo, "new", "rm-del", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	res := runWT(t, repo, "rm", "rm-del", "--delete-branch", "--non-interactive")
	if res.ExitCode != 0 {
		t.Fatalf("rm exit %d: %s", res.ExitCode, res.Stderr)
	}
	branches := mustGit(t, repo, "branch", "--list", "--format=%(refname:short)")
	if strings.Contains(branches, "rm-del") {
		t.Errorf("expected branch `rm-del` to be deleted, got: %s", branches)
	}
}

func TestRm_RefusesMain(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "rm", "main", "--non-interactive")
	if res.ExitCode == 0 {
		t.Errorf("expected refusal to rm main, got success")
	}
	if !strings.Contains(res.Stderr, "main") {
		t.Errorf("expected `main` in error, got: %s", res.Stderr)
	}
}

func TestClean_Orphaned(t *testing.T) {
	repo := newRepo(t)
	if r := runWT(t, repo, "new", "orphan", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	// Detach worktree from its branch by removing the ref directly.
	mustGit(t, repo, "checkout", "-q", "main")
	if err := orphanBranch(repo, "orphan"); err != nil {
		t.Fatalf("orphan: %v", err)
	}

	res := runWT(t, repo, "clean", "--non-interactive", "--orphaned-only", "--no-fetch")
	if res.ExitCode != 0 {
		t.Fatalf("clean exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stderr, "branch deleted") {
		t.Errorf("expected `branch deleted` reason in stderr, got: %s", res.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "orphan")
	mustNotExist(t, wtPath)
}

func TestClean_UpstreamGone(t *testing.T) {
	repo, origin := newRepoWithRemote(t)
	_ = origin
	mustGit(t, repo, "checkout", "-q", "-b", "doomed")
	mustGit(t, repo, "push", "-q", "-u", "origin", "doomed")
	mustGit(t, repo, "checkout", "-q", "main")

	if r := runWT(t, repo, "new", "doomed-tree", "--from", "doomed", "--non-interactive", "--no-copy"); r.ExitCode != 0 {
		t.Fatalf("setup: %s", r.Stderr)
	}
	mustGit(t, repo, "branch", "--set-upstream-to=origin/doomed", "doomed-tree")
	mustGit(t, repo, "push", "origin", "--delete", "doomed")

	res := runWT(t, repo, "clean", "--non-interactive", "--upstream-gone-only")
	if res.ExitCode != 0 {
		t.Fatalf("clean exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stderr, "upstream gone") {
		t.Errorf("expected `upstream gone` reason in stderr, got: %s", res.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "doomed-tree")
	mustNotExist(t, wtPath)
}

func TestAlias_GeneratesValidShell(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "alias", "mygwt")
	if res.ExitCode != 0 {
		t.Fatalf("alias exit %d: %s", res.ExitCode, res.Stderr)
	}
	// runWT pins --fd 9, so the wrapper bakes 9 too.
	for _, want := range []string{"mygwt() {", "case \"$_sub\" in", "go|new|add|eject|rm)", "9>&1 1>&2"} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("alias output missing %q\n--- output ---\n%s", want, res.Stdout)
		}
	}
}

// TestAlias_DefaultFD pins the user-facing default fd of 3. It bypasses
// runWT (which prepends --fd 9 for test-isolation reasons) and calls the
// binary with no --fd override, exercising the default that ships in
// real-world `eval "$(git-wt alias gwt)"` installs.
func TestAlias_DefaultFD(t *testing.T) {
	repo := newRepo(t)
	res := doRun(t, repo, false, "alias", "mygwt")
	if res.ExitCode != 0 {
		t.Fatalf("alias exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "3>&1 1>&2") {
		t.Errorf("alias default fd should be 3, got:\n%s", res.Stdout)
	}
}

// TestAlias_WrapperCdWorksWithLeadingGlobalFlag pins the fix for the case
// where `gwt --debug go <branch>` (any pre-subcommand global flag) was
// falling through to the pass-through branch of the wrapper and skipping
// the cd. The wrapper now walks args to find the real subcommand.
func TestAlias_WrapperCdWorksWithLeadingGlobalFlag(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not on PATH; wrapper-shell test requires bash")
	}

	repo := newRepo(t)
	if res := runWT(t, repo, "new", "feature", "--non-interactive", "--no-copy"); res.ExitCode != 0 {
		t.Fatalf("new: exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}
	wtPath := filepath.Join(repo, ".worktrees", "feature")
	mustExist(t, wtPath)

	// The generated wrapper bakes in os.Executable(), which during the test
	// is the freshly-built binPath — so eval+invoke will hit the test binary.
	aliasRes := runWT(t, repo, "alias", "gwt")
	if aliasRes.ExitCode != 0 {
		t.Fatalf("alias: exit %d, stderr: %s", aliasRes.ExitCode, aliasRes.Stderr)
	}

	// Note: a user-supplied `--fd N` is intentionally out of scope. The wrapper
	// bakes its fd redirect at generation time; overriding the fd via CLI flag
	// would need a matching shell-side redirect the wrapper can't know about.
	cases := []struct {
		name   string
		invoke string
	}{
		// Each of these has a pre-subcommand global flag that previously caused
		// the wrapper's `case "$1"` to miss the subcommand and skip the cd.
		{"leading --debug", `gwt --debug go feature`},
		{"leading --plain", `gwt --plain go feature`},
		{"leading -n", `gwt -n go feature`},
		// Trailing flag — always worked, regression guard.
		{"trailing --debug", `gwt go feature --debug`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			script := aliasRes.Stdout + "\n" + tc.invoke + "\npwd\n"
			cmd := exec.Command("bash", "-c", script)
			cmd.Dir = repo
			cmd.Env = hermeticEnv()
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("bash run: %v\nstderr: %s", err, stderr.String())
			}

			gotPwd := strings.TrimSpace(stdout.String())
			got, _ := filepath.EvalSymlinks(gotPwd)
			want, _ := filepath.EvalSymlinks(wtPath)
			if got == "" || want == "" {
				t.Fatalf("evalSymlinks failed: got=%q want=%q (gotPwd=%q)", got, want, gotPwd)
			}
			if got != want {
				t.Errorf("after %q:\n  pwd  = %q\n  want = %q\nstderr:\n%s",
					tc.invoke, got, want, stderr.String())
			}
		})
	}
}

func TestVersion(t *testing.T) {
	res := runWT(t, "", "--version")
	if res.ExitCode != 0 {
		t.Fatalf("--version exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "git-wt version") {
		t.Errorf("expected `git-wt version` prefix, got: %s", res.Stdout)
	}
}

