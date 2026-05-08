package cli

import (
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
	if needsBounce(nil, []wt.Worktree{{Path: "/a"}}) {
		t.Errorf("nil current should never bounce")
	}
}

func TestNeedsBounce_CurrentInTargets(t *testing.T) {
	cur := &wt.Worktree{Path: "/p/feat"}
	targets := []wt.Worktree{{Path: "/p/other"}, {Path: "/p/feat"}}
	if !needsBounce(cur, targets) {
		t.Errorf("expected bounce when current is among targets")
	}
}

func TestNeedsBounce_CurrentNotInTargets(t *testing.T) {
	cur := &wt.Worktree{Path: "/p/feat"}
	targets := []wt.Worktree{{Path: "/p/other"}}
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
	got, err := resolveRmFromArgs(wts, repo, []string{"feat", "other"})
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
	_, err := resolveRmFromArgs(wts, repo, []string{"main"})
	if err == nil {
		t.Errorf("expected error rejecting main worktree")
	}
}

func TestResolveRmFromArgs_UnknownBranch(t *testing.T) {
	wts := []wt.Worktree{{Path: "/p/main", Branch: "main"}}
	repo := &wt.RepoInfo{MainRoot: "/p/main"}
	_, err := resolveRmFromArgs(wts, repo, []string{"nonexistent"})
	if err == nil {
		t.Errorf("expected error for unknown branch")
	}
}
