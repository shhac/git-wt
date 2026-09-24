package cli

import (
	"fmt"
	"os"

	"github.com/shhac/git-wt/internal/debug"
	"github.com/shhac/git-wt/internal/picker"
	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/wt"
)

// pickWorktree opens an interactive single-select over wts. Returns
// (nil, nil) on cancel (ESC, Ctrl-C, q).
func pickWorktree(title string, wts []wt.Worktree, mainRoot, treesDir string) (_ *wt.Worktree, err error) {
	end := debug.Op("pick.one", fmt.Sprintf("%d-row(s)", len(wts)))
	defer func() { end(err) }()

	rows := buildWorktreeRows(wts, mainRoot, treesDir)
	value, ok, err := picker.SelectOne(title, rows)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	for i := range wts {
		if wts[i].Path == value {
			return &wts[i], nil
		}
	}
	return nil, nil
}

// pickWorktreesTo is the no-argument path of the commands that act on
// worktrees named on the command line (rm, lock, unlock): say noneLeft when
// nothing qualifies, insist on names without a terminal, otherwise open a
// multi-select over wts titled with verb. Returns nil when there is nothing
// to pick, or on cancel or an empty selection.
func pickWorktreesTo(verb, noneLeft string, wts []wt.Worktree, mainRoot, treesDir string) (_ []wt.Worktree, err error) {
	if len(wts) == 0 {
		fmt.Fprintln(os.Stderr, noneLeft)
		return nil, nil
	}
	if !interactive() {
		return nil, fmt.Errorf("no branches specified (run with branch args in non-interactive mode)")
	}

	end := debug.Op("pick.many", fmt.Sprintf("%d-row(s)", len(wts)))
	defer func() { end(err) }()

	title := "Select worktrees to " + verb + " (space to toggle, enter to continue, esc to cancel)"
	values, ok, err := picker.SelectMany(title, buildWorktreeRows(wts, mainRoot, treesDir))
	if err != nil {
		return nil, err
	}
	if !ok || len(values) == 0 {
		return nil, nil
	}
	out := make([]wt.Worktree, 0, len(values))
	for _, v := range values {
		for i := range wts {
			if wts[i].Path == v {
				out = append(out, wts[i])
				break
			}
		}
	}
	return out, nil
}

// buildWorktreeRows renders the picker rows for a slice of worktrees.
// Columns: branch (cyan) | location (dim) | mtime (dim).
func buildWorktreeRows(wts []wt.Worktree, mainRoot, treesDir string) []picker.Row {
	branchW, parentW := columnWidths(wts, mainRoot, treesDir)
	rows := make([]picker.Row, len(wts))
	for i, t := range wts {
		rows[i] = picker.Row{
			Display: formatPickerRow(t, mainRoot, treesDir, branchW, parentW),
			Value:   t.Path,
		}
	}
	return rows
}

// formatPickerRow lays out one worktree row as "branch  location  mtime"
// with aligned columns, plus the lock tag and reason when the worktree is locked. The "location" column is the worktree's DisplayPath.
func formatPickerRow(t wt.Worktree, mainRoot, treesDir string, branchW, parentW int) string {
	branch := padRight(t.Display(), branchW)
	loc := padRight(t.DisplayPath(mainRoot, treesDir), parentW)
	mtime := ui.HumanSince(t.ModTime)
	return withLockTag(ui.Branch(branch)+"  "+ui.Dim(loc)+"  "+ui.Dim(mtime), t, ui.Locked, ui.Dim)
}
