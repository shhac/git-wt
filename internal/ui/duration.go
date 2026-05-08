package ui

import (
	"fmt"
	"time"
)

// durationUnit is a single (amount, suffix) row in the breakdown.
type durationUnit struct {
	amount time.Duration
	suffix string
}

// Units are ordered largest-first. 30d for "month" and 365d for "year" —
// close enough for "how stale is this worktree" labelling, and it keeps
// boundaries predictable.
//
// Suffixes are 1- or 2-char so the "%-2s" format below produces a fixed
// 4-char-wide unit cell (`%2d` + `%-2s`).
var durationUnits = []durationUnit{
	{365 * 24 * time.Hour, "y"},
	{30 * 24 * time.Hour, "mo"},
	{7 * 24 * time.Hour, "wk"},
	{24 * time.Hour, "d"},
	{time.Hour, "h"},
	{time.Minute, "m"},
	{time.Second, "s"},
}

// HumanDuration formats a duration as a fixed-width 9-char label suitable
// for column-aligned picker rows. Each unit cell is `%2d%-2s` (4 chars),
// joined with a single space — total width 9.
//
// Always two units, even when one is zero, for visual consistency:
//
//	"12h  5m"   "1h  0m"
//	" 1d 12h"   " 1d  0h"
//	" 1wk 6d"   " 1wk 0d"
//	" 3mo 2wk"  " 1mo 0wk"
//	" 1y  0mo"
//
// Sub-minute durations render as " 0m XXs" (zero minutes + seconds), keeping
// the same width. Non-positive input renders as "0m  0s" (also width 9).
func HumanDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%2d%-2s %2d%-2s", 0, "m", int(d.Seconds()), "s")
	}
	for i, u := range durationUnits {
		if d < u.amount {
			continue
		}
		top := int(d / u.amount)
		rem := d - time.Duration(top)*u.amount
		// `s` is the smallest unit; below `m` we drop into the sub-minute path
		// above, so we always have an i+1 here.
		next := durationUnits[i+1]
		second := int(rem / next.amount)
		return fmt.Sprintf("%2d%-2s %2d%-2s", top, u.suffix, second, next.suffix)
	}
	return fmt.Sprintf("%2d%-2s %2d%-2s", 0, "m", 0, "s")
}

// HumanSince formats time.Since(t) via HumanDuration. Zero t yields a
// 9-char-wide em-dash placeholder so column alignment stays consistent.
func HumanSince(t time.Time) string {
	if t.IsZero() {
		return "        —"
	}
	return HumanDuration(time.Since(t))
}
