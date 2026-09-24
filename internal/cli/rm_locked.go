package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// resolveLocked decides what to do about locked targets, before anything is
// deleted — git refuses a locked worktree only at removal time, which would
// stop a multi-target run after earlier targets were already gone.
//
// A lock is someone's explicit "leave this alone", a separate question from
// the uncommitted work --force answers, so --force doesn't override it.
// Non-interactive refuses and names every locked target; interactive shows
// each lock's age and reason and asks which to unlock and remove anyway.
// Returns (nil, nil) when the user cancels.
func resolveLocked(targets []rmTarget) ([]rmTarget, error) {
	locked := lockedTargets(targets)
	if len(locked) == 0 {
		return targets, nil
	}
	if !interactive() {
		return nil, lockedBailError(locked)
	}

	title := fmt.Sprintf(
		"%d of %d selected worktrees are locked.\n"+
			"Select any you want to unlock and remove; unselected are skipped.\n"+
			"(space toggles, enter continues, esc cancels)",
		len(locked), len(targets),
	)
	chosen, ok, err := pickTargetSubset(title, locked, func(t rmTarget) string { return lockStateDetail(t.Worktree) })
	if err != nil {
		return nil, err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "cancelled")
		return nil, nil
	}
	for _, t := range locked {
		if !chosen[t.Path] {
			reportSkippedLocked(os.Stderr, t)
		}
	}
	kept := applyUnlockChoice(targets, chosen)
	if len(kept) == 0 {
		fmt.Fprintln(os.Stderr, "nothing left to remove")
	}
	return kept, nil
}

func lockedTargets(targets []rmTarget) []rmTarget {
	var out []rmTarget
	for _, t := range targets {
		if t.Locked {
			out = append(out, t)
		}
	}
	return out
}

// applyUnlockChoice keeps every unlocked target plus the locked ones whose
// path is in chosen, marked to be unlocked just before their removal — not
// now, so cancelling at the confirm step leaves every lock in place.
func applyUnlockChoice(targets []rmTarget, chosen map[string]bool) []rmTarget {
	out := make([]rmTarget, 0, len(targets))
	for _, t := range targets {
		if !t.Locked {
			out = append(out, t)
			continue
		}
		if chosen[t.Path] {
			t.unlock = true
			out = append(out, t)
		}
	}
	return out
}

// lockedBailError names every locked target and the command that releases
// them, so a non-interactive caller learns the whole picture in one run.
func lockedBailError(locked []rmTarget) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%d worktree(s) locked:", len(locked))
	for _, t := range locked {
		fmt.Fprintf(&b, "\n    %s (%s)", t.label(), lockStateDetail(t.Worktree))
	}
	b.WriteString("\n" + unlockHint(locked) + "; nothing was removed")
	return errors.New(b.String())
}

// unlockHint is the command to run before retrying. git-wt unlock resolves
// branches; a branchless worktree is named by path, which only git takes.
func unlockHint(locked []rmTarget) string {
	branches := make([]string, 0, len(locked))
	for _, t := range locked {
		if t.Branch == "" {
			return "unlock them first with `git worktree unlock <path>`"
		}
		branches = append(branches, t.Branch)
	}
	return "unlock them first with `git-wt unlock " + strings.Join(branches, " ") + "`"
}

func reportSkippedLocked(w io.Writer, t rmTarget) {
	_, _ = fmt.Fprintf(w, "skipping %s: %s\n", t.label(), lockStateDetail(t.Worktree))
}

// skipLocked drops every locked target, saying so. `clean` uses it for the
// same reason it uses skipDirty: it sweeps worktrees the user never named,
// so a lock is respected rather than argued with.
func skipLocked(w io.Writer, targets []rmTarget) []rmTarget {
	kept := make([]rmTarget, 0, len(targets))
	for _, t := range targets {
		if t.Locked {
			reportSkippedLocked(w, t)
			continue
		}
		kept = append(kept, t)
	}
	return kept
}
