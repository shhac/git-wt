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
	cleanDryRun  bool
	cleanNoFetch bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove worktrees whose branch was deleted from remote",
	Long: "Remove worktrees whose tracking branch is gone (typically: the PR\n" +
		"was merged + the remote branch deleted).\n\n" +
		"By default a `git fetch --prune` is run first. Pass --dry-run to\n" +
		"see what would be removed without acting.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		repo, err := wt.Inspect(ctx, "")
		if err != nil {
			return err
		}

		if !cleanNoFetch {
			fmt.Fprintln(os.Stderr, "fetching with --prune…")
			if _, err := git.Run(ctx, "fetch", "--prune"); err != nil {
				return fmt.Errorf("git fetch --prune: %w", err)
			}
		}

		gone, err := goneBranches(ctx)
		if err != nil {
			return err
		}
		if len(gone) == 0 {
			fmt.Fprintln(os.Stderr, "nothing to clean")
			return nil
		}

		wts, err := wt.List(ctx, "")
		if err != nil {
			return err
		}
		cur := wt.Current(wts, mustWD())

		targets := matchWorktreesByBranch(wts, repo, gone)
		if len(targets) == 0 {
			fmt.Fprintln(os.Stderr, "no worktrees match gone branches (gone branches without worktrees:", strings.Join(gone, ", ")+")")
			return nil
		}

		fmt.Fprintln(os.Stderr, "worktrees to remove:")
		for _, t := range targets {
			fmt.Fprintf(os.Stderr, "  %s  (%s)\n", t.Display(), t.Path)
		}

		if cleanDryRun {
			return nil
		}

		if interactive() {
			var confirm bool
			err := huh.NewConfirm().
				Title(fmt.Sprintf("Remove these %d worktree(s) and their branches?", len(targets))).
				Affirmative("Remove").
				Negative("Cancel").
				Value(&confirm).
				WithTheme(huh.ThemeBase()).
				Run()
			if err != nil {
				return err
			}
			if !confirm {
				fmt.Fprintln(os.Stderr, "cancelled")
				return nil
			}
		}

		// Always pass force=true: branches are gone upstream and the user
		// has confirmed (or asked for non-interactive cleanup).
		return executeRm(ctx, repo, targets, cur, rmTreeAndBranch, true)
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "list candidates without removing them")
	cleanCmd.Flags().BoolVar(&cleanNoFetch, "no-fetch", false, "skip the leading `git fetch --prune`")
}

// goneBranches returns local branches whose upstream has been pruned (`[gone]`).
func goneBranches(ctx context.Context) ([]string, error) {
	out, err := git.Run(ctx,
		"for-each-ref",
		"--format=%(refname:short)\t%(upstream:track)",
		"refs/heads/",
	)
	if err != nil {
		return nil, err
	}
	var gone []string
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		name, track, _ := strings.Cut(line, "\t")
		if strings.Contains(track, "gone") {
			gone = append(gone, name)
		}
	}
	return gone, nil
}

// matchWorktreesByBranch returns worktrees whose branch is in names. The
// main worktree is always excluded from the result.
func matchWorktreesByBranch(wts []wt.Worktree, repo *wt.RepoInfo, names []string) []wt.Worktree {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	out := make([]wt.Worktree, 0, len(wts))
	for _, t := range wts {
		if t.Path == repo.MainRoot || t.Branch == "" {
			continue
		}
		if _, ok := set[t.Branch]; ok {
			out = append(out, t)
		}
	}
	return out
}

