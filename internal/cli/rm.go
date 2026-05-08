package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/git"
	"github.com/shhac/git-wt/internal/wt"
)

var (
	rmKeepBranch   bool
	rmDeleteBranch bool
	rmForce        bool
)

// rmAction is the disposition picked for a rm operation.
type rmAction int

const (
	rmCancel rmAction = iota
	rmTreeOnly
	rmTreeAndBranch
)

var rmCmd = &cobra.Command{
	Use:     "rm [branch...]",
	Aliases: []string{"remove"},
	Short:   "Remove one or more worktrees",
	Long: "Remove one or more worktrees.\n\n" +
		"With branch arguments, removes those worktrees. Without, opens an\n" +
		"interactive multi-select over the non-main worktrees.\n\n" +
		"By default the local branch is kept. Pass --delete-branch to remove\n" +
		"the branch as well, or --keep-branch to silence the interactive prompt\n" +
		"that would otherwise ask. Use --force to skip the uncommitted-changes\n" +
		"safety check.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if rmKeepBranch && rmDeleteBranch {
			return fmt.Errorf("--keep-branch and --delete-branch are mutually exclusive")
		}

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

		targets, err := resolveRmTargets(wts, repo, args)
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			return nil // user cancelled the picker
		}

		action, err := chooseRmAction(targets)
		if err != nil {
			return err
		}
		if action == rmCancel {
			fmt.Fprintln(os.Stderr, "cancelled")
			return nil
		}

		return executeRm(ctx, repo, targets, cur, action, rmForce)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVar(&rmKeepBranch, "keep-branch", false, "keep the local branch (default)")
	rmCmd.Flags().BoolVar(&rmDeleteBranch, "delete-branch", false, "also delete the local branch")
	rmCmd.Flags().BoolVar(&rmForce, "force", false, "skip uncommitted-changes safety checks")
}

// resolveRmTargets returns the worktrees to remove. It excludes the main
// worktree and refuses if the user explicitly named it.
func resolveRmTargets(wts []wt.Worktree, repo *wt.RepoInfo, args []string) ([]wt.Worktree, error) {
	if len(args) > 0 {
		return resolveRmFromArgs(wts, repo, args)
	}

	pickable := filterRemovable(wts, repo)
	if len(pickable) == 0 {
		fmt.Fprintln(os.Stderr, "no worktrees to remove")
		return nil, nil
	}
	if !interactive() {
		return nil, fmt.Errorf("no branches specified (run with branch args in non-interactive mode)")
	}
	picked, err := pickWorktreesToRemove(pickable)
	if err != nil {
		return nil, err
	}
	return picked, nil
}

func resolveRmFromArgs(wts []wt.Worktree, repo *wt.RepoInfo, args []string) ([]wt.Worktree, error) {
	out := make([]wt.Worktree, 0, len(args))
	for _, a := range args {
		t := findByBranch(wts, a)
		if t == nil {
			return nil, fmt.Errorf("no worktree for branch %q", a)
		}
		if t.Path == repo.MainRoot {
			return nil, fmt.Errorf("cannot remove the main worktree (%q)", t.Display())
		}
		out = append(out, *t)
	}
	return out, nil
}

func filterRemovable(wts []wt.Worktree, repo *wt.RepoInfo) []wt.Worktree {
	out := make([]wt.Worktree, 0, len(wts))
	for _, t := range wts {
		if t.Path == repo.MainRoot {
			continue
		}
		out = append(out, t)
	}
	return out
}

// pickWorktreesToRemove opens a huh multi-select.
func pickWorktreesToRemove(wts []wt.Worktree) ([]wt.Worktree, error) {
	branchW, parentW := columnWidths(wts)
	options := make([]huh.Option[string], len(wts))
	for i, t := range wts {
		options[i] = huh.NewOption(formatPickerRow(t, branchW, parentW), t.Path)
	}

	var picked []string
	err := huh.NewMultiSelect[string]().
		Title("Select worktrees to remove (space to toggle, enter to continue)").
		Options(options...).
		Value(&picked).
		WithTheme(huh.ThemeBase()).
		Run()
	if err != nil {
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

// chooseRmAction implements the confirmation step. Flags pre-narrow the
// options; non-interactive mode picks the default and skips the prompt.
func chooseRmAction(targets []wt.Worktree) (rmAction, error) {
	defaultAction := rmTreeOnly
	if rmDeleteBranch {
		defaultAction = rmTreeAndBranch
	}
	if !interactive() {
		return defaultAction, nil
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Remove %d worktree(s):", len(targets)))
	for _, t := range targets {
		summary.WriteString("\n    " + t.Display())
	}

	options := []huh.Option[rmAction]{}
	switch {
	case rmKeepBranch:
		options = append(options, huh.NewOption("Worktree only (keep branch)", rmTreeOnly))
	case rmDeleteBranch:
		options = append(options, huh.NewOption("Worktree and branch", rmTreeAndBranch))
	default:
		options = append(options,
			huh.NewOption("Worktree only (keep branch)", rmTreeOnly),
			huh.NewOption("Worktree and branch", rmTreeAndBranch),
		)
	}
	options = append(options, huh.NewOption("Cancel", rmCancel))

	var choice rmAction
	err := huh.NewSelect[rmAction]().
		Title(summary.String()).
		Options(options...).
		Value(&choice).
		WithTheme(huh.ThemeBase()).
		Run()
	if err != nil {
		return rmCancel, err
	}
	return choice, nil
}

// executeRm performs the removals. If the current worktree is one of the
// targets, we chdir to the main repo and emit its path so the parent shell
// follows.
func executeRm(ctx context.Context, repo *wt.RepoInfo, targets []wt.Worktree, cur *wt.Worktree, action rmAction, force bool) error {
	bouncing := false
	if cur != nil {
		for _, t := range targets {
			if t.Path == cur.Path {
				bouncing = true
				break
			}
		}
	}
	if bouncing {
		if err := os.Chdir(repo.MainRoot); err != nil {
			return fmt.Errorf("chdir to main repo: %w", err)
		}
	}

	for _, t := range targets {
		args := []string{"worktree", "remove", t.Path}
		if force {
			args = append(args, "--force")
		}
		if _, err := git.Run(ctx, args...); err != nil {
			return fmt.Errorf("remove worktree %s: %w", t.Display(), err)
		}
		fmt.Fprintf(os.Stderr, "removed %s\n", t.Display())

		if action == rmTreeAndBranch && t.Branch != "" {
			delArgs := []string{"branch"}
			if force {
				delArgs = append(delArgs, "-D")
			} else {
				delArgs = append(delArgs, "-d")
			}
			delArgs = append(delArgs, t.Branch)
			if _, err := git.Run(ctx, delArgs...); err != nil {
				fmt.Fprintf(os.Stderr, "warning: delete branch %s: %v\n", t.Branch, err)
				continue
			}
			fmt.Fprintf(os.Stderr, "deleted branch %s\n", t.Branch)
		}
	}

	if bouncing {
		return emitTarget(repo.MainRoot)
	}
	return nil
}
