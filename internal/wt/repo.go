package wt

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/shhac/git-wt/internal/git"
)

// RepoInfo describes the repository the user is currently inside.
type RepoInfo struct {
	// Root is the absolute path to the current worktree's top-level directory.
	Root string
	// MainRoot is the absolute path to the main worktree (== Root when not in a secondary worktree).
	MainRoot string
	// Name is the basename of MainRoot — used for the trees directory name.
	Name string
	// Bare indicates whether the repository is bare (worktrees are not supported).
	Bare bool
}

// Inspect resolves the current repository from dir (use "" for CWD).
func Inspect(ctx context.Context, dir string) (*RepoInfo, error) {
	root, err := git.RunIn(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}
	root, _ = filepath.Abs(root)

	bareOut, err := git.RunIn(ctx, dir, "rev-parse", "--is-bare-repository")
	if err != nil {
		return nil, err
	}
	bare := bareOut == "true"

	commonDir, err := git.RunIn(ctx, dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(root, commonDir)
	}
	mainRoot := filepath.Dir(commonDir) // common dir is `<main>/.git`

	return &RepoInfo{
		Root:     root,
		MainRoot: mainRoot,
		Name:     filepath.Base(mainRoot),
		Bare:     bare,
	}, nil
}

// IsClean reports whether the repository has no in-progress operation
// (merge, rebase, cherry-pick, bisect, revert).
func IsClean(ctx context.Context, dir string) (bool, string, error) {
	commonDir, err := git.RunIn(ctx, dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return false, "", err
	}
	if !filepath.IsAbs(commonDir) {
		root, err := git.RunIn(ctx, dir, "rev-parse", "--show-toplevel")
		if err != nil {
			return false, "", err
		}
		commonDir = filepath.Join(root, commonDir)
	}
	for _, op := range []struct{ marker, name string }{
		{"MERGE_HEAD", "merge"},
		{"rebase-merge", "rebase"},
		{"rebase-apply", "rebase"},
		{"CHERRY_PICK_HEAD", "cherry-pick"},
		{"REVERT_HEAD", "revert"},
		{"BISECT_LOG", "bisect"},
	} {
		if pathExists(filepath.Join(commonDir, op.marker)) {
			return false, op.name, nil
		}
	}
	return true, "", nil
}

// BranchExists reports whether a local branch with this name exists.
// `git show-ref --verify --quiet` exits 0 when the ref exists and 1 (with
// no stderr) when it doesn't, so any error from RunIn means "not found".
func BranchExists(ctx context.Context, dir, name string) (bool, error) {
	_, err := git.RunIn(ctx, dir, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil, nil
}

// TreesDirFor returns the default worktree-parent directory: a `.gwt`
// directory inside the main repo. Users will typically want to add `.gwt/`
// to their .gitignore to keep `git status` clean. Override with
// `git-wt new --parent-dir <path>`.
func TreesDirFor(mainRoot string) string {
	return filepath.Join(mainRoot, ".gwt")
}

// ConstructPath builds the worktree path for a branch under parent (or the
// default trees dir if parent is empty). Branch names may contain slashes —
// they're preserved as subdirectories.
func ConstructPath(parent, branch string) string {
	return filepath.Join(parent, filepath.FromSlash(branch))
}
