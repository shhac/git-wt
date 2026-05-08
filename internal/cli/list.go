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
		repo, err := wt.Inspect(ctx, "")
		if err != nil {
			return err
		}
		wts, err := wt.List(ctx, "")
		if err != nil {
			return err
		}
		wt.SortByModTime(wts)
		cur := wt.Current(wts, mustWD())
		printList(os.Stdout, wts, cur, repo.MainRoot, wt.TreesDirFor(repo.MainRoot))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

// printList renders the table to w using the user's --plain preference.
// Columns: marker | branch | location | mtime. The "location" column follows
// Worktree.DisplayPath rules (basename for main, # prefix when inside the
// trees dir, rel-to-repo when inside the repo, absolute when outside).
func printList(w io.Writer, wts []wt.Worktree, cur *wt.Worktree, mainRoot, treesDir string) {
	if len(wts) == 0 {
		fmt.Fprintln(w, "no worktrees")
		return
	}

	branchW, parentW := columnWidths(wts, mainRoot, treesDir)
	for i := range wts {
		t := &wts[i]
		marker := "  "
		if cur != nil && t.Path == cur.Path {
			marker = "* "
		}
		branch := padRight(t.Display(), branchW)
		loc := padRight(t.DisplayPath(mainRoot, treesDir), parentW)
		mtime := ui.HumanSince(t.ModTime)
		// Current row gets the current/green style applied to the whole row so
		// it stays visually consistent. Other rows get cyan branch + dim loc/mtime.
		var row string
		if cur != nil && t.Path == cur.Path {
			row = ui.Current(fmt.Sprintf("%s%s  %s  %s", marker, branch, loc, mtime))
		} else {
			row = fmt.Sprintf("%s%s  %s  %s", marker, ui.Branch(branch), ui.Dim(loc), ui.Dim(mtime))
		}
		fmt.Fprintln(w, row)
	}
}

// mustWD returns the current working directory, falling back to "." on error.
func mustWD() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
