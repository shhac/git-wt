package cli

import (
	"strings"
	"testing"

	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/wt"
)

func TestPrintList_Empty(t *testing.T) {
	var buf strings.Builder
	printList(&buf, nil, nil)
	got := buf.String()
	if !strings.Contains(got, "no worktrees") {
		t.Errorf("expected `no worktrees`, got %q", got)
	}
}

func TestPrintList_NoCurrentMarker(t *testing.T) {
	ui.Plain = true
	defer func() { ui.Plain = false }()

	var buf strings.Builder
	wts := []wt.Worktree{
		{Path: "/p/a", Branch: "a"},
		{Path: "/p/b", Branch: "b"},
	}
	printList(&buf, wts, nil)
	out := buf.String()
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.HasPrefix(line, "* ") {
			t.Errorf("no row should have a marker when cur is nil; got %q", line)
		}
	}
}

func TestPrintList_MarkerOnCurrentRow(t *testing.T) {
	ui.Plain = true
	defer func() { ui.Plain = false }()

	wts := []wt.Worktree{
		{Path: "/p/a", Branch: "a"},
		{Path: "/p/b", Branch: "b"},
		{Path: "/p/c", Branch: "c"},
	}
	cur := &wts[1]

	var buf strings.Builder
	printList(&buf, wts, cur)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 rows, got %d:\n%s", len(lines), buf.String())
	}
	if strings.HasPrefix(lines[0], "* ") || strings.HasPrefix(lines[2], "* ") {
		t.Errorf("only the current row should be marked; got:\n%s", buf.String())
	}
	if !strings.HasPrefix(lines[1], "* ") {
		t.Errorf("expected current row (index 1) to start with `* `; got %q", lines[1])
	}
}

func TestPrintList_ColumnAlignment(t *testing.T) {
	ui.Plain = true
	defer func() { ui.Plain = false }()

	// Branches of varying widths. With aligned columns, the parent-dir
	// column should start at the same column index across rows.
	wts := []wt.Worktree{
		{Path: "/repo-trees/short", Branch: "short"},
		{Path: "/repo-trees/much-longer-name", Branch: "much-longer-name"},
	}
	var buf strings.Builder
	printList(&buf, wts, nil)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(lines))
	}
	idx := func(line string) int { return strings.Index(line, "repo-trees") }
	if idx(lines[0]) != idx(lines[1]) {
		t.Errorf("parent-dir column not aligned:\n  row0=%q\n  row1=%q", lines[0], lines[1])
	}
}

func TestPrintCleanTargets(t *testing.T) {
	var buf strings.Builder
	targets := []taggedTarget{
		{wt: wt.Worktree{Path: "/p/a", Branch: "a"}, reason: "branch deleted"},
		{wt: wt.Worktree{Path: "/p/b", Branch: "b"}, reason: "upstream gone"},
	}
	printCleanTargets(&buf, targets)
	got := buf.String()
	for _, want := range []string{"worktrees to remove:", "[branch deleted]", "[upstream gone]", "/p/a", "/p/b"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n--- got ---\n%s", want, got)
		}
	}
}
