package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shhac/git-wt/internal/debug"
	"github.com/shhac/git-wt/internal/picker"
	"github.com/shhac/git-wt/internal/wt"
)

// forceReason records why a worktree with uncommitted changes is being
// removed anyway. It decides whether executeRm warns about the work it is
// discarding: someone who picked the worktree out of the dirty prompt has
// just read the same counts, so repeating them is noise.
type forceReason int

const (
	forceNone   forceReason = iota // clean, or not being forced
	forceFlag                      // --force on the command line
	forceChosen                    // explicitly toggled in the dirty prompt
)

// scanDirty annotates every target with its uncommitted-file counts, before
// anything is deleted. Orphans are skipped — an unregistered leftover has no
// worktree for git to report on. A worktree whose status can't be read is
// left marked clean; the re-check inside removeWorktree is the backstop, and
// it defers to git when it can't establish safety either.
func scanDirty(ctx context.Context, targets []rmTarget) []rmTarget {
	end := debug.Op("rm.scan-dirty", fmt.Sprintf("%d-target(s)", len(targets)))
	defer func() { end(nil) }()

	out := make([]rmTarget, len(targets))
	copy(out, targets)
	for i := range out {
		if out[i].orphan {
			continue
		}
		stat, err := wt.WorkingTreeStatus(ctx, out[i].Path)
		if err != nil {
			debug.Logf("status %s: %v (treating as clean)", out[i].Path, err)
			continue
		}
		out[i].dirty = stat
	}
	return out
}

// dirtyTargets returns the subset with uncommitted changes, in order.
func dirtyTargets(targets []rmTarget) []rmTarget {
	var out []rmTarget
	for _, t := range targets {
		if t.dirty.Any() {
			out = append(out, t)
		}
	}
	return out
}

// applyDirtyChoice keeps every clean target plus the dirty ones whose path
// appears in forced, marking those with reason. Dirty targets absent from
// forced are dropped. Order is preserved.
func applyDirtyChoice(targets []rmTarget, forced map[string]bool, reason forceReason) []rmTarget {
	out := make([]rmTarget, 0, len(targets))
	for _, t := range targets {
		if !t.dirty.Any() {
			out = append(out, t)
			continue
		}
		if forced[t.Path] {
			t.force = reason
			out = append(out, t)
		}
	}
	return out
}

// forceAll marks every path in targets, for the --force path where the user
// has already opted into destroying whatever is there.
func forceAll(targets []rmTarget) map[string]bool {
	out := make(map[string]bool, len(targets))
	for _, t := range targets {
		out[t.Path] = true
	}
	return out
}

// dirtyBailError names every dirty target in one error, so a
// non-interactive caller learns the full picture from a single run instead
// of discovering them one re-run at a time.
func dirtyBailError(dirty []rmTarget) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%d worktree(s) have uncommitted changes:", len(dirty))
	for _, t := range dirty {
		fmt.Fprintf(&b, "\n    %s (%s)", t.label(), t.dirty.Summary())
	}
	b.WriteString("\nuse --force to remove them anyway")
	return fmt.Errorf("%s", b.String())
}

// dirtyRows renders the multi-select rows for pickDirtyToForce, padding the
// labels so the counts line up.
func dirtyRows(dirty []rmTarget) []picker.Row {
	width := 0
	for _, t := range dirty {
		if n := len(t.label()); n > width {
			width = n
		}
	}
	rows := make([]picker.Row, len(dirty))
	for i, t := range dirty {
		rows[i] = picker.Row{
			Display: fmt.Sprintf("%-*s  %s", width, t.label(), t.dirty.Summary()),
			Value:   t.Path,
		}
	}
	return rows
}

// pickDirtyToForce asks which dirty worktrees to remove anyway. Toggling
// nothing — the default — skips them all, which is the safe outcome.
// ok=false means the user cancelled the whole operation.
func pickDirtyToForce(dirty []rmTarget, total int) (_ map[string]bool, ok bool, err error) {
	end := debug.Op("pick.many", fmt.Sprintf("%d-dirty", len(dirty)))
	defer func() { end(err) }()

	title := fmt.Sprintf(
		"%d of %d worktree(s) have uncommitted changes.\n"+
			"Select any you want to remove anyway; unselected are skipped.\n"+
			"(space toggles, enter continues, esc cancels)",
		len(dirty), total,
	)
	values, ok, err := picker.SelectMany(title, dirtyRows(dirty))
	if err != nil || !ok {
		return nil, false, err
	}
	forced := make(map[string]bool, len(values))
	for _, v := range values {
		forced[v] = true
	}
	return forced, true, nil
}

// resolveDirty decides what to do about targets carrying uncommitted work.
// Assumes scanDirty has already run. --force keeps everything;
// non-interactive refuses and names them all; interactive offers the
// skip/force choice. Returns (nil, nil) when the user cancels.
func resolveDirty(targets []rmTarget, force bool) ([]rmTarget, error) {
	dirty := dirtyTargets(targets)
	if len(dirty) == 0 {
		return targets, nil
	}

	if force {
		return applyDirtyChoice(targets, forceAll(dirty), forceFlag), nil
	}
	if !interactive() {
		return nil, dirtyBailError(dirty)
	}

	forced, ok, err := pickDirtyToForce(dirty, len(targets))
	if err != nil {
		return nil, err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "cancelled")
		return nil, nil
	}
	kept := applyDirtyChoice(targets, forced, forceChosen)
	for _, t := range dirty {
		if !forced[t.Path] {
			fmt.Fprintf(os.Stderr, "skipping %s (%s)\n", t.label(), t.dirty.Summary())
		}
	}
	if len(kept) == 0 {
		fmt.Fprintln(os.Stderr, "nothing left to remove")
	}
	return kept, nil
}

// preflightDirty is scanDirty followed by resolveDirty — the whole
// uncommitted-work gate for a caller that has nothing to print in between.
func preflightDirty(ctx context.Context, targets []rmTarget, force bool) ([]rmTarget, error) {
	return resolveDirty(scanDirty(ctx, targets), force)
}

// warnDiscarding notes work that --force is about to destroy. Targets the
// user picked out of the dirty prompt stay silent: they have already read
// these counts and chosen. --force never suppresses the scan, only the
// refusal, so there is always something to report.
func warnDiscarding(w io.Writer, t rmTarget) {
	if t.force != forceFlag || !t.dirty.Any() {
		return
	}
	fmt.Fprintf(w, "warning: %s has uncommitted changes (%s); removing anyway\n", t.label(), t.dirty.Summary())
}

// skipDirty drops every target with uncommitted work, warning about each.
// `clean` uses this instead of the interactive gate: it is a bulk sweep over
// worktrees the user never named, so the answer to "this one still has
// uncommitted work" is always to leave it alone and say so, rather than
// offer to destroy it.
func skipDirty(w io.Writer, targets []rmTarget) []rmTarget {
	kept := make([]rmTarget, 0, len(targets))
	for _, t := range targets {
		if t.dirty.Any() {
			fmt.Fprintf(w, "skipping %s: uncommitted changes (%s)\n", t.label(), t.dirty.Summary())
			continue
		}
		kept = append(kept, t)
	}
	return kept
}

// dirtyByPath indexes scanned targets for callers that render their own
// listing (clean prints reasons alongside the counts).
func dirtyByPath(targets []rmTarget) map[string]wt.DirtyStat {
	out := make(map[string]wt.DirtyStat, len(targets))
	for _, t := range targets {
		if t.dirty.Any() {
			out[t.Path] = t.dirty
		}
	}
	return out
}
