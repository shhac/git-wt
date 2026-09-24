package cli

import (
	"strings"

	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/wt"
)

// How a lock is described, in its three settings: the padded tag column in
// list, picker and completion rows; the compact label inside clean's and
// rm's target listings; and the full sentence in lock, unlock, rm and clean
// messages.

// lockReasonWidth caps the reason shown in list and picker rows; the full
// text is in the messages lock and unlock print.
const lockReasonWidth = 40

// lockTag is the trailing column shown for a locked worktree: "locked" plus
// how long ago the lock was taken, so a lock outliving whatever took it
// stands out. "" for an unlocked worktree. The duration keeps HumanSince's
// fixed-width leading padding so tags line up across rows.
func lockTag(t wt.Worktree) string {
	if !t.Locked {
		return ""
	}
	if t.LockedAt.IsZero() {
		return "locked"
	}
	return "locked " + strings.TrimRight(ui.HumanSince(t.LockedAt), " ")
}

// withLockTag appends t's lock tag and a shortened reason to a rendered row.
func withLockTag(row string, t wt.Worktree, tagStyle, reasonStyle func(string) string) string {
	tag := lockTag(t)
	if tag == "" {
		return row
	}
	row += "  " + tagStyle(tag)
	if reason := truncate(oneLine(t.LockReason), lockReasonWidth); reason != "" {
		row += "  " + reasonStyle(reason)
	}
	return row
}

// lockLabel is lockTag without the column padding, for inline use:
// "locked" or "locked 3d 2h".
func lockLabel(t wt.Worktree) string {
	if age := lockAge(t); age != "" {
		return "locked " + age
	}
	return "locked"
}

// lockStateDetail describes t's lock state for messages: "not locked", or
// "locked" with how long ago and the full reason when known. Doubles as the
// "nothing to do" note, where the existing lock's age and reason tell the
// user whether it is the lock they meant to take.
func lockStateDetail(t wt.Worktree) string {
	if !t.Locked {
		return "not locked"
	}
	detail := "locked"
	if age := lockAge(t); age != "" {
		detail += " " + age + " ago"
	}
	if reason := oneLine(t.LockReason); reason != "" {
		detail += ": " + reason
	}
	return detail
}

// lockAge is how long t has been locked, unpadded ("3d 2h"); "" if unknown.
func lockAge(t wt.Worktree) string {
	if t.LockedAt.IsZero() {
		return ""
	}
	return strings.Join(strings.Fields(ui.HumanSince(t.LockedAt)), " ")
}
