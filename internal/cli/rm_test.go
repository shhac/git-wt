package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestRmProgressError_SingleTargetUnchanged(t *testing.T) {
	targets := []rmTarget{{Worktree: wt.Worktree{Path: "/p/a", Branch: "a"}}}
	base := errors.New("boom")
	got := rmProgressError(targets, 0, base)
	if got != base {
		t.Errorf("got %v, want the original error untouched", got)
	}
}

func TestRmProgressError_ReportsProgressAndRemainder(t *testing.T) {
	targets := []rmTarget{
		{Worktree: wt.Worktree{Path: "/p/a", Branch: "a"}},
		{Worktree: wt.Worktree{Path: "/p/b", Branch: "b"}},
		{Worktree: wt.Worktree{Path: "/p/c", Branch: "c"}},
	}
	base := errors.New("boom")
	got := rmProgressError(targets, 1, base)
	msg := got.Error()
	for _, want := range []string{"boom", "removed 1 of 3", "not attempted", "c"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q\n--- got ---\n%s", want, msg)
		}
	}
	if !errors.Is(got, base) {
		t.Error("wrapped error should still unwrap to the cause")
	}
}

func TestRmProgressError_LastTargetHasNoRemainder(t *testing.T) {
	targets := []rmTarget{
		{Worktree: wt.Worktree{Path: "/p/a", Branch: "a"}},
		{Worktree: wt.Worktree{Path: "/p/b", Branch: "b"}},
	}
	msg := rmProgressError(targets, 1, errors.New("boom")).Error()
	if strings.Contains(msg, "not attempted") {
		t.Errorf("nothing was left over\n--- got ---\n%s", msg)
	}
	if !strings.Contains(msg, "removed 1 of 2") {
		t.Errorf("missing progress count\n--- got ---\n%s", msg)
	}
}

func TestRmSummary_MarksDirtyTargets(t *testing.T) {
	got := rmSummary([]rmTarget{
		{Worktree: wt.Worktree{Path: "/p/a", Branch: "a"}},
		{Worktree: wt.Worktree{Path: "/p/b", Branch: "b"}, dirty: wt.DirtyStat{Modified: 4, Untracked: 1}, force: true},
	})
	if !strings.Contains(got, "Remove 2 worktree(s):") {
		t.Errorf("missing header\n--- got ---\n%s", got)
	}
	if !strings.Contains(got, "[discards 4 modified, 1 untracked]") {
		t.Errorf("dirty target not called out\n--- got ---\n%s", got)
	}
	if strings.Count(got, "discards") != 1 {
		t.Errorf("clean target should carry no note\n--- got ---\n%s", got)
	}
}

func TestRmSummary_CleanTargetsStayQuiet(t *testing.T) {
	got := rmSummary([]rmTarget{{Worktree: wt.Worktree{Path: "/p/a", Branch: "a"}}})
	if strings.Contains(got, "discards") {
		t.Errorf("unexpected note\n--- got ---\n%s", got)
	}
}

func TestDedupeTargets(t *testing.T) {
	in := []rmTarget{
		{Worktree: wt.Worktree{Path: "/p/a", Branch: "a"}},
		{Worktree: wt.Worktree{Path: "/p/b", Branch: "b"}},
		{Worktree: wt.Worktree{Path: "/p/a", Branch: "a"}},
	}
	got := dedupeTargets(in)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Path != "/p/a" || got[1].Path != "/p/b" {
		t.Errorf("first position not kept: %v, %v", got[0].Path, got[1].Path)
	}
}

func TestDedupeTargets_NoDupesIsUnchanged(t *testing.T) {
	in := []rmTarget{{Worktree: wt.Worktree{Path: "/p/a"}}, {Worktree: wt.Worktree{Path: "/p/b"}}}
	if got := dedupeTargets(in); len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

func TestFindByTreesDirLeaf(t *testing.T) {
	treesDir := filepath.Join("/repo", ".worktrees")
	wts := []wt.Worktree{
		{Path: "/repo", Branch: "main"},
		{Path: filepath.Join(treesDir, "detached")}, // no branch
		{Path: filepath.Join(treesDir, "feat"), Branch: "paul/feat"},
	}
	got := findByTreesDirLeaf(wts, treesDir, "detached")
	if got == nil || got.Path != filepath.Join(treesDir, "detached") {
		t.Fatalf("got %v, want the detached worktree", got)
	}
	if findByTreesDirLeaf(wts, treesDir, "nope") != nil {
		t.Error("unknown leaf should not resolve")
	}
	if findByTreesDirLeaf(wts, "", "detached") != nil {
		t.Error("empty treesDir should not resolve")
	}
}

// The leaf lookup builds a path from user input, so it must not be usable to
// reach a worktree outside the trees dir.
func TestFindByTreesDirLeaf_RefusesEscapingArgs(t *testing.T) {
	treesDir := filepath.Join("/repo", ".worktrees")
	wts := []wt.Worktree{{Path: "/repo", Branch: "main"}}
	for _, arg := range []string{"..", "../..", "/repo"} {
		if got := findByTreesDirLeaf(wts, treesDir, arg); got != nil {
			t.Errorf("findByTreesDirLeaf(%q) = %v, want nil", arg, got.Path)
		}
	}
}

func TestRmTargetLabel(t *testing.T) {
	tests := []struct {
		name string
		in   rmTarget
		want string
	}{
		{"branch", rmTarget{Worktree: wt.Worktree{Path: "/p/a", Branch: "a"}}, "a"},
		{
			"detached carries its path, so two are distinguishable",
			rmTarget{Worktree: wt.Worktree{Path: "/p/d", Detached: true}},
			"(detached) /p/d",
		},
		{
			"orphan",
			rmTarget{Worktree: wt.Worktree{Path: "/p/o"}, orphan: true},
			"/p/o (unregistered leftover)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.label(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// Two detached worktrees must not present as the same line in the prompt
// that precedes deleting them.
func TestRmSummary_DistinguishesTwoDetachedWorktrees(t *testing.T) {
	got := rmSummary([]rmTarget{
		{Worktree: wt.Worktree{Path: "/p/one", Detached: true}},
		{Worktree: wt.Worktree{Path: "/p/two", Detached: true}},
	})
	if !strings.Contains(got, "/p/one") || !strings.Contains(got, "/p/two") {
		t.Errorf("both paths should appear\n--- got ---\n%s", got)
	}
}
