package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/wt"
)

var goCmd = &cobra.Command{
	Use:   "go [branch]",
	Short: "Navigate to a worktree",
	Long: "Navigate to a worktree.\n\n" +
		"With a branch argument, jumps directly. Without, opens an interactive\n" +
		"picker over the other worktrees (the current worktree is hidden).\n\n" +
		"In wrapper mode (under the shell function from `git-wt alias`) the\n" +
		"target path is written to fd N for the parent shell to cd into.\n" +
		"In bare mode the path is printed on stdout with a copy/paste hint\n" +
		"on stderr — supports `cd \"$(git-wt go branch)\"`.",
	Args: cobra.MaximumNArgs(1),
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

		target, err := resolveGoTarget(wts, cur, args, repo.MainRoot, wt.TreesDirFor(repo.MainRoot))
		if err != nil {
			return err
		}
		if target == nil {
			return nil // user cancelled the picker (ESC / Ctrl-C)
		}
		return emitTarget(target.Path)
	},
}

func init() {
	rootCmd.AddCommand(goCmd)
}

// resolveGoTarget returns the worktree the user wants to navigate to.
func resolveGoTarget(wts []wt.Worktree, cur *wt.Worktree, args []string, mainRoot, treesDir string) (*wt.Worktree, error) {
	if len(args) == 1 {
		t := findByBranch(wts, args[0])
		if t == nil {
			return nil, fmt.Errorf("no worktree for branch %q", args[0])
		}
		return t, nil
	}

	choices := filterOutCurrent(wts, cur)
	if len(choices) == 0 {
		return nil, fmt.Errorf("no other worktrees to navigate to")
	}
	if !interactive() {
		return nil, fmt.Errorf("no branch specified (use a branch arg in non-interactive mode)")
	}
	return pickWorktree("Choose a worktree", choices, mainRoot, treesDir)
}
