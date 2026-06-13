package wt

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/shhac/git-wt/internal/config"
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
	if op := inProgressOp(commonDir); op != "" {
		return false, op, nil
	}
	return true, "", nil
}

// inProgressOp returns the name of a git operation in progress at commonDir
// (the path of `.git` for the main worktree), or "" if the repo is clean.
// Pure: only stats well-known marker files. Order is fixed so tests stay
// deterministic when multiple markers are present (shouldn't happen in
// practice — git refuses to start a second operation while one is active).
func inProgressOp(commonDir string) string {
	markers := []struct{ file, name string }{
		{"MERGE_HEAD", "merge"},
		{"rebase-merge", "rebase"},
		{"rebase-apply", "rebase"},
		{"CHERRY_PICK_HEAD", "cherry-pick"},
		{"REVERT_HEAD", "revert"},
		{"BISECT_LOG", "bisect"},
	}
	for _, m := range markers {
		if pathExists(filepath.Join(commonDir, m.file)) {
			return m.name
		}
	}
	return ""
}

// BranchExists reports whether a local branch with this name exists.
// `git show-ref --verify --quiet` exits 0 when the ref exists and 1 (with
// no stderr) when it doesn't, so any error from RunIn means "not found".
func BranchExists(ctx context.Context, dir, name string) (bool, error) {
	_, err := git.RunIn(ctx, dir, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil, nil
}

// CurrentBranch returns the short branch name HEAD points at, or an
// error if HEAD is detached. Useful for commands that operate on the
// "branch you're currently on" rather than a user-supplied ref.
func CurrentBranch(ctx context.Context) (string, error) {
	out, err := git.Run(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("HEAD is detached or not on a branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// IsWorkingTreeDirty reports whether `git status --porcelain` shows any
// modifications — tracked changes, staged changes, or untracked files.
// dir selects the worktree to check ("" = current working directory).
// Distinct from [IsClean], which checks for an in-progress operation
// (merge / rebase / cherry-pick / etc.) via marker files.
func IsWorkingTreeDirty(ctx context.Context, dir string) (bool, error) {
	out, err := git.RunIn(ctx, dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// DetectBaseBranch returns the branch a command should treat as "the
// trunk" — either the caller-supplied override (which must exist
// locally) or, when override is empty, the first of "main" or "master"
// that exists. Returns an error if nothing matches.
func DetectBaseBranch(ctx context.Context, override string) (string, error) {
	if override != "" {
		exists, err := BranchExists(ctx, "", override)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("base branch %q does not exist locally", override)
		}
		return override, nil
	}
	for _, candidate := range []string{"main", "master"} {
		exists, err := BranchExists(ctx, "", candidate)
		if err != nil {
			return "", err
		}
		if exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no `main` or `master` branch found; pass an explicit base")
}

// TreesDirFor returns the default worktree-parent directory: a `.worktrees`
// directory inside the main repo. Users will typically want to add `.worktrees/`
// to their .gitignore to keep `git status` clean. Override with
// `git-wt new --parent-dir <path>`.
func TreesDirFor(mainRoot string) string {
	return filepath.Join(mainRoot, ".worktrees")
}

// ResolveParentDir returns the worktree-parent directory for a creation
// command. An empty override yields TreesDirFor(mainRoot). Otherwise:
//
//   - ${...} template variables are expanded against mainRoot (see
//     internal/config.ExpandPath for the vocabulary). Unknown vars
//     error rather than landing in the resulting path silently.
//   - Relative paths (post-expansion) are joined against mainRoot, not
//     the caller's CWD — running `--parent-dir trees` from a subdir
//     should still resolve relative to the repo, which is what users
//     mean.
func ResolveParentDir(mainRoot, override string) (string, error) {
	if override == "" {
		return TreesDirFor(mainRoot), nil
	}
	expanded, err := config.ExpandPath(override, config.VarsFor(mainRoot))
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(mainRoot, expanded)
	}
	return filepath.Clean(expanded), nil
}

// ConstructPath builds the worktree path for a branch under parent (or the
// default trees dir if parent is empty). Branch names may contain slashes —
// they're preserved as subdirectories.
func ConstructPath(parent, branch string) string {
	return filepath.Join(parent, filepath.FromSlash(branch))
}
