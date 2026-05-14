package cli

import (
	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/wt"
)

var (
	addParentDir      string
	addNoCopy         bool
	addCopyFileConfig string
)

var addCmd = &cobra.Command{
	Use:   "add [<leaf>] <branch|remote-ref>",
	Short: "Create a worktree for an existing local or remote branch",
	Long: "Create a worktree at <repo>/.worktrees/<leaf>/ for an existing branch.\n" +
		"With one positional, <branch|remote-ref> serves both as the branch to\n" +
		"check out and as the leaf directory name. With two positionals, the\n" +
		"first overrides the leaf.\n\n" +
		"A remote-ref is `<remote>/<rest>` where <remote> matches `git remote`\n" +
		"and refs/remotes/<remote>/<rest> exists; git creates a local branch\n" +
		"named <rest> tracking the remote. Anything else resolves as a local\n" +
		"branch — so `paul/auth-bug` (a slash-bearing local name) works as\n" +
		"long as no remote is called `paul`.\n\n" +
		"This command never creates new branches; for that, use `new`.",
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: completeAddRef,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// The ref is always the last positional; the optional first
		// positional overrides the leaf.
		refArg := args[len(args)-1]
		var leafOverride string
		if len(args) == 2 {
			leafOverride = args[0]
		}

		repo, err := requireMutableRepo(ctx)
		if err != nil {
			return err
		}

		resolved, err := wt.ResolveAddRef(ctx, "", refArg)
		if err != nil {
			return err
		}

		leaf, err := defaultedLeaf(leafOverride, resolved.LocalName)
		if err != nil {
			return err
		}

		parent, err := resolveParentDir(ctx, repo.MainRoot, addParentDir)
		if err != nil {
			return err
		}
		path, err := prepareWorktreeSite(parent, leaf, "leaf")
		if err != nil {
			return err
		}

		if err := checkoutWorktree(ctx, path, resolved); err != nil {
			return err
		}
		return finalizeWorktreeSite(ctx, repo, path, parent, addNoCopy, addCopyFileConfig)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addParentDir, "parent-dir", "p", "", "parent directory for the worktree (default: <repo>/.worktrees/)")
	addCmd.Flags().BoolVar(&addNoCopy, "no-copy", false, "skip copying project config files")
	addCmd.Flags().StringVar(&addCopyFileConfig, "copy-file-config", "", "path to copy spec (default: <repo>/.git-wt-copy-files)")
}

