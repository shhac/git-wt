package cli

import (
	"strings"
	"testing"

	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/wt"
)

func TestSpaces(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, ""},
		{-1, ""},
		{1, " "},
		{4, "    "},
	}
	for _, c := range cases {
		if got := spaces(c.in); got != c.want {
			t.Errorf("spaces(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPadRight(t *testing.T) {
	cases := []struct {
		s     string
		width int
		want  string
	}{
		{"abc", 5, "abc  "},
		{"abc", 3, "abc"},     // exact width — no-op
		{"abcde", 3, "abcde"}, // already wider — no-op
		{"", 3, "   "},
		{"x", 0, "x"},
	}
	for _, c := range cases {
		if got := padRight(c.s, c.width); got != c.want {
			t.Errorf("padRight(%q, %d) = %q, want %q", c.s, c.width, got, c.want)
		}
	}
}

func TestPadRight_IgnoresANSI(t *testing.T) {
	// padRight uses lipgloss.Width to measure visible width, so the cyan
	// styling around "abc" must NOT inflate the count and pull padding away.
	ui.Plain = false
	defer func() { ui.Plain = false }()
	cyan := ui.Branch("abc")
	got := padRight(cyan, 5)
	// Visible width of "abc" is 3; should pad with 2 spaces.
	if !strings.HasSuffix(got, "  ") {
		t.Errorf("expected trailing 2 spaces past ANSI; got %q", got)
	}
}

func TestColumnWidths(t *testing.T) {
	wts := []wt.Worktree{
		{Path: "/repo", Branch: "main"},                       // branch=4, loc="repo" (parent)
		{Path: "/repo/.worktrees/feat-a", Branch: "feat-a"},         // branch=6, loc="#feat-a"=7
		{Path: "/repo/.conductor/abuja", Branch: "long-name"}, // branch=9, loc=".conductor/abuja"=16
	}
	branchW, parentW := columnWidths(wts, "/repo", "/repo/.worktrees")
	if branchW != 9 {
		t.Errorf("branchW = %d, want 9 (longest is `long-name`)", branchW)
	}
	if parentW != 16 {
		t.Errorf("parentW = %d, want 16 (longest is `.conductor/abuja`)", parentW)
	}
}

func TestColumnWidths_EmptySlice(t *testing.T) {
	branchW, parentW := columnWidths(nil, "/repo", "/repo/.worktrees")
	if branchW != 0 || parentW != 0 {
		t.Errorf("got (%d, %d), want (0, 0) for empty slice", branchW, parentW)
	}
}
