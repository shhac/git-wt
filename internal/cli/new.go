package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/wt"
)

var (
	newParentDir      string
	newFromRef        string
	newNoCopy         bool
	newCopyFileConfig string
)

var newCmd = &cobra.Command{
	Use:   "new <branch>",
	Short: "Create a new worktree with branch <branch>",
	Long: "Create a new worktree at <repo>-trees/<branch>/, branching from the\n" +
		"current HEAD (or --from <ref>). After creation, copies project-local\n" +
		"files according to <repo>/.git-wt-copy-files (override path with\n" +
		"--copy-file-config). Built-in defaults are used when no spec file\n" +
		"is present. Skip the copy entirely with --no-copy.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		branch := args[0]

		if err := wt.ValidateBranchName(branch); err != nil {
			return err
		}

		repo, err := requireMutableRepo(ctx)
		if err != nil {
			return err
		}
		exists, err := wt.BranchExists(ctx, "", branch)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("branch %q already exists", branch)
		}

		parent, err := resolveParentDir(ctx, repo.MainRoot, newParentDir)
		if err != nil {
			return err
		}
		path, err := prepareWorktreeSite(parent, branch, "branch name")
		if err != nil {
			return err
		}

		if err := createWorktree(ctx, path, branch, newFromRef); err != nil {
			return err
		}
		return finalizeWorktreeSite(ctx, repo, path, parent, newNoCopy, newCopyFileConfig)
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVarP(&newParentDir, "parent-dir", "p", "", "parent directory for the worktree (default: <repo>/.worktrees/)")
	newCmd.Flags().StringVar(&newFromRef, "from", "", "ref to branch from (default: current HEAD)")
	newCmd.Flags().BoolVar(&newNoCopy, "no-copy", false, "skip copying project config files")
	newCmd.Flags().StringVar(&newCopyFileConfig, "copy-file-config", "", "path to copy spec (default: <repo>/.git-wt-copy-files)")
}
