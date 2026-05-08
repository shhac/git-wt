package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/copyspec"
	"github.com/shhac/git-wt/internal/git"
	"github.com/shhac/git-wt/internal/wt"
)

// DefaultCopyFile is the conventional location of the per-project copy spec.
// Override with `--copy-file-config <path>`.
const DefaultCopyFile = ".git-wt-copy-files"

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
		if clean, op, err := wt.IsClean(ctx, ""); err != nil {
			return err
		} else if !clean {
			return fmt.Errorf("repository has a %s in progress; complete or abort it first", op)
		}
		if exists, err := wt.BranchExists(ctx, "", branch); err != nil {
			return err
		} else if exists {
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

		if conflict, err := wt.FindCaseCollision(parent, branch); err != nil {
			return fmt.Errorf("check case collision: %w", err)
		} else if conflict != "" {
			return fmt.Errorf(
				"case-insensitive conflict: %q already exists; this filesystem would treat it as the same path as %q. Choose a branch name that doesn't collide.",
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
			if err := copyConfigs(repo.MainRoot, path); err != nil {
				fmt.Fprintf(os.Stderr, "warning: copy configs: %v\n", err)
			}
		}

		return emitTarget(path)
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVarP(&newParentDir, "parent-dir", "p", "", "parent directory for the worktree (default: <repo>-trees/)")
	newCmd.Flags().StringVar(&newFromRef, "from", "", "ref to branch from (default: current HEAD)")
	newCmd.Flags().BoolVar(&newNoCopy, "no-copy", false, "skip copying project config files")
	newCmd.Flags().StringVar(&newCopyFileConfig, "copy-file-config", "", "path to copy spec (default: <repo>/.git-wt-copy-files)")
}

// createWorktree runs `git worktree add` at the given path with a new branch.
func createWorktree(ctx context.Context, path, branch, fromRef string) error {
	args := []string{"worktree", "add", path, "-b", branch}
	if fromRef != "" {
		args = append(args, fromRef)
	}
	_, err := git.Run(ctx, args...)
	return err
}

// copyConfigs loads the copy spec and copies the matching paths into dst.
// Spec resolution: --copy-file-config flag if set, else <repoRoot>/.git-wt-copy-files,
// else built-in defaults (when neither file exists).
func copyConfigs(repoRoot, dst string) error {
	specPath := newCopyFileConfig
	if specPath == "" {
		specPath = filepath.Join(repoRoot, DefaultCopyFile)
	}
	spec, err := copyspec.Load(specPath)
	if err != nil {
		return err
	}
	rels, err := spec.Match(repoRoot)
	if err != nil {
		return err
	}
	for _, rel := range rels {
		src := filepath.Join(repoRoot, rel)
		out := filepath.Join(dst, rel)
		if err := wt.CopyTree(src, out); err != nil {
			return fmt.Errorf("copy %s: %w", rel, err)
		}
	}
	return nil
}
