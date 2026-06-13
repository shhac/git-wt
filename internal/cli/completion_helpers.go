package cli

import (
	"context"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/git"
	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/wt"
)

// completionDesc formats a worktree's display location + recency as
// the description shown to the right of a completion candidate.
// Mirrors the columns rendered by `gwt ls`:
//
//   - location padded to parentW so the recency column lines up across
//     candidates (zsh/bash treat the description as opaque text and
//     don't align internal columns themselves);
//   - HumanSince left at its native 9-char fixed width so unit-pairs
//     like ` 3d   7h ` and ` 3d  10h ` align at the `h`.
//
// parentW comes from columnWidths over the whole candidate batch.
func completionDesc(t wt.Worktree, mainRoot, treesDir string, parentW int) string {
	return padRight(t.DisplayPath(mainRoot, treesDir), parentW) + "  " + ui.HumanSince(t.ModTime)
}

// worktreeBranchesForGo returns the branches of every worktree the
// caller could navigate to: all worktrees with a branch name, minus
// the current worktree (you can't `go` to where you already are).
// Each entry is `"branch\tdescription"` per Cobra's convention —
// shells that support completion descriptions (bash-v2, zsh, fish,
// pwsh) render the right half as metadata; bash-v1 just shows the
// branch.
//
// Pure: takes a slice + optional current + display anchors. Sorted
// by branch for stable output.
func worktreeBranchesForGo(wts []wt.Worktree, cur *wt.Worktree, mainRoot, treesDir string) []string {
	_, parentW := columnWidths(wts, mainRoot, treesDir)
	out := make([]string, 0, len(wts))
	for _, t := range wts {
		if t.Branch == "" {
			continue
		}
		if cur != nil && t.Path == cur.Path {
			continue
		}
		out = append(out, t.Branch+"\t"+completionDesc(t, mainRoot, treesDir, parentW))
	}
	sort.Strings(out)
	return out
}

// worktreeBranchesForRm returns branches eligible for removal: every
// worktree branch except the main repo's, minus any branches the
// shell already accepted on this same command line (so a second TAB
// after `rm feat-a ` doesn't re-offer feat-a). Format is the same
// `"branch\tdescription"` as worktreeBranchesForGo.
func worktreeBranchesForRm(wts []wt.Worktree, mainRoot, treesDir string, alreadyChosen []string) []string {
	taken := make(map[string]struct{}, len(alreadyChosen))
	for _, a := range alreadyChosen {
		taken[a] = struct{}{}
	}
	_, parentW := columnWidths(wts, mainRoot, treesDir)
	out := make([]string, 0, len(wts))
	for _, t := range wts {
		if t.Branch == "" || t.Path == mainRoot {
			continue
		}
		if _, dup := taken[t.Branch]; dup {
			continue
		}
		out = append(out, t.Branch+"\t"+completionDesc(t, mainRoot, treesDir, parentW))
	}
	sort.Strings(out)
	return out
}

// addCandidates returns ref names eligible for `git-wt add`: every
// local branch plus every remote-tracking ref, minus any branch
// already checked out in a worktree (re-adding would fail at runtime).
// `<remote>/HEAD` symbolic refs are excluded.
//
// The slash-bearing remote pattern (`origin/feat-x`) is preserved
// verbatim — that's what the user types and what `add`'s resolver
// expects.
func addCandidates(localBranches, remoteRefs []string, wts []wt.Worktree) []string {
	used := make(map[string]struct{}, len(wts))
	for _, t := range wts {
		if t.Branch != "" {
			used[t.Branch] = struct{}{}
		}
	}
	out := make([]string, 0, len(localBranches)+len(remoteRefs))
	for _, b := range localBranches {
		if _, taken := used[b]; taken {
			continue
		}
		out = append(out, b)
	}
	for _, r := range remoteRefs {
		// A remote ref `origin/feat-x` resolves to local branch `feat-x`
		// when added. If that local already exists in a worktree, skip.
		_, rest, ok := strings.Cut(r, "/")
		if ok {
			if _, taken := used[rest]; taken {
				continue
			}
		}
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// listLocalBranches enumerates short refs/heads names. Used by the
// `add` completer; small wrapper around `git for-each-ref` to keep
// the completer pure-testable.
func listLocalBranches(ctx context.Context) ([]string, error) {
	out, err := git.Run(ctx, "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if err != nil {
		return nil, err
	}
	return splitNonEmptyLines(out), nil
}

// listRemoteRefs enumerates short refs/remotes names except the
// `<remote>/HEAD` symbolic refs (which aren't useful as `add` targets).
func listRemoteRefs(ctx context.Context) ([]string, error) {
	out, err := git.Run(ctx, "for-each-ref", "--format=%(refname:short)", "refs/remotes/")
	if err != nil {
		return nil, err
	}
	lines := splitNonEmptyLines(out)
	filtered := lines[:0]
	for _, l := range lines {
		if strings.HasSuffix(l, "/HEAD") {
			continue
		}
		filtered = append(filtered, l)
	}
	return filtered, nil
}

func splitNonEmptyLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// completeGoBranches is the ValidArgsFunction for `git-wt go`. It
// silently returns no candidates on any error — completion must not
// fail loudly mid-typing.
func completeGoBranches(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ctx := context.Background()
	repo, wts, cur, err := loadRepoAndWorktrees(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return worktreeBranchesForGo(wts, cur, repo.MainRoot, wt.TreesDirFor(repo.MainRoot)),
		cobra.ShellCompDirectiveNoFileComp
}

// completeRmBranches is the ValidArgsFunction for `git-wt rm`. Args
// already on the command line are excluded so successive TABs offer
// the remaining branches.
func completeRmBranches(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	ctx := context.Background()
	repo, wts, _, err := loadRepoAndWorktrees(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return worktreeBranchesForRm(wts, repo.MainRoot, wt.TreesDirFor(repo.MainRoot), args),
		cobra.ShellCompDirectiveNoFileComp
}

// completeAddRef is the ValidArgsFunction for `git-wt add`. We don't
// try to complete the optional `<leaf>` positional — it's a free-form
// directory name, no useful candidate set.
func completeAddRef(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 2 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ctx := context.Background()
	wts, _, err := loadWorktrees(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	locals, err := listLocalBranches(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	remotes, err := listRemoteRefs(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return addCandidates(locals, remotes, wts), cobra.ShellCompDirectiveNoFileComp
}
