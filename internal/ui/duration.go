package ui

import (
	"fmt"
	"time"
)

// HumanDuration formats a duration as a short human-readable label suitable
// for picker rows: "now", "14m", "3h", "2d", "5w", "8mo", "2y".
// Negative or zero durations render as "now".
func HumanDuration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw", int(d.Hours()/24/7))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/24/365))
	}
}

// HumanSince formats time.Since(t) via HumanDuration. Zero t yields "—".
func HumanSince(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return HumanDuration(time.Since(t))
}
