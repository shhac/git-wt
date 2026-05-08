package cli

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/wt"
)

// pickerTheme returns the huh theme for interactive prompts. ThemeBase is
// used in --plain mode (no color); ThemeCharm otherwise.
func pickerTheme() *huh.Theme {
	if ui.Plain {
		return huh.ThemeBase()
	}
	return huh.ThemeCharm()
}

// silentIfAborted maps huh.ErrUserAborted (ESC / Ctrl-C) to nil so callers
// can distinguish "user cancelled, exit cleanly" from "real error".
func silentIfAborted(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return nil
	}
	return err
}

// pickWorktree opens an interactive single-select over wts and returns the
// chosen entry, or (nil, nil) if the user cancelled (ESC / Ctrl-C).
// Caller is responsible for ensuring wts is non-empty.
func pickWorktree(title string, wts []wt.Worktree, mainRoot, treesDir string) (*wt.Worktree, error) {
	branchW, parentW := columnWidths(wts, mainRoot, treesDir)
	options := make([]huh.Option[string], len(wts))
	for i, t := range wts {
		options[i] = huh.NewOption(formatPickerRow(t, mainRoot, treesDir, branchW, parentW), t.Path)
	}

	var pickedPath string
	err := huh.NewSelect[string]().
		Title(title).
		Options(options...).
		Value(&pickedPath).
		WithTheme(pickerTheme()).
		Run()
	if err := silentIfAborted(err); err != nil {
		return nil, err
	}
	if pickedPath == "" {
		return nil, nil // user cancelled
	}
	for i := range wts {
		if wts[i].Path == pickedPath {
			return &wts[i], nil
		}
	}
	return nil, fmt.Errorf("internal: picked worktree not in list")
}

// pickWorktreesToRemove opens an interactive multi-select for the rm command.
// Returns the chosen worktrees in display order; an empty selection (user
// cancelled or pressed enter without toggling) returns nil.
func pickWorktreesToRemove(wts []wt.Worktree, mainRoot, treesDir string) ([]wt.Worktree, error) {
	branchW, parentW := columnWidths(wts, mainRoot, treesDir)
	options := make([]huh.Option[string], len(wts))
	for i, t := range wts {
		options[i] = huh.NewOption(formatPickerRow(t, mainRoot, treesDir, branchW, parentW), t.Path)
	}

	var picked []string
	err := huh.NewMultiSelect[string]().
		Title("Select worktrees to remove (space to toggle, enter to continue)").
		Options(options...).
		Value(&picked).
		WithTheme(pickerTheme()).
		Run()
	if err := silentIfAborted(err); err != nil {
		return nil, err
	}
	if len(picked) == 0 {
		return nil, nil
	}
	out := make([]wt.Worktree, 0, len(picked))
	for _, p := range picked {
		for i := range wts {
			if wts[i].Path == p {
				out = append(out, wts[i])
				break
			}
		}
	}
	return out, nil
}

// formatPickerRow lays out one worktree row as "branch  location  mtime"
// with aligned columns. Used by pickers and by `list`. The "location" column
// is the worktree's DisplayPath (see Worktree.DisplayPath for rules).
func formatPickerRow(t wt.Worktree, mainRoot, treesDir string, branchW, parentW int) string {
	branch := padRight(t.Display(), branchW)
	loc := padRight(t.DisplayPath(mainRoot, treesDir), parentW)
	mtime := ui.HumanSince(t.ModTime)
	return fmt.Sprintf("%s  %s  %s", ui.Branch(branch), ui.Dim(loc), ui.Dim(mtime))
}

// columnWidths returns the maximum widths of the branch and location columns
// across wts. lipgloss.Width is used so ANSI escape codes don't inflate the
// measurement.
func columnWidths(wts []wt.Worktree, mainRoot, treesDir string) (branchW, parentW int) {
	for _, t := range wts {
		if n := lipgloss.Width(t.Display()); n > branchW {
			branchW = n
		}
		if n := lipgloss.Width(t.DisplayPath(mainRoot, treesDir)); n > parentW {
			parentW = n
		}
	}
	return
}

// padRight pads s with spaces on the right so its visible width equals width.
// No-op if s is already wider.
func padRight(s string, width int) string {
	pad := width - lipgloss.Width(s)
	if pad <= 0 {
		return s
	}
	return s + spaces(pad)
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, n)
	for i := range out {
		out[i] = ' '
	}
	return string(out)
}
