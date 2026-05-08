package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/git"
	"github.com/shhac/git-wt/internal/wt"
)

// configFiles are the project-local files copied from the main repo into a
// freshly-created worktree. Glob-style patterns are expanded against the main
// repo root; only files/dirs that exist are copied.
var configFiles = []string{
	".env",
	".env.*",
	".claude",
	"CLAUDE.local.md",
	".ai-cache",
}

var (
	newParentDir string
	newFromRef   string
	newNoCopy    bool
)

var newCmd = &cobra.Command{
	Use:   "new <branch>",
	Short: "Create a new worktree with branch <branch>",
	Long: "Create a new worktree at <repo>-trees/<branch>/, branching from the\n" +
		"current HEAD (or --from <ref>). After creation, copies project-local\n" +
		"config files (.env*, .claude/, CLAUDE.local.md, .ai-cache/) and emits\n" +
		"the worktree path so the wrapper can cd into it.",
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

// copyConfigs copies each entry in configFiles from src to dst (best-effort).
// Glob patterns are expanded.
func copyConfigs(src, dst string) error {
	for _, pat := range configFiles {
		matches, err := filepath.Glob(filepath.Join(src, pat))
		if err != nil {
			return err
		}
		// If the literal name doesn't contain a glob char and didn't match, also
		// try a direct stat — `filepath.Glob` returns no error for nonexistent
		// non-glob paths.
		if len(matches) == 0 && !containsGlobChar(pat) {
			matches = []string{filepath.Join(src, pat)}
		}
		for _, m := range matches {
			if !wt.PathExists(m) {
				continue
			}
			rel, err := filepath.Rel(src, m)
			if err != nil {
				return err
			}
			if err := wt.CopyTree(m, filepath.Join(dst, rel)); err != nil {
				return fmt.Errorf("copy %s: %w", rel, err)
			}
		}
	}
	return nil
}

func containsGlobChar(s string) bool {
	for _, r := range s {
		if r == '*' || r == '?' || r == '[' {
			return true
		}
	}
	return false
}
