package wt

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shhac/git-wt/internal/git"
)

// Lock locks the worktree at path, recording reason ("" for none).
func Lock(ctx context.Context, path, reason string) error {
	args := []string{"worktree", "lock"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	_, err := git.Run(ctx, append(args, path)...)
	return err
}

// Unlock removes the lock on the worktree at path.
func Unlock(ctx context.Context, path string) error {
	_, err := git.Run(ctx, "worktree", "unlock", path)
	return err
}

// parseLockReason decodes the value of a porcelain `locked` line. git
// C-quotes a reason containing newlines or other special characters.
func parseLockReason(val string) string {
	if !strings.HasPrefix(val, `"`) {
		return val
	}
	if unquoted, err := strconv.Unquote(val); err == nil {
		return unquoted
	}
	return val
}

// attachLockTimes fills LockedAt for every locked worktree. Porcelain output
// carries the reason but not when the lock was taken; that is the mtime of
// the lock file in the worktree's admin dir, which git writes once and
// deletes on unlock. Best effort: anything unresolvable leaves LockedAt zero.
func attachLockTimes(ctx context.Context, dir string, wts []Worktree) {
	if !anyLocked(wts) {
		return
	}
	commonDir, err := CommonDir(ctx, dir)
	if err != nil {
		return
	}

	times := lockTimes(commonDir)
	for i := range wts {
		if !wts[i].Locked {
			continue
		}
		if at, ok := times[canonicalPath(wts[i].Path)]; ok {
			wts[i].LockedAt = at
		}
	}
}

func anyLocked(wts []Worktree) bool {
	for _, t := range wts {
		if t.Locked {
			return true
		}
	}
	return false
}

// lockTimes maps each locked linked worktree's path to the mtime of its lock
// file. The admin dir's name isn't in porcelain output, so the mapping goes
// through each admin dir's gitdir file, which points at `<worktree>/.git` —
// absolute, or relative to the admin dir under worktree.useRelativePaths.
func lockTimes(commonDir string) map[string]time.Time {
	adminDirs, _ := filepath.Glob(filepath.Join(commonDir, "worktrees", "*"))
	out := make(map[string]time.Time)
	for _, admin := range adminDirs {
		info, err := os.Stat(filepath.Join(admin, "locked"))
		if err != nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(admin, "gitdir"))
		if err != nil {
			continue
		}
		dotGit := strings.TrimSpace(string(raw))
		if !filepath.IsAbs(dotGit) {
			dotGit = filepath.Join(admin, dotGit)
		}
		out[canonicalPath(filepath.Dir(dotGit))] = info.ModTime()
	}
	return out
}

// canonicalPath resolves symlinks (macOS reaches temp dirs via /var ->
// /private/var) so paths from porcelain and from gitdir files compare
// equal. A worktree whose directory is gone can't be resolved whole, so its
// deepest existing ancestor is resolved and the missing tail re-appended.
func canonicalPath(p string) string {
	p = filepath.Clean(p)
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	parent := filepath.Dir(p)
	if parent == p {
		return p
	}
	return filepath.Join(canonicalPath(parent), filepath.Base(p))
}
