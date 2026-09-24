package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/wt"
)

var lockReason string

var lockCmd = &cobra.Command{
	Use:   "lock [branch...]",
	Short: "Lock worktrees against removal and pruning",
	Long: "Lock worktrees so `rm`, `clean` and `git worktree prune` leave them alone.\n\n" +
		"With branch arguments, locks those worktrees. Without, opens an\n" +
		"interactive multi-select over the unlocked ones. --reason records why;\n" +
		"list and the pickers show it next to how long the lock has been held.\n" +
		"Locking a worktree that is already locked keeps the existing lock.",
	ValidArgsFunction: completeLockBranches,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLockChange(cmd.Context(), args, true, lockReason)
	},
}

var unlockCmd = &cobra.Command{
	Use:   "unlock [branch...]",
	Short: "Unlock worktrees",
	Long: "Unlock worktrees, e.g. one left locked by a tool that has since exited.\n\n" +
		"With branch arguments, unlocks those worktrees. Without, opens an\n" +
		"interactive multi-select over the locked ones, showing how long each\n" +
		"has been locked and why.",
	ValidArgsFunction: completeUnlockBranches,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLockChange(cmd.Context(), args, false, "")
	},
}

func init() {
	rootCmd.AddCommand(lockCmd, unlockCmd)
	lockCmd.Flags().StringVar(&lockReason, "reason", "", "why the worktree is locked")
}

// runLockChange takes the named (or picked) worktrees to the wanted lock
// state: lock=true locks them with reason, lock=false unlocks them.
func runLockChange(ctx context.Context, args []string, lock bool, reason string) error {
	repo, wts, _, err := loadRepoAndWorktrees(ctx)
	if err != nil {
		return err
	}
	targets, err := resolveLockTargets(wts, repo, args, wt.TreesDirFor(repo.MainRoot), lock)
	if err != nil {
		return err
	}
	return applyLockChange(os.Stderr, targets, lock, func(t wt.Worktree) error {
		if lock {
			return wt.Lock(ctx, t.Path, reason)
		}
		return wt.Unlock(ctx, t.Path)
	})
}

// applyLockChange runs change on each target not already in the wanted
// state, reporting to w, and stops at the first failure. A target already in
// that state is a note rather than an error: the end state is what the user
// asked for. change is the git call, taken as a parameter so the reporting
// is testable without a repository.
func applyLockChange(w io.Writer, targets []wt.Worktree, lock bool, change func(wt.Worktree) error) error {
	for _, t := range targets {
		label := worktreeLabel(t)
		if !needsLockChange(t, lock) {
			_, _ = fmt.Fprintf(w, "%s is %s\n", label, lockStateDetail(t))
			continue
		}
		if err := change(t); err != nil {
			return fmt.Errorf("%s %s: %w", lockVerb(lock), label, err)
		}
		if lock {
			_, _ = fmt.Fprintf(w, "locked %s\n", label)
			continue
		}
		// Name the lock just released, so the user can see it was the one
		// they meant to clear.
		_, _ = fmt.Fprintf(w, "unlocked %s (was %s)\n", label, lockStateDetail(t))
	}
	return nil
}

// resolveLockTargets returns the named worktrees, or the ones picked
// interactively from those lock/unlock would change. Returns an empty slice
// when there is nothing to pick or the user cancels.
func resolveLockTargets(wts []wt.Worktree, repo *wt.RepoInfo, args []string, treesDir string, lock bool) ([]wt.Worktree, error) {
	verb := lockVerb(lock)
	if len(args) > 0 {
		return resolveNamedWorktrees(wts, repo, args, treesDir, verb)
	}

	pickable := filterNeedsLockChange(filterRemovable(wts, repo), lock)
	if len(pickable) == 0 {
		fmt.Fprintln(os.Stderr, nothingToLockChange(lock))
		return nil, nil
	}
	if !interactive() {
		return nil, fmt.Errorf("no branches specified (run with branch args in non-interactive mode)")
	}
	return pickWorktrees("Select worktrees to "+verb+" (space to toggle, enter to continue, esc to cancel)",
		pickable, repo.MainRoot, treesDir)
}

func lockVerb(lock bool) string {
	if lock {
		return "lock"
	}
	return "unlock"
}

func nothingToLockChange(lock bool) string {
	if lock {
		return "no unlocked worktrees to lock"
	}
	return "no locked worktrees"
}

// needsLockChange reports whether t is not yet in the wanted lock state —
// the one rule behind the skip note, the picker and tab completion.
func needsLockChange(t wt.Worktree, wantLocked bool) bool {
	return t.Locked != wantLocked
}

// filterNeedsLockChange keeps the worktrees that locking (wantLocked=true)
// or unlocking would change.
func filterNeedsLockChange(wts []wt.Worktree, wantLocked bool) []wt.Worktree {
	out := make([]wt.Worktree, 0, len(wts))
	for _, t := range wts {
		if needsLockChange(t, wantLocked) {
			out = append(out, t)
		}
	}
	return out
}

// resolveNamedWorktrees resolves each arg, refusing unknown names and the
// main worktree, and drops repeats by path.
func resolveNamedWorktrees(wts []wt.Worktree, repo *wt.RepoInfo, args []string, treesDir, verb string) ([]wt.Worktree, error) {
	seen := make(map[string]bool, len(args))
	out := make([]wt.Worktree, 0, len(args))
	for _, a := range args {
		t, err := findNamedWorktree(wts, repo, treesDir, a, verb)
		if err != nil {
			return nil, err
		}
		if t == nil {
			return nil, fmt.Errorf("no worktree for branch %q", a)
		}
		if seen[t.Path] {
			continue
		}
		seen[t.Path] = true
		out = append(out, *t)
	}
	return out, nil
}
