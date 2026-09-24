package cli

import (
	"fmt"
	"io"
	"strings"
)

// refuseLocked fails the run before anything is deleted when any target is
// locked, naming them all. A lock is someone's explicit "leave this alone",
// separate from the uncommitted-work check that --force answers, so it is
// released with `unlock` rather than overridden here. Checking up front
// matters: git refuses a locked worktree at removal time, which would stop a
// multi-target run after the earlier targets were already gone.
func refuseLocked(targets []rmTarget) error {
	var locked []rmTarget
	for _, t := range targets {
		if t.Locked {
			locked = append(locked, t)
		}
	}
	if len(locked) == 0 {
		return nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d worktree(s) locked:", len(locked))
	for _, t := range locked {
		fmt.Fprintf(&b, "\n    %s (%s)", t.label(), lockStateDetail(t.Worktree))
	}
	b.WriteString("\nrun `git-wt unlock` on them first; nothing was removed")
	return fmt.Errorf("%s", b.String())
}

// skipLocked drops every locked target, saying so. `clean` uses it for the
// same reason it uses skipDirty: it sweeps worktrees the user never named,
// so a lock is respected rather than argued with.
func skipLocked(w io.Writer, targets []rmTarget) []rmTarget {
	kept := make([]rmTarget, 0, len(targets))
	for _, t := range targets {
		if t.Locked {
			_, _ = fmt.Fprintf(w, "skipping %s: %s\n", t.label(), lockStateDetail(t.Worktree))
			continue
		}
		kept = append(kept, t)
	}
	return kept
}
