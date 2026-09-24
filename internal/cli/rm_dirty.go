package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shhac/git-wt/internal/debug"
	"github.com/shhac/git-wt/internal/picker"
	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/wt"
)

// scanDirty annotates every target with its uncommitted-file counts, before
// anything is deleted. A target whose status can't be read is left marked
// clean; the re-check inside removeWorktree is the backstop, and it defers
// to git when it can't establish safety either.
//
// Orphans are scanned too. A leftover from a crashed removal has a .git file
// pointing at records git has already pruned, so status fails there and it
// reads as clean — but the trees dir can also hold a directory that is a
// perfectly good repository of its own, and that one has work worth
// reporting before it is deleted.
func scanDirty(ctx context.Context, targets []rmTarget) []rmTarget {
	end := debug.Op("rm.scan-dirty", fmt.Sprintf("%d-target(s)", len(targets)))
	defer func() { end(nil) }()

	out := make([]rmTarget, len(targets))
	copy(out, targets)
	for i := range out {
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
// appears in forced. Dirty targets absent from forced are dropped. Order is
// preserved.
func applyDirtyChoice(targets []rmTarget, forced map[string]bool) []rmTarget {
	out := make([]rmTarget, 0, len(targets))
	for _, t := range targets {
		if !t.dirty.Any() {
			out = append(out, t)
			continue
		}
		if forced[t.Path] {
			t.force = true
			out = append(out, t)
		}
	}
	return out
}

// forceEveryTarget clears the whole list for removal and reports the work
// --force is about to destroy. Marking is unconditional: the flag means
// "don't stop for anything", so a target whose status scan failed must still
// be removed rather than refused by the re-check inside removeWorktree.
//
// This is where flag-forced work gets announced, so the user sees the full
// list before the first deletion. Targets picked out of the interactive
// prompt are marked elsewhere and stay silent — they have already been shown
// these counts and chosen.
func forceEveryTarget(w io.Writer, targets []rmTarget) []rmTarget {
	out := make([]rmTarget, len(targets))
	copy(out, targets)
	for i := range out {
		out[i].force = true
		if out[i].dirty.Any() {
			_, _ = fmt.Fprintf(w, "warning: %s has uncommitted changes (%s); removing anyway\n",
				out[i].label(), out[i].dirty.Summary())
		}
	}
	return out
}

// dirtyBailError names every dirty target in one error, so a
// non-interactive caller learns the full picture from a single run instead
// of discovering them one re-run at a time.
func dirtyBailError(dirty []rmTarget) error {
	var b strings.Builder
	fmt.Fprintf(&b, "uncommitted changes in %d worktree(s):", len(dirty))
	for _, t := range dirty {
		fmt.Fprintf(&b, "\n    %s (%s)", t.label(), t.dirty.Summary())
	}
	b.WriteString("\nuse --force to remove them anyway")
	return fmt.Errorf("%s", b.String())
}

// targetRows renders multi-select rows over targets, padding the labels so
// each target's note lines up. Uses the package's own column helpers rather
// than a %-*s of its own, so the alignment measures visible width and the
// rows carry the same dim styling as every other picker.
func targetRows(targets []rmTarget, note func(rmTarget) string) []picker.Row {
	labels := make([]string, len(targets))
	for i, t := range targets {
		labels[i] = t.label()
	}
	width := maxWidth(labels)

	rows := make([]picker.Row, len(targets))
	for i, t := range targets {
		rows[i] = picker.Row{
			Display: padRight(labels[i], width) + "  " + ui.Dim(note(t)),
			Value:   t.Path,
		}
	}
	return rows
}

func dirtyNote(t rmTarget) string { return t.dirty.Summary() }

// pickTargetSubset asks which of targets to go ahead with, returning the
// chosen paths. Toggling nothing — the default — chooses none, which is the
// safe outcome for every question it is used for. ok=false means the user
// cancelled the whole operation.
func pickTargetSubset(title string, targets []rmTarget, note func(rmTarget) string) (_ map[string]bool, ok bool, err error) {
	end := debug.Op("pick.many", fmt.Sprintf("%d-subset", len(targets)))
	defer func() { end(err) }()

	values, ok, err := picker.SelectMany(title, targetRows(targets, note))
	if err != nil || !ok {
		return nil, false, err
	}
	chosen := make(map[string]bool, len(values))
	for _, v := range values {
		chosen[v] = true
	}
	return chosen, true, nil
}

// resolveDirty decides what to do about targets carrying uncommitted work.
// Assumes scanDirty has already run. --force keeps everything;
// non-interactive refuses and names them all; interactive offers the
// skip/force choice. Returns (nil, nil) when the user cancels.
func resolveDirty(targets []rmTarget, force bool) ([]rmTarget, error) {
	// Checked before the dirty list is consulted: a target whose scan failed
	// reads as clean, and deciding from that list would leave it unforced for
	// removeWorktree's re-check to refuse — on a run that passed --force.
	if force {
		return forceEveryTarget(os.Stderr, targets), nil
	}

	dirty := dirtyTargets(targets)
	if len(dirty) == 0 {
		return targets, nil
	}
	if !interactive() {
		return nil, dirtyBailError(dirty)
	}

	title := fmt.Sprintf(
		"%d of %d selected worktrees have uncommitted changes.\n"+
			"Select any you want to remove anyway; unselected are skipped.\n"+
			"(space toggles, enter continues, esc cancels)",
		len(dirty), len(targets),
	)
	forced, ok, err := pickTargetSubset(title, dirty, dirtyNote)
	if err != nil {
		return nil, err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "cancelled")
		return nil, nil
	}
	kept := applyDirtyChoice(targets, forced)
	for _, t := range dirty {
		if !forced[t.Path] {
			reportSkipped(os.Stderr, t)
		}
	}
	if len(kept) == 0 {
		fmt.Fprintln(os.Stderr, "nothing left to remove")
	}
	return kept, nil
}

// reportSkipped names a worktree left alone for holding uncommitted work.
// Shared so `rm` and `clean` report the same event in the same words.
func reportSkipped(w io.Writer, t rmTarget) {
	_, _ = fmt.Fprintf(w, "skipping %s: uncommitted changes (%s)\n", t.label(), t.dirty.Summary())
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
			reportSkipped(w, t)
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
