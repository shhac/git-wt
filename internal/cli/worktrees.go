package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/shhac/git-wt/internal/wt"
)

// loadWorktrees runs the standard list-then-sort-then-resolve-current pipeline
// shared by list, go, rm, and clean. Centralising it means future changes
// (e.g. a different sort criterion or a symlink-aware CWD lookup) only touch
// one place.
func loadWorktrees(ctx context.Context) ([]wt.Worktree, *wt.Worktree, error) {
	wts, err := wt.List(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	wt.SortByModTime(wts)
	cur := wt.Current(wts, mustWD())
	return wts, cur, nil
}

// loadRepoAndWorktrees combines wt.Inspect with loadWorktrees — the opening
// move shared by go, list, rm, and the completion helpers.
func loadRepoAndWorktrees(ctx context.Context) (*wt.RepoInfo, []wt.Worktree, *wt.Worktree, error) {
	repo, err := wt.Inspect(ctx, "")
	if err != nil {
		return nil, nil, nil, err
	}
	wts, cur, err := loadWorktrees(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	return repo, wts, cur, nil
}

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

// findByTreesDirLeaf resolves a name against the leaf directory of a
// registered worktree inside the trees dir. A detached worktree has no branch
// for findByBranch to match, but `list` shows it as `#leaf` and that is the
// only handle the user has for it.
func findByTreesDirLeaf(wts []wt.Worktree, treesDir, name string) *wt.Worktree {
	if treesDir == "" || name == "" {
		return nil
	}
	want := filepath.Join(treesDir, filepath.FromSlash(name))
	if !strings.HasPrefix(want, treesDir+string(filepath.Separator)) {
		return nil // absolute or ../ args must not escape
	}
	for i := range wts {
		if wts[i].Path == want {
			return &wts[i]
		}
	}
	return nil
}

// findNamedWorktree resolves a worktree named on the command line: by branch
// (exact, then unique suffix), then by leaf directory under the trees dir.
// (nil, nil) means nothing matched, which callers handle differently: rm
// goes on to look for a leftover directory. Naming the main worktree is an
// error — no command that takes names can act on it — and verb says which
// command refused.
func findNamedWorktree(wts []wt.Worktree, repo *wt.RepoInfo, treesDir, name, verb string) (*wt.Worktree, error) {
	t := findByBranch(wts, name)
	if t == nil {
		t = findByTreesDirLeaf(wts, treesDir, name)
	}
	if t == nil {
		return nil, nil
	}
	if t.Path == repo.MainRoot {
		return nil, fmt.Errorf("cannot %s the main worktree (%q)", verb, t.Display())
	}
	return t, nil
}

// worktreeLabel is the human-readable name used in prompts and messages.
func worktreeLabel(t wt.Worktree) string {
	if t.Branch == "" {
		// Display() collapses to "(detached)" or "(bare)" for every such
		// worktree, so the path is the only thing that tells two of them
		// apart.
		return t.Display() + " " + t.Path
	}
	return t.Display()
}
