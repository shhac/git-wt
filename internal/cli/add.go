package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/debug"
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
	Long: "Create a worktree at <repo>/.gwt/<leaf>/ for an existing branch.\n" +
		"With one positional, <branch|remote-ref> serves both as the branch to\n" +
		"check out and as the leaf directory name. With two positionals, the\n" +
		"first overrides the leaf.\n\n" +
		"A remote-ref is `<remote>/<rest>` where <remote> matches `git remote`\n" +
		"and refs/remotes/<remote>/<rest> exists; git creates a local branch\n" +
		"named <rest> tracking the remote. Anything else resolves as a local\n" +
		"branch — so `paul/auth-bug` (a slash-bearing local name) works as\n" +
		"long as no remote is called `paul`.\n\n" +
		"This command never creates new branches; for that, use `new`.",
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		var leafOverride, refArg string
		if len(args) == 2 {
			leafOverride, refArg = args[0], args[1]
		} else {
			refArg = args[0]
		}

		repo, err := wt.Inspect(ctx, "")
		if err != nil {
			return err
		}
		if repo.Bare {
			return fmt.Errorf("cannot create worktrees in a bare repository")
		}
		clean, op, err := wt.IsClean(ctx, "")
		if err != nil {
			return err
		}
		if !clean {
			return fmt.Errorf("repository has a %s in progress; complete or abort it first", op)
		}

		resolved, err := wt.ResolveAddRef(ctx, "", refArg)
		if err != nil {
			return err
		}

		leaf := leafOverride
		if leaf == "" {
			leaf = resolved.LocalName
		}
		if err := wt.ValidateBranchName(leaf); err != nil {
			return fmt.Errorf("invalid leaf %q: %w", leaf, err)
		}

		parent := addParentDir
		if parent == "" {
			parent = wt.TreesDirFor(repo.MainRoot)
		} else {
			parent, err = filepath.Abs(parent)
			if err != nil {
				return err
			}
		}
		path := wt.ConstructPath(parent, leaf)
		if wt.PathExists(path) {
			return fmt.Errorf("worktree path already exists: %s", path)
		}

		conflict, err := wt.FindCaseCollision(parent, leaf)
		if err != nil {
			return fmt.Errorf("check case collision: %w", err)
		}
		if conflict != "" {
			return fmt.Errorf(
				"case-insensitive conflict: %q already exists; this filesystem would treat it as the same path as %q — choose a leaf that doesn't collide",
				conflict, path,
			)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create parent directory: %w", err)
		}

		if err := checkoutWorktree(ctx, path, resolved); err != nil {
			return err
		}

		if !addNoCopy {
			specPath := addCopyFileConfig
			if specPath == "" {
				specPath = filepath.Join(repo.MainRoot, DefaultCopyFile)
			}
			copyEnd := debug.Op("copy-configs", specPath)
			err := copyConfigs(repo.MainRoot, path, specPath)
			copyEnd(err)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: copy configs: %v\n", err)
			}
		}

		warnIfParentNotIgnored(ctx, repo.MainRoot, parent)
		return emitTarget(path)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addParentDir, "parent-dir", "p", "", "parent directory for the worktree (default: <repo>/.gwt/)")
	addCmd.Flags().BoolVar(&addNoCopy, "no-copy", false, "skip copying project config files")
	addCmd.Flags().StringVar(&addCopyFileConfig, "copy-file-config", "", "path to copy spec (default: <repo>/.git-wt-copy-files)")
}

