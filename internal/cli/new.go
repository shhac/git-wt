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
		exists, err := wt.BranchExists(ctx, "", branch)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("branch %q already exists", branch)
		}

		parent := newParentDir
		if parent == "" {
			parent = wt.TreesDirFor(repo.MainRoot)
		} else {
			parent, err = filepath.Abs(parent)
			if err != nil {
				return err
			}
		}
		path := wt.ConstructPath(parent, branch)
		if wt.PathExists(path) {
			return fmt.Errorf("worktree path already exists: %s", path)
		}

		conflict, err := wt.FindCaseCollision(parent, branch)
		if err != nil {
			return fmt.Errorf("check case collision: %w", err)
		}
		if conflict != "" {
			return fmt.Errorf(
				"case-insensitive conflict: %q already exists; this filesystem would treat it as the same path as %q — choose a branch name that doesn't collide",
				conflict, path,
			)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create parent directory: %w", err)
		}

		if err := createWorktree(ctx, path, branch, newFromRef); err != nil {
			return err
		}

		if !newNoCopy {
			specPath := newCopyFileConfig
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
	rootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVarP(&newParentDir, "parent-dir", "p", "", "parent directory for the worktree (default: <repo>/.gwt/)")
	newCmd.Flags().StringVar(&newFromRef, "from", "", "ref to branch from (default: current HEAD)")
	newCmd.Flags().BoolVar(&newNoCopy, "no-copy", false, "skip copying project config files")
	newCmd.Flags().StringVar(&newCopyFileConfig, "copy-file-config", "", "path to copy spec (default: <repo>/.git-wt-copy-files)")
}
