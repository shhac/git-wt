package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
			specPath := newCopyFileConfig
			if specPath == "" {
				specPath = filepath.Join(repo.MainRoot, DefaultCopyFile)
			}
			if err := copyConfigs(repo.MainRoot, path, specPath); err != nil {
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

// warnIfParentNotIgnored prints a one-line stderr hint when parentDir lives
// inside the main repo but isn't covered by any .gitignore rule. Worktrees
// inside an unignored path show up as untracked content in `git status` on
// the main worktree, which is almost always not what the user wants.
//
// Silent when:
//   - parentDir is outside the repo (no concern)
//   - parentDir resolves to the repo root itself (we'd be telling the user
//     to ignore their own working tree)
//   - `git check-ignore` reports the path as ignored (exit 0)
//   - any other git error (we don't want to be noisy on edge cases)
func warnIfParentNotIgnored(ctx context.Context, mainRoot, parentDir string) {
	// Resolve symlinks on both sides so /tmp vs /private/tmp on macOS doesn't
	// throw off the relative-path comparison. Fall back to the input on error.
	if r, err := filepath.EvalSymlinks(mainRoot); err == nil {
		mainRoot = r
	}
	if r, err := filepath.EvalSymlinks(parentDir); err == nil {
		parentDir = r
	}
	rel, err := filepath.Rel(mainRoot, parentDir)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return
	}
	// `git check-ignore`:
	//   exit 0 → at least one path is ignored
	//   exit 1 → none of the paths are ignored (this is the case we warn on)
	//   other  → an actual error; stay silent
	cmd := exec.CommandContext(ctx, "git", "check-ignore", "--quiet", rel)
	cmd.Dir = mainRoot
	err = cmd.Run()
	if err == nil {
		return
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 1 {
		return
	}
	rel = filepath.ToSlash(rel)
	fmt.Fprintf(os.Stderr, "note: %s/ is not in .gitignore — `git status` in the main worktree will show worktree contents as untracked\n", rel)
	fmt.Fprintf(os.Stderr, "      add `%s/` to .gitignore to silence this\n", rel)
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

// copyConfigs loads the copy spec at specPath (falling back to built-in
// defaults when the file is absent) and copies the matching paths from
// repoRoot into dst.
func copyConfigs(repoRoot, dst, specPath string) error {
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
