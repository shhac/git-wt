package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/fd"
	"github.com/shhac/git-wt/internal/ui"
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
		wts, err := wt.List(ctx, "")
		if err != nil {
			return err
		}
		wt.SortByModTime(wts)
		cur := wt.Current(wts, mustWD())

		target, err := resolveGoTarget(wts, cur, args)
		if err != nil {
			return err
		}
		return emitTarget(target.Path)
	},
}

func init() {
	rootCmd.AddCommand(goCmd)
}

// resolveGoTarget returns the worktree the user wants to navigate to.
func resolveGoTarget(wts []wt.Worktree, cur *wt.Worktree, args []string) (*wt.Worktree, error) {
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
	return pickWorktree("Choose a worktree", choices)
}

// findByBranch returns the worktree whose Branch matches name exactly. If no
// exact match is found, falls back to a unique suffix match on the branch
// (e.g. "auth" can match "paul/auth" if it's the only candidate).
func findByBranch(wts []wt.Worktree, name string) *wt.Worktree {
	for i := range wts {
		if wts[i].Branch == name {
			return &wts[i]
		}
	}
	var hits []*wt.Worktree
	for i := range wts {
		b := wts[i].Branch
		if b == "" {
			continue
		}
		if strings.HasSuffix(b, "/"+name) || b == name {
			hits = append(hits, &wts[i])
		}
	}
	if len(hits) == 1 {
		return hits[0]
	}
	return nil
}

// filterOutCurrent returns the list with cur removed (if non-nil).
func filterOutCurrent(wts []wt.Worktree, cur *wt.Worktree) []wt.Worktree {
	if cur == nil {
		return wts
	}
	out := make([]wt.Worktree, 0, len(wts))
	for _, t := range wts {
		if t.Path != cur.Path {
			out = append(out, t)
		}
	}
	return out
}

// pickWorktree opens an interactive single-select over wts.
func pickWorktree(title string, wts []wt.Worktree) (*wt.Worktree, error) {
	branchW, parentW := columnWidths(wts)
	options := make([]huh.Option[string], len(wts))
	for i, t := range wts {
		label := formatPickerRow(t, branchW, parentW)
		options[i] = huh.NewOption(label, t.Path)
	}

	var pickedPath string
	err := huh.NewSelect[string]().
		Title(title).
		Options(options...).
		Value(&pickedPath).
		WithTheme(huh.ThemeBase()).
		Run()
	if err != nil {
		return nil, err
	}
	for i := range wts {
		if wts[i].Path == pickedPath {
			return &wts[i], nil
		}
	}
	return nil, fmt.Errorf("internal: picked worktree not in list")
}

// formatPickerRow lays out one row "branch  parent  mtime" with aligned columns.
func formatPickerRow(t wt.Worktree, branchW, parentW int) string {
	branch := padRight(t.Display(), branchW)
	parent := padRight(t.ParentDirName(), parentW)
	mtime := ui.HumanSince(t.ModTime)
	return fmt.Sprintf("%s  %s  %s", branch, ui.Dim(parent), ui.Dim(mtime))
}

func columnWidths(wts []wt.Worktree) (branchW, parentW int) {
	for _, t := range wts {
		if n := lipgloss.Width(t.Display()); n > branchW {
			branchW = n
		}
		if n := lipgloss.Width(t.ParentDirName()); n > parentW {
			parentW = n
		}
	}
	return
}

// emitTarget delivers a path to the caller. Wrapper mode writes to fd N;
// bare mode prints to stdout with a copy/paste hint on stderr.
func emitTarget(path string) error {
	if w, ok := fd.Open(flagFD); ok {
		defer w.Close()
		_, err := fmt.Fprintln(w, path)
		return err
	}
	fmt.Println(path)
	arrow := "→"
	if ui.Plain {
		arrow = "->"
	}
	fmt.Fprintf(os.Stderr, "%s cd %s\n", arrow, shellQuote(path))
	return nil
}
