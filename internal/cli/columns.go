package cli

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/shhac/git-wt/internal/wt"
)

// Column-layout helpers used by both the picker (picker.go) and the list
// renderer (list.go). They live here rather than in either consumer because
// they're general formatting primitives, not picker-specific.

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
// No-op if s is already wider. Visible width is measured via lipgloss.Width
// so embedded ANSI escapes don't inflate the count.
func padRight(s string, width int) string {
	pad := width - lipgloss.Width(s)
	if pad <= 0 {
		return s
	}
	return s + spaces(pad)
}

// spaces returns a string of n spaces. Returns "" when n is non-positive.
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
