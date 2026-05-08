// Package wt models worktrees and parses `git worktree list --porcelain`.
package wt

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shhac/git-wt/internal/git"
)

// Worktree describes a single git worktree as parsed from porcelain output,
// with on-disk metadata attached.
type Worktree struct {
	Path     string    // absolute path to the worktree directory
	HEAD     string    // commit SHA at HEAD
	Branch   string    // short branch name; "" if detached
	Detached bool      // HEAD is detached
	Bare     bool      // bare repository (rare, but porcelain reports it)
	Locked   bool      // worktree is locked
	Prunable bool      // worktree is prunable
	ModTime  time.Time // mtime of the worktree directory; zero if unknown
}

// Display returns the branch name (or "(detached)" / "(bare)" if there isn't one).
func (w Worktree) Display() string {
	switch {
	case w.Bare:
		return "(bare)"
	case w.Detached || w.Branch == "":
		return "(detached)"
	default:
		return w.Branch
	}
}

// DisplayPath returns the human-readable location string shown in list/picker
// rows. Rules (in priority order):
//
//  1. Main worktree         → repo basename            (e.g. "backend")
//  2. Inside treesDir       → "#" + rel-to-treesDir    (e.g. "#abuja",
//     "#paul/feat/aaa")
//  3. Inside repo (outside) → rel-to-repo              (e.g. ".conductor/abuja")
//  4. Outside repo          → absolute path            (e.g. "/tmp/dddd")
//
// All slashes are normalized to forward slashes for cross-platform consistency.
func (w Worktree) DisplayPath(mainRoot, treesDir string) string {
	if w.Path == mainRoot {
		return filepath.Base(mainRoot)
	}
	if rel, ok := relUnder(treesDir, w.Path); ok {
		return "#" + rel
	}
	if rel, ok := relUnder(mainRoot, w.Path); ok {
		return rel
	}
	return w.Path
}

// relUnder returns the path of target relative to base (slash-separated)
// and ok=true iff target is at or under base. The "or under" check requires
// the result to not start with "..", which filepath.Rel emits for upward
// traversal.
func relUnder(base, target string) (string, bool) {
	if base == "" || target == "" {
		return "", false
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", false
	}
	if rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// List returns all worktrees for the repository containing dir.
// Pass "" to use the current working directory.
func List(ctx context.Context, dir string) ([]Worktree, error) {
	out, err := git.RunIn(ctx, dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	wts := parsePorcelain(out)
	for i := range wts {
		if info, err := os.Stat(wts[i].Path); err == nil {
			wts[i].ModTime = info.ModTime()
		}
	}
	return wts, nil
}

// SortByModTime sorts in place, most-recent first. Worktrees with zero ModTime sort last.
func SortByModTime(wts []Worktree) {
	sort.SliceStable(wts, func(i, j int) bool {
		zi := wts[i].ModTime.IsZero()
		zj := wts[j].ModTime.IsZero()
		if zi {
			return false // zero on the left always loses (covers zi&&zj too)
		}
		if zj {
			return true
		}
		return wts[i].ModTime.After(wts[j].ModTime)
	})
}

// Current returns the worktree containing dir (the user's CWD by default), if any.
// Uses path comparison rather than git resolution so callers can run from any subdir.
func Current(wts []Worktree, dir string) *Worktree {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}
	// EvalSymlinks fails for non-existent paths (returns "" + err). Fall back
	// to the un-resolved absolute path so we still match worktrees correctly
	// from a directory the user is just about to create.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}

	// Pick the deepest match — handles nested worktrees (the Bug #36 / v0.6.2 case).
	var best *Worktree
	bestLen := -1
	for i := range wts {
		wp, err := filepath.EvalSymlinks(wts[i].Path)
		if err != nil {
			wp = wts[i].Path
		}
		if abs == wp || strings.HasPrefix(abs, wp+string(filepath.Separator)) {
			if len(wp) > bestLen {
				best = &wts[i]
				bestLen = len(wp)
			}
		}
	}
	return best
}

func parsePorcelain(out string) []Worktree {
	var wts []Worktree
	var cur *Worktree
	flush := func() {
		if cur != nil {
			wts = append(wts, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			flush()
			continue
		}
		key, val, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			flush()
			cur = &Worktree{Path: val}
		case "HEAD":
			if cur != nil {
				cur.HEAD = val
			}
		case "branch":
			if cur != nil {
				cur.Branch = strings.TrimPrefix(val, "refs/heads/")
			}
		case "detached":
			if cur != nil {
				cur.Detached = true
			}
		case "bare":
			if cur != nil {
				cur.Bare = true
			}
		case "locked":
			if cur != nil {
				cur.Locked = true
			}
		case "prunable":
			if cur != nil {
				cur.Prunable = true
			}
		}
	}
	flush()
	return wts
}
