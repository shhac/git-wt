package cli

import (
	"strings"
	"unicode"

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

// maxWidth returns the widest visible string in labels. Companion to
// columnWidths for callers whose rows aren't worktree-shaped, so they can
// reach padRight without measuring with len().
func maxWidth(labels []string) int {
	w := 0
	for _, s := range labels {
		if n := lipgloss.Width(s); n > w {
			w = n
		}
	}
	return w
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

// oneLine collapses whitespace runs to single spaces and drops control
// characters. A lock reason is free text from whoever took the lock; a
// newline or escape sequence in it would break row layout, or the
// tab-separated completion protocol.
func oneLine(s string) string {
	clean := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return strings.Join(strings.Fields(clean), " ")
}

// truncate shortens s to at most width runes, marking the cut with "…".
func truncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width-1]) + "…"
}
