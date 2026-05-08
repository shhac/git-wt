package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/wt"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all worktrees",
	Long:    "Show every worktree with its branch, parent directory, and last-modified time. The current worktree is marked with `*`.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		wts, err := wt.List(ctx, "")
		if err != nil {
			return err
		}
		wt.SortByModTime(wts)
		cur := wt.Current(wts, mustWD())
		printList(os.Stdout, wts, cur)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

// printList renders the table to w using the user's --plain preference.
// Columns: marker | branch | parent dir | mtime
func printList(w *os.File, wts []wt.Worktree, cur *wt.Worktree) {
	if len(wts) == 0 {
		fmt.Fprintln(w, "no worktrees")
		return
	}

	branchW, parentW := 0, 0
	for _, t := range wts {
		if n := lipgloss.Width(t.Display()); n > branchW {
			branchW = n
		}
		if n := lipgloss.Width(t.ParentDirName()); n > parentW {
			parentW = n
		}
	}

	for i := range wts {
		t := &wts[i]
		marker := "  "
		if cur != nil && t.Path == cur.Path {
			marker = "* "
		}
		branch := padRight(t.Display(), branchW)
		parent := padRight(t.ParentDirName(), parentW)
		mtime := ui.HumanSince(t.ModTime)
		row := fmt.Sprintf("%s%s  %s  %s", marker, branch, ui.Dim(parent), ui.Dim(mtime))
		if cur != nil && t.Path == cur.Path {
			row = ui.Current(row)
		}
		fmt.Fprintln(w, row)
	}
}

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

// mustWD returns the current working directory, falling back to "." on error.
func mustWD() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
