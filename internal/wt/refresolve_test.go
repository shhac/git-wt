package wt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hermeticGitEnv returns an environment that hides developer-side gitconfig
// so test behaviour is reproducible.
func hermeticGitEnv() []string {
	return append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
	)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = hermeticGitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// resolverRepo returns a fresh repo with one commit. Caller is free to add
// branches, remotes, etc.
func resolverRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "-q", "-b", "main", ".")
	runGit(t, repo, "commit", "-q", "--allow-empty", "-m", "init")
	return repo
}

// resolverRepoWithOrigin returns a repo plus a bare origin remote, with main
// pushed so origin/main exists as a remote-tracking ref locally.
func resolverRepoWithOrigin(t *testing.T) string {
	t.Helper()
	repo := resolverRepo(t)
	origin := filepath.Join(filepath.Dir(repo), "origin.git")
	runGit(t, "", "init", "-q", "--bare", origin)
	runGit(t, repo, "remote", "add", "origin", origin)
	runGit(t, repo, "push", "-q", "-u", "origin", "main")
	return repo
}

func TestResolveAddRef_LocalBranch(t *testing.T) {
	repo := resolverRepo(t)
	runGit(t, repo, "branch", "feat-local")

	res, err := ResolveAddRef(context.Background(), repo, "feat-local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != AddRefLocal {
		t.Errorf("Kind = %v, want AddRefLocal", res.Kind)
	}
	if res.SourceRef != "feat-local" {
		t.Errorf("SourceRef = %q, want feat-local", res.SourceRef)
	}
	if res.LocalName != "feat-local" {
		t.Errorf("LocalName = %q, want feat-local", res.LocalName)
	}
}

func TestResolveAddRef_LocalSlashedBranch(t *testing.T) {
	// Slash-bearing local branches like `paul/auth-bug` must resolve as
	// local when no remote named `paul` exists.
	repo := resolverRepo(t)
	runGit(t, repo, "branch", "paul/auth-bug")

	res, err := ResolveAddRef(context.Background(), repo, "paul/auth-bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != AddRefLocal {
		t.Errorf("Kind = %v, want AddRefLocal (no remote named `paul`)", res.Kind)
	}
	if res.SourceRef != "paul/auth-bug" || res.LocalName != "paul/auth-bug" {
		t.Errorf("got SourceRef=%q LocalName=%q, want both paul/auth-bug", res.SourceRef, res.LocalName)
	}
}

func TestResolveAddRef_RemoteBranchStripsPrefixForLocalName(t *testing.T) {
	repo := resolverRepoWithOrigin(t)
	// Push a branch then delete the local copy so only origin/feature-x exists.
	runGit(t, repo, "checkout", "-q", "-b", "feature-x")
	runGit(t, repo, "push", "-q", "-u", "origin", "feature-x")
	runGit(t, repo, "checkout", "-q", "main")
	runGit(t, repo, "branch", "-D", "feature-x")

	res, err := ResolveAddRef(context.Background(), repo, "origin/feature-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != AddRefRemote {
		t.Errorf("Kind = %v, want AddRefRemote", res.Kind)
	}
	if res.SourceRef != "origin/feature-x" {
		t.Errorf("SourceRef = %q, want origin/feature-x", res.SourceRef)
	}
	if res.LocalName != "feature-x" {
		t.Errorf("LocalName = %q, want feature-x (remote prefix stripped)", res.LocalName)
	}
}

func TestResolveAddRef_NoSlashFormAlwaysLocal(t *testing.T) {
	// When user types `feature` with no slash, treat as local even if
	// origin/feature also exists — local-no-slash always wins.
	repo := resolverRepoWithOrigin(t)
	runGit(t, repo, "checkout", "-q", "-b", "feature")
	runGit(t, repo, "push", "-q", "-u", "origin", "feature")
	runGit(t, repo, "checkout", "-q", "main")
	// keep local `feature`

	res, err := ResolveAddRef(context.Background(), repo, "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != AddRefLocal {
		t.Errorf("Kind = %v, want AddRefLocal (no-slash form prefers local)", res.Kind)
	}
}

func TestResolveAddRef_RemoteShapeWithMissingRefFallsThroughToLocal(t *testing.T) {
	// `origin/foo` shape, remote `origin` exists, but refs/remotes/origin/foo
	// does NOT. Resolver must NOT mistakenly succeed with the remote form;
	// it should fall through to the local check. With no local `origin/foo`
	// either, this should error.
	repo := resolverRepoWithOrigin(t)

	_, err := ResolveAddRef(context.Background(), repo, "origin/foo")
	if err == nil {
		t.Errorf("expected error: no local branch and no remote ref for `origin/foo`")
	}
}

func TestResolveAddRef_RemoteShapeWithMissingRefMatchesLocalSlashName(t *testing.T) {
	// `origin/foo` shape, remote `origin` exists but no remote ref —
	// yet there IS a local branch literally named `origin/foo` (cursed
	// but legal). Resolver should fall through to local and succeed.
	repo := resolverRepoWithOrigin(t)
	runGit(t, repo, "branch", "origin/foo")

	res, err := ResolveAddRef(context.Background(), repo, "origin/foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != AddRefLocal {
		t.Errorf("Kind = %v, want AddRefLocal (literal local branch wins when remote ref is missing)", res.Kind)
	}
	if res.LocalName != "origin/foo" {
		t.Errorf("LocalName = %q, want origin/foo", res.LocalName)
	}
}

func TestResolveAddRef_NotFound(t *testing.T) {
	repo := resolverRepo(t)
	_, err := ResolveAddRef(context.Background(), repo, "does-not-exist")
	if err == nil {
		t.Errorf("expected error for missing ref")
	}
}

func TestResolveAddRef_EmptyString(t *testing.T) {
	repo := resolverRepo(t)
	_, err := ResolveAddRef(context.Background(), repo, "")
	if err == nil {
		t.Errorf("expected error for empty ref")
	}
}

func TestAddRefResolution_WorktreeAddArgs_Local(t *testing.T) {
	r := &AddRefResolution{Kind: AddRefLocal, SourceRef: "feat", LocalName: "feat"}
	got := r.WorktreeAddArgs("/tmp/wt")
	want := []string{"worktree", "add", "/tmp/wt", "feat"}
	if !sliceEq(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAddRefResolution_WorktreeAddArgs_Remote(t *testing.T) {
	// Pins the DWIM-trap workaround: remote refs MUST get --track -b
	// <LocalName> or `git worktree add origin/feat` creates a detached
	// worktree instead of a tracking branch.
	r := &AddRefResolution{Kind: AddRefRemote, SourceRef: "origin/feat", LocalName: "feat"}
	got := r.WorktreeAddArgs("/tmp/wt")
	want := []string{"worktree", "add", "--track", "-b", "feat", "/tmp/wt", "origin/feat"}
	if !sliceEq(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
