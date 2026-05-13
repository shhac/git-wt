package cli

// Shared helpers for the worktree-creation commands (new, add). These
// commands have nearly identical pipelines; the helpers here are the
// pieces both call. Command-specific orchestration stays in new.go and
// add.go.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shhac/git-wt/internal/copyspec"
	"github.com/shhac/git-wt/internal/debug"
	"github.com/shhac/git-wt/internal/git"
	"github.com/shhac/git-wt/internal/wt"
)

// DefaultCopyFile is the conventional location of the per-project copy spec.
// Override with `--copy-file-config <path>`.
const DefaultCopyFile = ".git-wt-copy-files"

// requireMutableRepo returns the current repo info on success. It refuses
// when there is no git repository, when the repo is bare, or when a
// merge/rebase/cherry-pick/etc. is in progress. Used by `new` and `add`
// to enforce shared preconditions before any worktree mutation.
func requireMutableRepo(ctx context.Context) (*wt.RepoInfo, error) {
	repo, err := wt.Inspect(ctx, "")
	if err != nil {
		return nil, err
	}
	if repo.Bare {
		return nil, fmt.Errorf("cannot create worktrees in a bare repository")
	}
	clean, op, err := wt.IsClean(ctx, "")
	if err != nil {
		return nil, err
	}
	if !clean {
		return nil, fmt.Errorf("repository has a %s in progress; complete or abort it first", op)
	}
	return repo, nil
}

// defaultedLeaf picks the leaf for a worktree-creation command: the
// caller's override if non-empty, otherwise the fallback. Either way
// the chosen value is run through wt.ValidateBranchName so callers
// don't repeat the wrap.
func defaultedLeaf(override, fallback string) (string, error) {
	leaf := override
	if leaf == "" {
		leaf = fallback
	}
	if err := wt.ValidateBranchName(leaf); err != nil {
		return "", fmt.Errorf("invalid leaf %q: %w", leaf, err)
	}
	return leaf, nil
}

// prepareWorktreeSite computes the destination path under parent for the
// given leaf, refuses if the path is already in use, runs the case-
// insensitive collision check, and creates the parent directory chain.
// Returns the resolved path on success.
//
// noun ("branch name" / "leaf") is used in the collision error so the
// suggestion matches the positional argument vocabulary of the caller.
func prepareWorktreeSite(parent, leaf, noun string) (string, error) {
	path := wt.ConstructPath(parent, leaf)
	if wt.PathExists(path) {
		return "", fmt.Errorf("worktree path already exists: %s", path)
	}
	conflict, err := wt.FindCaseCollision(parent, leaf)
	if err != nil {
		return "", fmt.Errorf("check case collision: %w", err)
	}
	if conflict != "" {
		return "", fmt.Errorf(
			"case-insensitive conflict: %q already exists; this filesystem would treat it as the same path as %q — choose a %s that doesn't collide",
			conflict, path, noun,
		)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create parent directory: %w", err)
	}
	return path, nil
}

// finalizeWorktreeSite runs the post-create tail shared by new and add:
// optionally copy project config files, warn when parent isn't gitignored,
// then emit the destination path via the wrapper protocol. A copy-config
// failure is non-fatal (logged as a warning); the worktree still ships.
func finalizeWorktreeSite(ctx context.Context, repo *wt.RepoInfo, path, parent string, noCopy bool, copyFileConfig string) error {
	if !noCopy {
		specPath := copyFileConfig
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
}

// createWorktree runs `git worktree add -b <branch> [<fromRef>]` — used by
// the `new` command to materialise a fresh branch into a new worktree.
func createWorktree(ctx context.Context, path, branch, fromRef string) error {
	args := []string{"worktree", "add", path, "-b", branch}
	if fromRef != "" {
		args = append(args, fromRef)
	}
	_, err := git.Run(ctx, args...)
	return err
}

// checkoutWorktree runs `git worktree add` for an already-resolved ref.
// The args-construction (local vs remote-tracking) is encapsulated on
// the resolution itself; see wt.AddRefResolution.WorktreeAddArgs.
func checkoutWorktree(ctx context.Context, path string, ref *wt.AddRefResolution) error {
	_, err := git.Run(ctx, ref.WorktreeAddArgs(path)...)
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

// relInsideRepo returns the path of parentDir relative to mainRoot and
// whether parentDir lives at or under mainRoot.
//
// Symlinks are resolved on both sides so /tmp vs /private/tmp on macOS
// (or any other symlink-bridged paths) don't throw off the comparison.
// parentDir == mainRoot returns (_, false) — we don't want to tell the
// user to gitignore their own working tree.
func relInsideRepo(mainRoot, parentDir string) (string, bool) {
	if r, err := filepath.EvalSymlinks(mainRoot); err == nil {
		mainRoot = r
	}
	if r, err := filepath.EvalSymlinks(parentDir); err == nil {
		parentDir = r
	}
	rel, err := filepath.Rel(mainRoot, parentDir)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return rel, true
}

// isPathIgnored asks `git check-ignore --quiet` whether rel (relative to
// mainRoot) is matched by any .gitignore rule. check-ignore exits 0 when
// matched, 1 when not matched, and >1 on real errors. We map the first
// two to (matched, nil) and bubble the rest up so the caller can stay
// silent on unrelated git failures.
func isPathIgnored(ctx context.Context, mainRoot, rel string) (bool, error) {
	args := []string{"check-ignore", "--quiet", rel}
	end := debug.Op("git", args)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = mainRoot
	err := cmd.Run()
	end(err)
	if err == nil {
		return true, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// warnIfParentNotIgnored prints a one-line stderr hint when parentDir
// lives inside the main repo but isn't covered by any .gitignore rule.
// Worktrees inside an unignored path show up as untracked content in
// `git status` on the main worktree — almost never what the user wants.
func warnIfParentNotIgnored(ctx context.Context, mainRoot, parentDir string) {
	rel, inside := relInsideRepo(mainRoot, parentDir)
	if !inside {
		return
	}
	ignored, err := isPathIgnored(ctx, mainRoot, rel)
	if err != nil || ignored {
		return
	}
	rel = filepath.ToSlash(rel)
	fmt.Fprintf(os.Stderr, "note: %s/ is not in .gitignore — `git status` in the main worktree will show worktree contents as untracked\n", rel)
	fmt.Fprintf(os.Stderr, "      add `%s/` to .gitignore to silence this\n", rel)
}
