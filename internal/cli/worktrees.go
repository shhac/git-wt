package cli

import (
	"strings"

	"github.com/shhac/git-wt/internal/wt"
)

// findByBranch returns the worktree whose Branch matches name exactly. If no
// exact match is found, falls back to a unique suffix match (e.g. "auth"
// matches "paul/auth" if it's the only candidate).
func findByBranch(wts []wt.Worktree, name string) *wt.Worktree {
	for i := range wts {
		if wts[i].Branch == name {
			return &wts[i]
		}
	}
	var hits []*wt.Worktree
	for i := range wts {
		b := wts[i].Branch
		if b == "" {
			continue
		}
		if strings.HasSuffix(b, "/"+name) || b == name {
			hits = append(hits, &wts[i])
		}
	}
	if len(hits) == 1 {
		return hits[0]
	}
	return nil
}

// filterOutCurrent returns the list with cur removed. Returns the unchanged
// slice if cur is nil.
func filterOutCurrent(wts []wt.Worktree, cur *wt.Worktree) []wt.Worktree {
	if cur == nil {
		return wts
	}
	out := make([]wt.Worktree, 0, len(wts))
	for _, t := range wts {
		if t.Path != cur.Path {
			out = append(out, t)
		}
	}
	return out
}

// filterRemovable returns wts with the main worktree excluded. The main
// worktree must never be a target of `rm` or `clean`.
func filterRemovable(wts []wt.Worktree, repo *wt.RepoInfo) []wt.Worktree {
	out := make([]wt.Worktree, 0, len(wts))
	for _, t := range wts {
		if t.Path == repo.MainRoot {
			continue
		}
		out = append(out, t)
	}
	return out
}
