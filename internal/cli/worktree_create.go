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
// For a local branch this is a plain checkout. For a remote-tracking ref
// we pass `--track -b <localName>` explicitly, otherwise `git worktree add`
// treats `origin/feature` as a detached commit-ish and skips creating a
// local tracking branch (DWIM only fires when the start-point is a bare
// branch name that doesn't yet exist locally).
func checkoutWorktree(ctx context.Context, path string, ref *wt.AddRefResolution) error {
	var args []string
	switch ref.Kind {
	case wt.AddRefRemote:
		args = []string{"worktree", "add", "--track", "-b", ref.LocalName, path, ref.SourceRef}
	default:
		args = []string{"worktree", "add", path, ref.SourceRef}
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
	checkArgs := []string{"check-ignore", "--quiet", rel}
	end := debug.Op("git", checkArgs)
	cmd := exec.CommandContext(ctx, "git", checkArgs...)
	cmd.Dir = mainRoot
	err = cmd.Run()
	// exit 1 is expected (not ignored); the timeline records it as failed but
	// it's the path we act on rather than treat as an error.
	end(err)
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
