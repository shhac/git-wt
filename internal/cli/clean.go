package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/git"
	"github.com/shhac/git-wt/internal/picker"
	"github.com/shhac/git-wt/internal/wt"
)

var (
	cleanDryRun       bool
	cleanNoFetch      bool
	cleanOrphanedOnly bool
	cleanGoneOnly     bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove worktrees that no longer have a useful branch",
	Long: "Remove worktrees in two cases:\n" +
		"  - orphaned: the local branch was deleted (or was never a branch)\n" +
		"  - upstream-gone: the local branch's upstream is gone (post-PR cleanup)\n\n" +
		"Both checks run by default. Use --orphaned-only or --upstream-gone-only\n" +
		"to narrow. `git worktree prune` runs first to clean up worktrees whose\n" +
		"directory no longer exists. `git fetch --prune` runs before the\n" +
		"upstream-gone check (skip with --no-fetch).",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cleanOrphanedOnly && cleanGoneOnly {
			return fmt.Errorf("--orphaned-only and --upstream-gone-only are mutually exclusive")
		}
		return runClean(cmd.Context(), cleanFlags{
			dryRun:       cleanDryRun,
			noFetch:      cleanNoFetch,
			orphanedOnly: cleanOrphanedOnly,
			goneOnly:     cleanGoneOnly,
		})
	},
}

// cleanFlags is the resolved flag state for one clean invocation. Pulled out
// of the global flagXxx vars so runClean is callable from tests without
// touching package state.
type cleanFlags struct {
	dryRun, noFetch          bool
	orphanedOnly, goneOnly   bool
}

// runClean is the body of `git-wt clean`. Steps: setup → discover targets
// → render → maybe-confirm → delegate to executeRm.
func runClean(ctx context.Context, flags cleanFlags) error {
	repo, err := wt.Inspect(ctx, "")
	if err != nil {
		return err
	}

	// Prune any worktrees whose directories have been removed manually.
	// Cheap and almost always desirable.
	if _, err := git.Run(ctx, "worktree", "prune"); err != nil {
		return fmt.Errorf("git worktree prune: %w", err)
	}

	doOrphaned := !flags.goneOnly
	doGone := !flags.orphanedOnly

	if doGone && !flags.noFetch {
		fmt.Fprintln(os.Stderr, "fetching with --prune…")
		if _, err := git.Run(ctx, "fetch", "--prune"); err != nil {
			// Soft-fail: maybe no remote, or offline. Carry on; the orphaned
			// check still works without a fetch.
			fmt.Fprintf(os.Stderr, "warning: git fetch --prune failed: %v\n", err)
		}
	}

	wts, err := wt.List(ctx, "")
	if err != nil {
		return err
	}
	cur := wt.Current(wts, mustWD())

	targets, err := collectCleanTargets(ctx, wts, repo, doOrphaned, doGone)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "nothing to clean")
		return nil
	}

	printCleanTargets(os.Stderr, targets)
	if flags.dryRun {
		return nil
	}
	if interactive() {
		ok, err := confirmClean(len(targets))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "cancelled")
			return nil
		}
	}

	toRm := make([]wt.Worktree, len(targets))
	for i, t := range targets {
		toRm[i] = t.wt
	}
	// force=true: the branches/upstream are already gone, the user has
	// confirmed (or asked for non-interactive cleanup).
	return executeRm(ctx, repo, toRm, cur, rmTreeAndBranch, true)
}

// collectCleanTargets runs the requested checks and returns a deduped target
// list. doOrphaned/doGone gate the two scans.
func collectCleanTargets(ctx context.Context, wts []wt.Worktree, repo *wt.RepoInfo, doOrphaned, doGone bool) ([]taggedTarget, error) {
	var targets []taggedTarget
	if doOrphaned {
		branchExists := func(b string) (bool, error) { return wt.BranchExists(ctx, "", b) }
		ts, err := findOrphanedWorktrees(wts, repo, branchExists)
		if err != nil {
			return nil, err
		}
		targets = append(targets, ts...)
	}
	if doGone {
		gone, err := goneBranches(ctx)
		if err != nil {
			return nil, err
		}
		ts := findUpstreamGoneWorktrees(wts, repo, goneSetFromList(gone))
		seen := pathSet(targets)
		for _, t := range ts {
			if _, dupe := seen[t.wt.Path]; !dupe {
				targets = append(targets, t)
			}
		}
	}
	return targets, nil
}

// printCleanTargets writes the human-readable target list. Pure: takes any io.Writer.
func printCleanTargets(w io.Writer, targets []taggedTarget) {
	fmt.Fprintln(w, "worktrees to remove:")
	for _, t := range targets {
		fmt.Fprintf(w, "  %s  [%s]  (%s)\n", t.wt.Display(), t.reason, t.wt.Path)
	}
}

// confirmClean prompts the user for final approval (interactive only). ESC /
// Ctrl-C are treated as a cancel response (returns false, nil) rather than
// an error.
func confirmClean(n int) (bool, error) {
	choice, ok, err := picker.Confirm(
		fmt.Sprintf("Remove these %d worktree(s) and their branches?", n),
		[]picker.Option[bool]{
			{Label: "Remove", Value: true},
			{Label: "Cancel", Value: false},
		},
	)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	return choice, nil
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "list candidates without removing them")
	cleanCmd.Flags().BoolVar(&cleanNoFetch, "no-fetch", false, "skip the leading `git fetch --prune`")
	cleanCmd.Flags().BoolVar(&cleanOrphanedOnly, "orphaned-only", false, "only remove worktrees whose local branch is gone")
	cleanCmd.Flags().BoolVar(&cleanGoneOnly, "upstream-gone-only", false, "only remove worktrees whose upstream tracking is gone")
}

// taggedTarget pairs a worktree with the reason it was selected for cleanup.
type taggedTarget struct {
	wt     wt.Worktree
	reason string
}

func pathSet(ts []taggedTarget) map[string]struct{} {
	out := make(map[string]struct{}, len(ts))
	for _, t := range ts {
		out[t.wt.Path] = struct{}{}
	}
	return out
}

// branchExistsFn is the dependency-injection seam for BranchExists. The cli
// uses wt.BranchExists; tests pass a fake.
type branchExistsFn func(branch string) (bool, error)

// findOrphanedWorktrees returns worktrees whose local branch no longer exists,
// or who report as detached/prunable (i.e. the branch is gone or never was).
// branchExists is injected so the classification logic is testable without
// running git.
func findOrphanedWorktrees(wts []wt.Worktree, repo *wt.RepoInfo, branchExists branchExistsFn) ([]taggedTarget, error) {
	var out []taggedTarget
	for _, t := range wts {
		if t.Path == repo.MainRoot {
			continue
		}
		if t.Prunable {
			out = append(out, taggedTarget{wt: t, reason: "prunable"})
			continue
		}
		if t.Branch == "" {
			// Detached or bare — not a deletable-branch case but a worktree
			// without a branch is functionally orphaned for our purposes.
			out = append(out, taggedTarget{wt: t, reason: "detached"})
			continue
		}
		exists, err := branchExists(t.Branch)
		if err != nil {
			return nil, err
		}
		if !exists {
			out = append(out, taggedTarget{wt: t, reason: "branch deleted"})
		}
	}
	return out, nil
}

// findUpstreamGoneWorktrees returns worktrees whose branch is in goneSet.
// Pure: caller computes goneSet (typically via goneBranches→parseGoneBranches).
func findUpstreamGoneWorktrees(wts []wt.Worktree, repo *wt.RepoInfo, goneSet map[string]struct{}) []taggedTarget {
	var out []taggedTarget
	for _, t := range wts {
		if t.Path == repo.MainRoot || t.Branch == "" {
			continue
		}
		if _, ok := goneSet[t.Branch]; ok {
			out = append(out, taggedTarget{wt: t, reason: "upstream gone"})
		}
	}
	return out
}

// goneBranches returns local branches whose upstream has been pruned ([gone]).
// Wraps the git call + the pure parser; tests target parseGoneBranches.
func goneBranches(ctx context.Context) ([]string, error) {
	out, err := git.Run(ctx,
		"for-each-ref",
		"--format=%(refname:short)\t%(upstream:track)",
		"refs/heads/",
	)
	if err != nil {
		return nil, err
	}
	return parseGoneBranches(out), nil
}

// parseGoneBranches extracts branch names whose tracking column contains
// "gone" from the output of `git for-each-ref --format=…\t%(upstream:track)`.
func parseGoneBranches(output string) []string {
	var gone []string
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		name, track, _ := strings.Cut(line, "\t")
		if strings.Contains(track, "gone") {
			gone = append(gone, name)
		}
	}
	return gone
}

// goneSetFromList is a small helper for tests and callers that need the
// gone-branches list as a set.
func goneSetFromList(names []string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, n := range names {
		out[n] = struct{}{}
	}
	return out
}
