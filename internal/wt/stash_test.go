package wt

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindStashRefBySHA_Found(t *testing.T) {
	out := "abc123 stash@{0}\ndef456 stash@{1}\n"
	ref, ok := findStashRefBySHA(out, "def456")
	if !ok {
		t.Fatalf("expected to find def456")
	}
	if ref != "stash@{1}" {
		t.Errorf("ref = %q, want stash@{1}", ref)
	}
}

func TestFindStashRefBySHA_NotFound(t *testing.T) {
	if _, ok := findStashRefBySHA("abc123 stash@{0}\n", "missing"); ok {
		t.Errorf("expected not-found for missing SHA")
	}
}

func TestFindStashRefBySHA_EmptyInput(t *testing.T) {
	if _, ok := findStashRefBySHA("", "abc123"); ok {
		t.Errorf("expected not-found for empty input")
	}
}

func TestFindStashRefBySHA_SkipsMalformedLines(t *testing.T) {
	// A malformed line (no space) should be skipped without aborting.
	out := "malformed\nabc123 stash@{0}\n"
	ref, ok := findStashRefBySHA(out, "abc123")
	if !ok || ref != "stash@{0}" {
		t.Errorf("got ref=%q ok=%v, want stash@{0} true", ref, ok)
	}
}

func TestShortStashRef(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"abcdef0123456789", "abcdef01"},
		{"short", "short"},       // shorter than 8 → unchanged
		{"", ""},                 // empty → unchanged
		{"12345678", "12345678"}, // exactly 8 → unchanged
	}
	for _, c := range cases {
		got := ShortStashRef(c.in)
		if got != c.want {
			t.Errorf("ShortStashRef(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestStashPushApplyDrop exercises the end-to-end roundtrip through real
// git. Pins the by-SHA capture-then-apply-then-drop contract.
func TestStashPushApplyDrop(t *testing.T) {
	repo := resolverRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "tracked.txt")
	runGit(t, repo, "commit", "-q", "-m", "v1")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("v2-dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	sha, err := stashPushIn(ctx, repo, "test stash")
	if err != nil {
		t.Fatalf("StashPush in %s: %v", repo, err)
	}
	if sha == "" {
		t.Fatalf("expected non-empty SHA")
	}

	outcome, err := stashApplyIn(ctx, repo, sha)
	if err != nil {
		t.Fatalf("StashApply: %v", err)
	}
	if outcome != StashApplied {
		t.Errorf("outcome = %v, want StashApplied", outcome)
	}

	// File restored to dirty content.
	got, err := os.ReadFile(filepath.Join(repo, "tracked.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != "v2-dirty" {
		t.Errorf("tracked.txt = %q, want v2-dirty", string(got))
	}
}

// stashPushIn / stashApplyIn run StashPush / StashApply with cwd set to
// dir, by chdir'ing the test process. Both restore cwd on exit.
func stashPushIn(ctx context.Context, dir, msg string) (string, error) {
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(dir); err != nil {
		return "", err
	}
	return StashPush(ctx, msg)
}

func stashApplyIn(ctx context.Context, dir, sha string) (StashApplyOutcome, error) {
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(dir); err != nil {
		return StashApplyFailed, err
	}
	// Empty dir → runs in cwd, which is what we just chdir'd to.
	return StashApply(ctx, "", sha)
}
