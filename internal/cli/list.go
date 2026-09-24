package cli

import (
	"fmt"
	"io"
	"os"

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
		repo, wts, cur, err := loadRepoAndWorktrees(ctx)
		if err != nil {
			return err
		}
		printList(os.Stdout, wts, cur, repo.MainRoot, wt.TreesDirFor(repo.MainRoot))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

// printList renders the table to w using the user's --plain preference.
// Columns: marker | branch | location | mtime, then lock age and reason on
// locked rows. The "location" column follows Worktree.DisplayPath rules
// (basename for main, # prefix when inside the trees dir, rel-to-repo when
// inside the repo, absolute when outside).
func printList(w io.Writer, wts []wt.Worktree, cur *wt.Worktree, mainRoot, treesDir string) {
	if len(wts) == 0 {
		_, _ = fmt.Fprintln(w, "no worktrees")
		return
	}

	branchW, parentW := columnWidths(wts, mainRoot, treesDir)
	for _, t := range wts {
		if cur == nil || t.Path != cur.Path {
			_, _ = fmt.Fprintln(w, "  "+formatPickerRow(t, mainRoot, treesDir, branchW, parentW))
			continue
		}
		// The current row takes the current/green style across the whole
		// row, so its columns go in unstyled.
		branch := padRight(t.Display(), branchW)
		loc := padRight(t.DisplayPath(mainRoot, treesDir), parentW)
		row := fmt.Sprintf("* %s  %s  %s", branch, loc, ui.HumanSince(t.ModTime))
		_, _ = fmt.Fprintln(w, ui.Current(withLockTag(row, t, noStyle, noStyle)))
	}
}

func noStyle(s string) string { return s }

// mustWD returns the current working directory, falling back to "." on error.
func mustWD() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
