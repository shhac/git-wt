package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-isatty"

	"github.com/shhac/git-wt/internal/debug"
	"github.com/shhac/git-wt/internal/git"
	"github.com/shhac/git-wt/internal/wt"
)

// removeWorktree removes one registered worktree. The fast path mirrors
// git's own cleanliness check, renames the directory aside so the path frees
// up immediately, unregisters it, then deletes the renamed directory with
// wt.DeleteTree — which fixes read-only directories and immutable file flags
// that make `git worktree remove` die mid-delete and strand a half-removed
// tree. Locked worktrees and anything the fast path can't establish fall
// back to plain `git worktree remove`, preserving git's own errors.
func removeWorktree(ctx context.Context, t wt.Worktree, force bool) error {
	if !t.Locked {
		if handled, err := fastRemoveWorktree(ctx, t, force); handled {
			return err
		}
	}

	args := []string{"worktree", "remove", t.Path}
	if force {
		args = append(args, "--force")
	}
	_, err := git.Run(ctx, args...)
	return err
}

// fastRemoveWorktree attempts the rename-aside removal. handled=false means
// nothing was changed and the caller should fall back to `git worktree
// remove`; handled=true means the outcome (success or error) is final.
func fastRemoveWorktree(ctx context.Context, t wt.Worktree, force bool) (handled bool, err error) {
	end := debug.Op("rm.fast", t.Display())
	defer func() { end(err) }()

	if !force {
		dirty, derr := wt.IsWorkingTreeDirty(ctx, t.Path)
		if derr != nil {
			return false, nil // can't establish safety; let git decide
		}
		if dirty {
			return true, fmt.Errorf("%q contains modified or untracked files (use --force to remove anyway)", t.Display())
		}
	}

	aside, rerr := wt.RenameAside(t.Path)
	if rerr != nil {
		return false, nil // e.g. directory already missing; let git report it
	}

	// With the directory gone from its registered path, prune drops the
	// worktree from git's records (verified: immediate, no expiry involved).
	if _, perr := git.Run(ctx, "worktree", "prune"); perr != nil {
		if rbErr := os.Rename(aside, t.Path); rbErr == nil {
			return false, nil // fully restored; fall back
		}
		return true, fmt.Errorf("unregister failed (%v); the worktree directory is now at %s — move it back or delete it manually", perr, aside)
	}

	if derr := deleteTreeWithProgress(t.Display(), aside); derr != nil {
		return true, fmt.Errorf("worktree is unregistered but its directory could not be fully deleted (%w); remove %s manually", derr, aside)
	}
	return true, nil
}

// orphanRmTarget resolves arg against leftover directories: a directory
// under the trees dir that exists on disk but isn't any registered
// worktree's path. Lets `rm` rescue the debris of a removal that died
// mid-delete after git had already unregistered the worktree.
func orphanRmTarget(wts []wt.Worktree, treesDir, arg string) (rmTarget, bool) {
	if treesDir == "" {
		return rmTarget{}, false
	}
	p := filepath.Join(treesDir, filepath.FromSlash(arg))
	if !strings.HasPrefix(p, treesDir+string(filepath.Separator)) {
		return rmTarget{}, false // absolute or ../ args must not escape
	}
	info, err := os.Stat(p)
	if err != nil || !info.IsDir() {
		return rmTarget{}, false
	}
	for _, w := range wts {
		if w.Path == p {
			return rmTarget{}, false // a live worktree; branch-based flow owns it
		}
	}
	return rmTarget{Worktree: wt.Worktree{Path: p}, orphan: true}, true
}

// deleteTreeWithProgress deletes dir via wt.DeleteTree. When stderr is a
// TTY it repaints a single progress line (count + elapsed) so large
// deletions don't look like a hang; non-TTY runs stay silent until the
// caller's final "removed" line.
func deleteTreeWithProgress(label, dir string) error {
	if !isatty.IsTerminal(os.Stderr.Fd()) {
		return wt.DeleteTree(dir, nil)
	}

	start := time.Now()
	width := 0
	// Cadence comes from DeleteTree's progress ticker (~100ms); paints are
	// already serialized (ticker goroutine, then one final call after it
	// stops), so no throttling or locking needed here.
	paint := func(n int) {
		line := fmt.Sprintf("removing %s… %d files (%.1fs)", label, n, time.Since(start).Seconds())
		if pad := width - len(line); pad > 0 {
			line += strings.Repeat(" ", pad)
		} else {
			width = len(line)
		}
		fmt.Fprintf(os.Stderr, "\r%s", line)
	}

	err := wt.DeleteTree(dir, paint)
	if width > 0 {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", width))
	}
	return err
}
