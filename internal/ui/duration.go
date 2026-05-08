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

// units is ordered largest-first. We use 30d for "month" and 365d for "year"
// — close enough for "how stale is this worktree" labelling, and it keeps
// boundaries predictable. Seconds are included so the second-unit slot can
// show "1m 30s".
var durationUnits = []durationUnit{
	{365 * 24 * time.Hour, "y"},
	{30 * 24 * time.Hour, "mo"},
	{7 * 24 * time.Hour, "w"},
	{24 * time.Hour, "d"},
	{time.Hour, "h"},
	{time.Minute, "m"},
	{time.Second, "s"},
}

// HumanDuration formats a duration as a short label suitable for picker rows.
//
// Sub-minute durations render as a single seconds count ("30s") because
// there's nothing finer than seconds to show in the second slot. Anything
// longer always renders TWO units — the largest fitting unit and the unit
// immediately below it, even when the second slot is zero:
//   "5h 27m", "5h 0m", "1d 0h", "1y 0mo", "1mo 0w".
//
// Negative or zero durations render as "now".
func HumanDuration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	for i, u := range durationUnits {
		if d < u.amount {
			continue
		}
		top := int(d / u.amount)
		rem := d - time.Duration(top)*u.amount
		// Always emit the next-smaller unit if one exists, even at zero,
		// for consistent two-unit output.
		if i+1 < len(durationUnits) {
			next := durationUnits[i+1]
			second := int(rem / next.amount)
			return fmt.Sprintf("%d%s %d%s", top, u.suffix, second, next.suffix)
		}
		return fmt.Sprintf("%d%s", top, u.suffix)
	}
	return "now"
}

// HumanSince formats time.Since(t) via HumanDuration. Zero t yields "—".
func HumanSince(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return HumanDuration(time.Since(t))
}
