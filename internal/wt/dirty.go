package wt

import (
	"context"
	"fmt"
	"strings"

	"github.com/shhac/git-wt/internal/git"
)

// DirtyStat counts the uncommitted entries in a working tree, split so
// callers can tell "four files edited" from "one stray build artifact"
// when warning about work they are about to destroy.
type DirtyStat struct {
	Modified  int // tracked files with staged or unstaged changes
	Untracked int
}

// Any reports whether the working tree has anything uncommitted.
func (d DirtyStat) Any() bool { return d.Modified+d.Untracked > 0 }

// Summary renders the counts for a prompt or warning line, e.g.
// "4 modified, 1 untracked". Empty when nothing is dirty.
func (d DirtyStat) Summary() string {
	var parts []string
	if d.Modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", d.Modified))
	}
	if d.Untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", d.Untracked))
	}
	return strings.Join(parts, ", ")
}

// WorkingTreeStatus counts what `git status --porcelain` reports — tracked
// changes, staged changes, and untracked files. dir selects the worktree to
// check ("" = current working directory). Distinct from [IsClean], which
// checks for an in-progress operation (merge / rebase / cherry-pick / etc.)
// via marker files.
func WorkingTreeStatus(ctx context.Context, dir string) (DirtyStat, error) {
	out, err := git.RunIn(ctx, dir, "status", "--porcelain")
	if err != nil {
		return DirtyStat{}, err
	}
	_, headErr := git.RunIn(ctx, dir, "rev-parse", "--verify", "--quiet", "HEAD")
	return parseDirtyStat(out, headErr != nil), nil
}

// parseDirtyStat counts porcelain v1 lines. Every entry is two status
// characters then a space then the path: column one is index-vs-HEAD,
// column two is worktree-vs-index, and "??" marks untracked.
//
// unbornHEAD drops column one from the reckoning. With no resolvable HEAD —
// a fresh repo, or a worktree whose branch ref was deleted out from under it,
// which is exactly what `clean` hunts for — git compares the index against
// nothing and reports every tracked file as a staged addition. Reading that
// as uncommitted work would mark every orphaned worktree dirty. Column two
// still tells the truth in that state, so real edits are not lost.
func parseDirtyStat(porcelain string, unbornHEAD bool) DirtyStat {
	var d DirtyStat
	for _, line := range strings.Split(porcelain, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "??") {
			d.Untracked++
			continue
		}
		if len(line) < 2 {
			continue
		}
		if unbornHEAD && line[1] == ' ' {
			continue
		}
		d.Modified++
	}
	return d
}

// IsWorkingTreeDirty reports whether the working tree has any uncommitted
// changes. See [WorkingTreeStatus] for the breakdown.
func IsWorkingTreeDirty(ctx context.Context, dir string) (bool, error) {
	d, err := WorkingTreeStatus(ctx, dir)
	if err != nil {
		return false, err
	}
	return d.Any(), nil
}
