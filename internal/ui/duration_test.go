package ui

import (
	"testing"
	"time"
)

// All HumanDuration outputs are exactly 9 chars wide:
//
//	`%2d%-2s %2d%-2s`  →  2 + 2 + 1 + 2 + 2 = 9
//
// 1-char unit suffixes are padded to 2 chars by `%-2s`, so output may end
// with a trailing space when the last unit is single-char (m/s/h/d/y).
func TestHumanDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		// non-positive — renders as zero-zero
		{0, " 0m   0s "},
		{-1 * time.Second, " 0m   0s "},

		// sub-minute — zero minutes + seconds
		{30 * time.Second, " 0m  30s "},
		{59 * time.Second, " 0m  59s "},

		// minutes
		{60 * time.Second, " 1m   0s "},
		{61 * time.Second, " 1m   1s "},
		{90 * time.Second, " 1m  30s "},
		{59*time.Minute + 30*time.Second, "59m  30s "},

		// hours
		{60 * time.Minute, " 1h   0m "},
		{60*time.Minute + 1*time.Second, " 1h   0m "},
		{60*time.Minute + 1*time.Minute, " 1h   1m "},
		{5*time.Hour + 27*time.Minute, " 5h  27m "},
		{12*time.Hour + 5*time.Minute, "12h   5m "},
		{23 * time.Hour, "23h   0m "},

		// days
		{24 * time.Hour, " 1d   0h "},
		{24*time.Hour + 12*time.Hour, " 1d  12h "},
		{6*24*time.Hour + 23*time.Hour, " 6d  23h "},

		// weeks (suffix `wk`)
		{7 * 24 * time.Hour, " 1wk  0d "},
		{7*24*time.Hour + 6*24*time.Hour, " 1wk  6d "},
		{4*7*24*time.Hour + 24*time.Hour, " 4wk  1d "},
		{29 * 24 * time.Hour, " 4wk  1d "},

		// months/years
		{30 * 24 * time.Hour, " 1mo  0wk"},
		{38 * 24 * time.Hour, " 1mo  1wk"},
		{3*30*24*time.Hour + 14*24*time.Hour, " 3mo  2wk"},
		{364 * 24 * time.Hour, "12mo  0wk"},
		{365 * 24 * time.Hour, " 1y   0mo"},
		{2*365*24*time.Hour + 31*24*time.Hour, " 2y   1mo"},
	}
	for _, c := range cases {
		got := HumanDuration(c.in)
		if got != c.want {
			t.Errorf("HumanDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHumanDuration_FixedWidth(t *testing.T) {
	// All HumanDuration outputs must be exactly the same display width so
	// the picker's mtime column lines up across rows.
	const want = 9
	cases := []time.Duration{
		0, 30 * time.Second, 5 * time.Minute, time.Hour, 24 * time.Hour,
		7 * 24 * time.Hour, 30 * 24 * time.Hour, 365 * 24 * time.Hour,
		3 * 365 * 24 * time.Hour,
	}
	for _, d := range cases {
		s := HumanDuration(d)
		if len(s) != want {
			t.Errorf("HumanDuration(%v) length = %d (%q), want %d", d, len(s), s, want)
		}
	}
}

func TestHumanSince_ZeroIsPaddedDash(t *testing.T) {
	got := HumanSince(time.Time{})
	if len([]rune(got)) != 9 {
		t.Errorf("HumanSince(zero) rune-length = %d, want 9 (got %q)", len([]rune(got)), got)
	}
}

func TestHumanSince_PositiveDelta(t *testing.T) {
	past := time.Now().Add(-2 * time.Hour)
	got := HumanSince(past)
	if got != " 2h   0m " {
		t.Errorf("HumanSince(now-2h) = %q, want %q", got, " 2h   0m ")
	}
}
