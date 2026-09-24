package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/ui"
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
		return runLockChange(cmd.Context(), args, lockChange{
			verb:      "lock",
			lock:      true,
			pickTitle: "Select worktrees to lock (space to toggle, enter to continue, esc to cancel)",
			noneLeft:  "no unlocked worktrees to lock",
			apply: func(ctx context.Context, t wt.Worktree) error {
				return wt.Lock(ctx, t.Path, lockReason)
			},
		})
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
		return runLockChange(cmd.Context(), args, lockChange{
			verb:      "unlock",
			lock:      false,
			pickTitle: "Select worktrees to unlock (space to toggle, enter to continue, esc to cancel)",
			noneLeft:  "no locked worktrees",
			apply: func(ctx context.Context, t wt.Worktree) error {
				return wt.Unlock(ctx, t.Path)
			},
		})
	},
}

func init() {
	rootCmd.AddCommand(lockCmd, unlockCmd)
	lockCmd.Flags().StringVar(&lockReason, "reason", "", "why the worktree is locked")
}

// lockChange describes one direction of lock/unlock; the two commands differ
// only in these values.
type lockChange struct {
	verb      string
	lock      bool // the state the targets end up in
	pickTitle string
	noneLeft  string // said when no worktree can be picked
	apply     func(ctx context.Context, t wt.Worktree) error
}

func runLockChange(ctx context.Context, args []string, op lockChange) error {
	repo, wts, _, err := loadRepoAndWorktrees(ctx)
	if err != nil {
		return err
	}
	treesDir := wt.TreesDirFor(repo.MainRoot)

	targets, err := resolveLockTargets(wts, repo, args, treesDir, op)
	if err != nil {
		return err
	}
	for _, t := range targets {
		if t.Locked == op.lock {
			fmt.Fprintf(os.Stderr, "%s is %s\n", worktreeLabel(t), lockStateDetail(t))
			continue
		}
		if err := op.apply(ctx, t); err != nil {
			return fmt.Errorf("%s %s: %w", op.verb, worktreeLabel(t), err)
		}
		fmt.Fprintf(os.Stderr, "%sed %s%s\n", op.verb, worktreeLabel(t), unlockedNote(t, op))
	}
	return nil
}

// resolveLockTargets returns the named worktrees, or the ones picked
// interactively from those not already in the target state. Returns an
// empty slice when there is nothing to pick or the user cancels.
func resolveLockTargets(wts []wt.Worktree, repo *wt.RepoInfo, args []string, treesDir string, op lockChange) ([]wt.Worktree, error) {
	if len(args) > 0 {
		return resolveNamedWorktrees(wts, repo, args, treesDir, op.verb)
	}

	pickable := filterLockable(filterRemovable(wts, repo), !op.lock)
	if len(pickable) == 0 {
		fmt.Fprintln(os.Stderr, op.noneLeft)
		return nil, nil
	}
	if !interactive() {
		return nil, fmt.Errorf("no branches specified (run with branch args in non-interactive mode)")
	}
	return pickWorktrees(op.pickTitle, pickable, repo.MainRoot, treesDir)
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

// filterLockable keeps the worktrees whose Locked matches locked.
func filterLockable(wts []wt.Worktree, locked bool) []wt.Worktree {
	out := make([]wt.Worktree, 0, len(wts))
	for _, t := range wts {
		if t.Locked == locked {
			out = append(out, t)
		}
	}
	return out
}

// lockStateDetail describes t's lock state for messages: "not locked", or
// "locked" with how long ago and the full reason when known. Doubles as the
// "nothing to do" note, where the existing lock's age and reason tell the
// user whether it is the lock they meant to take.
func lockStateDetail(t wt.Worktree) string {
	if !t.Locked {
		return "not locked"
	}
	detail := "locked"
	if age := lockAge(t); age != "" {
		detail += " " + age + " ago"
	}
	if reason := oneLine(t.LockReason); reason != "" {
		detail += ": " + reason
	}
	return detail
}

// lockAge is how long t has been locked, unpadded ("3d 2h"); "" if unknown.
func lockAge(t wt.Worktree) string {
	if t.LockedAt.IsZero() {
		return ""
	}
	return strings.Join(strings.Fields(ui.HumanSince(t.LockedAt)), " ")
}

// unlockedNote says what lock was just released, so the user can see they
// cleared the one they meant to.
func unlockedNote(t wt.Worktree, op lockChange) string {
	if op.lock {
		return ""
	}
	return " (was " + lockStateDetail(t) + ")"
}
