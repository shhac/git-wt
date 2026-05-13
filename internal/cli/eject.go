package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

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

	currentBranch, err := currentBranch(ctx)
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

	dirty, err := workingTreeDirty(ctx)
	if err != nil {
		return err
	}

	if !confirmEject(currentBranch, base, path, dirty) {
		return fmt.Errorf("cancelled")
	}

	return executeEject(ctx, repo, currentBranch, base, path, parent, dirty)
}

// currentBranch returns the short branch name HEAD points at. Errors if
// HEAD is detached (which is the failure mode we want to surface).
func currentBranch(ctx context.Context) (string, error) {
	out, err := git.Run(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("HEAD is detached or not on a branch: %w", err)
	}
	return strings.TrimSpace(out), nil
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

// workingTreeDirty reports whether `git status --porcelain` returns any
// modifications (tracked, staged, or untracked).
func workingTreeDirty(ctx context.Context) (bool, error) {
	out, err := git.Run(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
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
		ref, err := stashUncommitted(ctx, fmt.Sprintf("git-wt eject of %s", branch))
		if err != nil {
			return fmt.Errorf("stash: %w", err)
		}
		stashRef = ref
	}

	if _, err := git.Run(ctx, "switch", base); err != nil {
		// Rollback: unstash so we're back where we started.
		if stashRef != "" {
			_, _ = git.Run(ctx, "stash", "pop", "--index", stashRef)
		}
		return fmt.Errorf("switch to %s: %w", base, err)
	}

	resolved := &wt.AddRefResolution{
		Kind:      wt.AddRefLocal,
		SourceRef: branch,
		LocalName: branch,
	}
	if err := checkoutWorktree(ctx, path, resolved); err != nil {
		// Rollback: switch back to original branch + unstash.
		_, _ = git.Run(ctx, "switch", branch)
		if stashRef != "" {
			_, _ = git.Run(ctx, "stash", "pop", "--index", stashRef)
		}
		return fmt.Errorf("create worktree: %w", err)
	}

	if stashRef != "" {
		applied := applyStashInWorktree(ctx, path, stashRef)
		if !applied {
			fmt.Fprintf(os.Stderr,
				"warning: stash apply did not complete cleanly. Your changes are preserved as %s; "+
					"inspect with `git -C %s status` and recover with `git -C %s stash apply --index %s`\n",
				shortStashRef(stashRef), path, path, stashRef,
			)
		}
	}

	warnIfParentNotIgnored(ctx, repo.MainRoot, parent)
	return emitTarget(path)
}

// stashUncommitted pushes a stash including untracked files and returns
// the stash's commit SHA so we can apply it explicitly later (independent
// of the @{N} index, which can shift if other stashes get pushed).
func stashUncommitted(ctx context.Context, message string) (string, error) {
	if _, err := git.Run(ctx, "stash", "push", "--include-untracked", "-m", message); err != nil {
		return "", err
	}
	sha, err := git.Run(ctx, "rev-parse", "stash@{0}")
	if err != nil {
		return "", fmt.Errorf("capture stash ref: %w", err)
	}
	return strings.TrimSpace(sha), nil
}

// applyStashInWorktree applies the stash by SHA inside worktreeDir.
// Returns true if applied cleanly (and the stash entry was dropped) and
// false on any failure — including merge conflicts. The stash is left
// intact on failure so the user can recover.
func applyStashInWorktree(ctx context.Context, worktreeDir, stashRef string) bool {
	_, err := git.RunIn(ctx, worktreeDir, "stash", "apply", "--index", stashRef)
	if err != nil {
		// stash apply prints conflicts to stderr and exits non-zero. We
		// don't try to distinguish conflicts from harder failures — the
		// caller's warning tells the user the stash is preserved either way.
		return false
	}
	// Clean apply — find the stash entry by SHA and drop it.
	if err := dropStashBySHA(ctx, stashRef); err != nil {
		fmt.Fprintf(os.Stderr, "warning: applied cleanly but failed to drop stash %s: %v\n",
			shortStashRef(stashRef), err)
	}
	return true
}

// dropStashBySHA scans `git stash list` for an entry matching sha and
// drops it. Matching by SHA (not by @{N}) avoids dropping the wrong stash
// if another process pushed in the meantime.
func dropStashBySHA(ctx context.Context, sha string) error {
	out, err := git.Run(ctx, "stash", "list", "--format=%H %gd")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.SplitN(line, " ", 2)
		if len(fields) != 2 {
			continue
		}
		if fields[0] == sha {
			_, err := git.Run(ctx, "stash", "drop", fields[1])
			return err
		}
	}
	return fmt.Errorf("stash %s not found in stash list", shortStashRef(sha))
}

func shortStashRef(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

