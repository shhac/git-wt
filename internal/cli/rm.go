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
	rmKeepBranch   bool
	rmDeleteBranch bool
	rmForce        bool
	rmForceBranch  bool
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
		"that would otherwise ask.\n\n" +
		"--force removes worktrees that still hold uncommitted work. It does\n" +
		"not affect branch deletion, which uses `git branch -d` and declines to\n" +
		"drop unmerged branches; --force-branch switches that to -D.",
	ValidArgsFunction: completeRmBranches,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if rmKeepBranch && rmDeleteBranch {
			return fmt.Errorf("--keep-branch and --delete-branch are mutually exclusive")
		}

		repo, wts, cur, err := loadRepoAndWorktrees(ctx)
		if err != nil {
			return err
		}

		targets, err := resolveRmTargets(wts, repo, args, wt.TreesDirFor(repo.MainRoot), rmForce)
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			return nil // user cancelled the picker
		}

		targets, err = preflightDirty(ctx, targets, rmForce)
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			return nil // cancelled, or every target was skipped
		}

		action, err := chooseRmAction(targets, rmKeepBranch, rmDeleteBranch)
		if err != nil {
			return err
		}
		if action == rmCancel {
			fmt.Fprintln(os.Stderr, "cancelled")
			return nil
		}

		return executeRm(ctx, repo, targets, cur, action, rmForceBranch)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVar(&rmKeepBranch, "keep-branch", false, "keep the local branch (default)")
	rmCmd.Flags().BoolVar(&rmDeleteBranch, "delete-branch", false, "also delete the local branch")
	rmCmd.Flags().BoolVar(&rmForce, "force", false, "remove worktrees that hold uncommitted changes")
	rmCmd.Flags().BoolVar(&rmForceBranch, "force-branch", false, "delete branches with -D, dropping unmerged commits")
}

// rmTarget is one removal unit: a registered worktree, or — the rescue case
// — an unregistered leftover directory inside the trees dir (what an earlier
// removal leaves behind when it dies mid-delete after git has already
// dropped the worktree from its records).
type rmTarget struct {
	wt.Worktree
	orphan bool
	dirty  wt.DirtyStat // filled by scanDirty, before anything is deleted
	force  bool         // cleared to remove despite dirty
}

// label is the human-readable name used in prompts and progress output.
func (t rmTarget) label() string {
	if t.orphan {
		return t.Path + " (unregistered leftover)"
	}
	return t.Display()
}

func toRmTargets(wts []wt.Worktree) []rmTarget {
	out := make([]rmTarget, len(wts))
	for i, t := range wts {
		out[i] = rmTarget{Worktree: t}
	}
	return out
}

// resolveRmTargets returns the worktrees to remove. It excludes the main
// worktree and refuses if the user explicitly named it.
func resolveRmTargets(wts []wt.Worktree, repo *wt.RepoInfo, args []string, treesDir string, force bool) ([]rmTarget, error) {
	if len(args) > 0 {
		return resolveRmFromArgs(wts, repo, args, treesDir, force)
	}

	pickable := filterRemovable(wts, repo)
	if len(pickable) == 0 {
		fmt.Fprintln(os.Stderr, "no worktrees to remove")
		return nil, nil
	}
	if !interactive() {
		return nil, fmt.Errorf("no branches specified (run with branch args in non-interactive mode)")
	}
	picked, err := pickWorktreesToRemove(pickable, repo.MainRoot, treesDir)
	if err != nil {
		return nil, err
	}
	return toRmTargets(picked), nil
}

func resolveRmFromArgs(wts []wt.Worktree, repo *wt.RepoInfo, args []string, treesDir string, force bool) ([]rmTarget, error) {
	out := make([]rmTarget, 0, len(args))
	for _, a := range args {
		t := findByBranch(wts, a)
		if t == nil {
			o, ok := orphanRmTarget(wts, treesDir, a)
			if !ok {
				return nil, fmt.Errorf("no worktree for branch %q", a)
			}
			if !interactive() && !force {
				return nil, fmt.Errorf("%s is not a registered worktree, just a leftover directory; use --force to delete it non-interactively", o.Path)
			}
			out = append(out, o)
			continue
		}
		if t.Path == repo.MainRoot {
			return nil, fmt.Errorf("cannot remove the main worktree (%q)", t.Display())
		}
		out = append(out, rmTarget{Worktree: *t})
	}
	return out, nil
}

// chooseRmAction implements the confirmation step. The keep/delete flags
// pre-narrow the option list; non-interactive mode picks the default and
// skips the prompt entirely.
func chooseRmAction(targets []rmTarget, keepBranch, deleteBranch bool) (action rmAction, err error) {
	defaultAction := rmTreeOnly
	if deleteBranch {
		defaultAction = rmTreeAndBranch
	}
	if !interactive() {
		return defaultAction, nil
	}

	end := debug.Op("pick.confirm", "rm-action")
	defer func() { end(err) }()

	choice, ok, err := picker.Confirm(rmSummary(targets), rmOptions(keepBranch, deleteBranch))
	if err != nil {
		return rmCancel, err
	}
	if !ok {
		return rmCancel, nil
	}
	return choice, nil
}

// rmSummary is the target list shown above the final confirm. Worktrees
// carrying uncommitted work are called out with their counts: this is the
// last screen before the deletion, so it should say what is at stake rather
// than leave the user to remember the dirty prompt two screens back.
func rmSummary(targets []rmTarget) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Remove %d worktree(s):", len(targets))
	for _, t := range targets {
		b.WriteString("\n    " + t.label())
		if t.dirty.Any() {
			fmt.Fprintf(&b, "  [discards %s]", t.dirty.Summary())
		}
	}
	return b.String()
}

// rmOptions builds the option list shown by chooseRmAction. Pure: only the
// flag combination matters. Cancel is always last.
func rmOptions(keepBranch, deleteBranch bool) []picker.Option[rmAction] {
	keepOpt := picker.Option[rmAction]{Label: "Worktree only (keep branch)", Value: rmTreeOnly}
	delOpt := picker.Option[rmAction]{Label: "Worktree and branch", Value: rmTreeAndBranch}
	cancelOpt := picker.Option[rmAction]{Label: "Cancel", Value: rmCancel}

	switch {
	case keepBranch:
		return []picker.Option[rmAction]{keepOpt, cancelOpt}
	case deleteBranch:
		return []picker.Option[rmAction]{delOpt, cancelOpt}
	default:
		return []picker.Option[rmAction]{keepOpt, delOpt, cancelOpt}
	}
}

// executeRm performs the removals. Whether a worktree with uncommitted work
// is removed anyway is per-target (set by the dirty preflight); branchForce
// governs only `git branch -d` vs `-D`. If the current worktree is one of the
// targets, we chdir to the main repo and emit its path so the parent shell
// follows. The emit runs whether or not every removal succeeded — otherwise
// a failure partway through the list would leave the shell sitting in a
// directory this function has already deleted.
func executeRm(ctx context.Context, repo *wt.RepoInfo, targets []rmTarget, cur *wt.Worktree, action rmAction, branchForce bool) (err error) {
	end := debug.Op("rm.execute", fmt.Sprintf("%d-target(s)", len(targets)))
	defer func() { end(err) }()

	bouncing := needsBounce(cur, targets)
	if bouncing {
		bounceEnd := debug.Op("chdir", repo.MainRoot)
		chdirErr := os.Chdir(repo.MainRoot)
		bounceEnd(chdirErr)
		if chdirErr != nil {
			return fmt.Errorf("chdir to main repo: %w", chdirErr)
		}
	}

	err = removeTargets(ctx, targets, action, branchForce)

	if bouncing {
		if emitErr := emitTarget(repo.MainRoot); emitErr != nil && err == nil {
			err = emitErr
		}
	}
	return err
}

// removeTargets deletes each target in order, reporting as it goes, and
// stops at the first failure. Split out of executeRm so the bounce emit can
// be a plain statement after it rather than a defer racing the debug span
// for the named return.
func removeTargets(ctx context.Context, targets []rmTarget, action rmAction, branchForce bool) error {
	branchFlag := "-d"
	if branchForce {
		branchFlag = "-D"
	}

	for i, t := range targets {
		if t.orphan {
			if err := deleteTreeWithProgress(t.label(), t.Path); err != nil {
				return rmProgressError(targets, i, fmt.Errorf("remove %s: %w", t.Path, err))
			}
			fmt.Fprintf(os.Stderr, "removed %s\n", t.label())
			continue
		}

		if err := removeWorktree(ctx, t.Worktree, t.force); err != nil {
			return rmProgressError(targets, i, fmt.Errorf("remove worktree %s: %w", t.Display(), err))
		}
		fmt.Fprintf(os.Stderr, "removed %s\n", t.Display())

		deleteBranchIfAsked(ctx, t, action, branchFlag)
	}

	return nil
}

// deleteBranchIfAsked removes t's local branch when the chosen action calls
// for it. Failure is a warning rather than an error: the worktree is already
// gone, and a surviving branch is recoverable where a half-done removal is
// not.
func deleteBranchIfAsked(ctx context.Context, t rmTarget, action rmAction, branchFlag string) {
	if action != rmTreeAndBranch || t.Branch == "" {
		return
	}
	if _, err := git.Run(ctx, "branch", branchFlag, t.Branch); err != nil {
		fmt.Fprintf(os.Stderr, "warning: delete branch %s: %v\n", t.Branch, err)
		return
	}
	fmt.Fprintf(os.Stderr, "deleted branch %s\n", t.Branch)
}

// rmProgressError decorates a mid-list failure with what did and did not
// happen. The per-target "removed" lines have usually scrolled away by the
// time the error surfaces, and a re-run needs to know where to resume.
func rmProgressError(targets []rmTarget, failed int, err error) error {
	if len(targets) == 1 {
		return err
	}
	suffix := fmt.Sprintf("\n  removed %d of %d", failed, len(targets))
	if remaining := targets[failed+1:]; len(remaining) > 0 {
		names := make([]string, len(remaining))
		for i, t := range remaining {
			names[i] = t.label()
		}
		suffix += "; not attempted: " + strings.Join(names, ", ")
	}
	return fmt.Errorf("%w%s", err, suffix)
}

// needsBounce reports whether any of targets is the current worktree, in
// which case rm must move the parent shell back to the main repo before
// deleting (the cwd would otherwise be invalidated).
func needsBounce(cur *wt.Worktree, targets []rmTarget) bool {
	if cur == nil {
		return false
	}
	for _, t := range targets {
		if t.Path == cur.Path {
			return true
		}
	}
	return false
}
