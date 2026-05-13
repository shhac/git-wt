package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/debug"
	"github.com/shhac/git-wt/internal/git"
	"github.com/shhac/git-wt/internal/picker"
	"github.com/shhac/git-wt/internal/wt"
)

var (
	ejectParentDir string
	ejectBase      string
)

var ejectCmd = &cobra.Command{
	Use:   "eject [<leaf>]",
	Short: "Move the currently-checked-out branch into a new worktree",
	Long: "Eject the current branch from the main working tree into a new worktree.\n" +
		"Stashes any uncommitted changes (tracked + untracked), switches the main\n" +
		"tree to `main`/`master` (or --base), creates the worktree, and restores\n" +
		"the stashed changes inside it.\n\n" +
		"Refuses when:\n" +
		"  - HEAD is detached\n" +
		"  - the current branch is the base branch\n" +
		"  - run from inside an existing worktree (must be in the main tree)\n" +
		"  - neither `main` nor `master` exists locally and --base isn't given",
	Args: cobra.MaximumNArgs(1),
	RunE: runEject,
}

func init() {
	rootCmd.AddCommand(ejectCmd)
	ejectCmd.Flags().StringVarP(&ejectParentDir, "parent-dir", "p", "", "parent directory for the worktree (default: <repo>/.gwt/)")
	ejectCmd.Flags().StringVar(&ejectBase, "base", "", "branch to switch the main tree to (default: main or master)")
}

func runEject(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var leafOverride string
	if len(args) == 1 {
		leafOverride = args[0]
	}

	repo, err := requireMutableRepo(ctx)
	if err != nil {
		return err
	}
	if repo.Root != repo.MainRoot {
		return fmt.Errorf("eject must be run from the main working tree (not a secondary worktree)")
	}

	currentBranch, err := wt.CurrentBranch(ctx)
	if err != nil {
		return err
	}

	base, err := detectBaseBranch(ctx, ejectBase)
	if err != nil {
		return err
	}
	if currentBranch == base {
		return fmt.Errorf("current branch %q is the base branch; nothing to eject", currentBranch)
	}

	leaf := leafOverride
	if leaf == "" {
		leaf = currentBranch
	}
	if err := wt.ValidateBranchName(leaf); err != nil {
		return fmt.Errorf("invalid leaf %q: %w", leaf, err)
	}

	parent, err := wt.ResolveParentDir(repo.MainRoot, ejectParentDir)
	if err != nil {
		return err
	}
	path, err := prepareWorktreeSite(parent, leaf, "leaf")
	if err != nil {
		return err
	}

	dirty, err := wt.IsWorkingTreeDirty(ctx)
	if err != nil {
		return err
	}

	if !confirmEject(currentBranch, base, path, dirty) {
		return fmt.Errorf("cancelled")
	}

	return executeEject(ctx, repo, currentBranch, base, path, parent, dirty)
}

// detectBaseBranch picks the branch the main tree should switch to. If
// override is set, it must exist locally. Otherwise we try `main`, then
// `master`. Returns an error if nothing matches.
func detectBaseBranch(ctx context.Context, override string) (string, error) {
	if override != "" {
		exists, err := wt.BranchExists(ctx, "", override)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("base branch %q does not exist locally", override)
		}
		return override, nil
	}
	for _, candidate := range []string{"main", "master"} {
		exists, err := wt.BranchExists(ctx, "", candidate)
		if err != nil {
			return "", err
		}
		if exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no `main` or `master` branch found; pass --base <name>")
}

// confirmEject prompts the user before doing anything destructive. Skipped
// when --non-interactive is set or stdin isn't a TTY.
func confirmEject(branch, base, path string, dirty bool) bool {
	if !interactive() {
		return true
	}
	stashMsg := "no uncommitted changes"
	if dirty {
		stashMsg = "stash uncommitted changes (incl. untracked)"
	}
	summary := fmt.Sprintf(
		"Eject %q to %s:\n    %s\n    switch main tree to %q\n    restore changes in new worktree",
		branch, path, stashMsg, base,
	)
	options := []picker.Option[bool]{
		{Label: "Proceed", Value: true},
		{Label: "Cancel", Value: false},
	}
	choice, ok, err := picker.Confirm(summary, options)
	if err != nil || !ok {
		return false
	}
	return choice
}

// executeEject performs the actual stash → switch → worktree-add → apply
// flow. Rollback on intermediate failure attempts to restore the original
// state (re-switch + unstash) so the user isn't left in a half-state.
func executeEject(ctx context.Context, repo *wt.RepoInfo, branch, base, path, parent string, dirty bool) error {
	end := debug.Op("eject.execute", branch)
	defer func() { end(nil) }()

	stashRef := ""
	if dirty {
		ref, err := wt.StashPush(ctx, fmt.Sprintf("git-wt eject of %s", branch))
		if err != nil {
			return fmt.Errorf("stash: %w", err)
		}
		stashRef = ref
	}

	if _, err := git.Run(ctx, "switch", base); err != nil {
		// Rollback: restore the stash so we're back where we started.
		rollbackStash(ctx, "", stashRef)
		return fmt.Errorf("switch to %s: %w", base, err)
	}

	resolved := &wt.AddRefResolution{
		Kind:      wt.AddRefLocal,
		SourceRef: branch,
		LocalName: branch,
	}
	if err := checkoutWorktree(ctx, path, resolved); err != nil {
		// Rollback: switch back to original branch + restore the stash.
		if _, swErr := git.Run(ctx, "switch", branch); swErr != nil {
			fmt.Fprintf(os.Stderr, "warning: rollback switch to %s failed: %v\n", branch, swErr)
		}
		rollbackStash(ctx, "", stashRef)
		return fmt.Errorf("create worktree: %w", err)
	}

	if stashRef != "" {
		switch outcome, applyErr := wt.StashApply(ctx, path, stashRef); outcome {
		case wt.StashApplied:
			// success, nothing to do
		case wt.StashAppliedKeptStash:
			fmt.Fprintf(os.Stderr,
				"warning: stash applied but failed to drop entry %s: %v\n",
				wt.ShortStashRef(stashRef), applyErr)
		case wt.StashApplyFailed:
			fmt.Fprintf(os.Stderr,
				"warning: stash apply did not complete cleanly: %v. Your changes are preserved as %s; "+
					"inspect with `git -C %s status` and recover with `git -C %s stash apply --index %s`\n",
				applyErr, wt.ShortStashRef(stashRef), path, path, stashRef)
		}
	}

	warnIfParentNotIgnored(ctx, repo.MainRoot, parent)
	return emitTarget(path)
}

// rollbackStash restores a previously-pushed stash to the working tree.
// Best-effort: errors are surfaced as warnings rather than returned, since
// we're already on an error path. dir == "" runs in the caller's cwd.
func rollbackStash(ctx context.Context, dir, stashRef string) {
	if stashRef == "" {
		return
	}
	outcome, err := wt.StashApply(ctx, dir, stashRef)
	switch outcome {
	case wt.StashApplied:
		// success
	case wt.StashAppliedKeptStash:
		fmt.Fprintf(os.Stderr, "warning: rollback stash drop (%s) failed: %v\n",
			wt.ShortStashRef(stashRef), err)
	case wt.StashApplyFailed:
		fmt.Fprintf(os.Stderr, "warning: rollback stash apply (%s) failed: %v\n",
			wt.ShortStashRef(stashRef), err)
	}
}

