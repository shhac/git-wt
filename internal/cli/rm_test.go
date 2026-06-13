package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shhac/git-wt/internal/wt"
)

func TestRmOptions_NoFlags(t *testing.T) {
	opts := rmOptions(false, false)
	if len(opts) != 3 {
		t.Fatalf("len = %d, want 3 (keep + delete + cancel)", len(opts))
	}
	if opts[0].Value != rmTreeOnly || opts[1].Value != rmTreeAndBranch || opts[2].Value != rmCancel {
		t.Errorf("unexpected order: %+v", opts)
	}
}

func TestRmOptions_KeepBranch(t *testing.T) {
	opts := rmOptions(true, false)
	if len(opts) != 2 {
		t.Fatalf("len = %d, want 2 (keep + cancel)", len(opts))
	}
	if opts[0].Value != rmTreeOnly || opts[1].Value != rmCancel {
		t.Errorf("expected [keep, cancel], got %+v", opts)
	}
}

func TestRmOptions_DeleteBranch(t *testing.T) {
	opts := rmOptions(false, true)
	if len(opts) != 2 {
		t.Fatalf("len = %d, want 2 (delete + cancel)", len(opts))
	}
	if opts[0].Value != rmTreeAndBranch || opts[1].Value != rmCancel {
		t.Errorf("expected [delete, cancel], got %+v", opts)
	}
}

func TestNeedsBounce_NilCurrent(t *testing.T) {
	if needsBounce(nil, toRmTargets([]wt.Worktree{{Path: "/a"}})) {
		t.Errorf("nil current should never bounce")
	}
}

func TestNeedsBounce_CurrentInTargets(t *testing.T) {
	cur := &wt.Worktree{Path: "/p/feat"}
	targets := toRmTargets([]wt.Worktree{{Path: "/p/other"}, {Path: "/p/feat"}})
	if !needsBounce(cur, targets) {
		t.Errorf("expected bounce when current is among targets")
	}
}

func TestNeedsBounce_CurrentNotInTargets(t *testing.T) {
	cur := &wt.Worktree{Path: "/p/feat"}
	targets := toRmTargets([]wt.Worktree{{Path: "/p/other"}})
	if needsBounce(cur, targets) {
		t.Errorf("expected no bounce when current is unaffected")
	}
}

func TestResolveRmFromArgs_Success(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/p/main", Branch: "main"},
		{Path: "/p/feat", Branch: "feat"},
		{Path: "/p/other", Branch: "other"},
	}
	repo := &wt.RepoInfo{MainRoot: "/p/main"}
	got, err := resolveRmFromArgs(wts, repo, []string{"feat", "other"}, "/p/main/.worktrees", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

func TestResolveRmFromArgs_RejectsMain(t *testing.T) {
	wts := []wt.Worktree{{Path: "/p/main", Branch: "main"}}
	repo := &wt.RepoInfo{MainRoot: "/p/main"}
	_, err := resolveRmFromArgs(wts, repo, []string{"main"}, "/p/main/.worktrees", false)
	if err == nil {
		t.Errorf("expected error rejecting main worktree")
	}
}

func TestResolveRmFromArgs_UnknownBranch(t *testing.T) {
	wts := []wt.Worktree{{Path: "/p/main", Branch: "main"}}
	repo := &wt.RepoInfo{MainRoot: "/p/main"}
	_, err := resolveRmFromArgs(wts, repo, []string{"nonexistent"}, "/p/main/.worktrees", false)
	if err == nil {
		t.Errorf("expected error for unknown branch")
	}
}

func TestOrphanRmTarget_ResolvesLeftoverDir(t *testing.T) {
	treesDir := t.TempDir()
	leftover := filepath.Join(treesDir, "ghost")
	if err := os.MkdirAll(leftover, 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := orphanRmTarget(nil, treesDir, "ghost")
	if !ok {
		t.Fatalf("expected leftover dir to resolve")
	}
	if !got.orphan || got.Path != leftover {
		t.Errorf("got %+v, want orphan target at %s", got, leftover)
	}
}

func TestOrphanRmTarget_SkipsRegisteredWorktree(t *testing.T) {
	treesDir := t.TempDir()
	p := filepath.Join(treesDir, "live")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	wts := []wt.Worktree{{Path: p, Branch: "other-name"}}
	if _, ok := orphanRmTarget(wts, treesDir, "live"); ok {
		t.Errorf("registered worktree must not resolve as an orphan")
	}
}

func TestOrphanRmTarget_RefusesEscape(t *testing.T) {
	treesDir := t.TempDir()
	for _, arg := range []string{"../outside", "/etc", ".."} {
		if _, ok := orphanRmTarget(nil, treesDir, arg); ok {
			t.Errorf("arg %q must not escape the trees dir", arg)
		}
	}
}

func TestOrphanRmTarget_MissingDir(t *testing.T) {
	if _, ok := orphanRmTarget(nil, t.TempDir(), "nope"); ok {
		t.Errorf("nonexistent dir must not resolve")
	}
}
