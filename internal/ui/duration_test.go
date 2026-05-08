package ui

import (
	"testing"
	"time"
)

func TestHumanDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, "now"},
		{-1 * time.Second, "now"},
		{30 * time.Second, "30s"},
		{59 * time.Second, "59s"},

		// Sub-minute is single unit (nothing finer than seconds to show in
		// the second slot). Everything else always shows TWO units, even
		// when the second slot is zero — consistency.
		{60 * time.Second, "1m 0s"},
		{61 * time.Second, "1m 1s"},
		{90 * time.Second, "1m 30s"},
		{59*time.Minute + 30*time.Second, "59m 30s"},
		{60 * time.Minute, "1h 0m"},
		{60*time.Minute + 1*time.Second, "1h 0m"}, // 1s doesn't fit a minute
		{60*time.Minute + 1*time.Minute, "1h 1m"},
		{5*time.Hour + 27*time.Minute, "5h 27m"},
		{23 * time.Hour, "23h 0m"},
		{24 * time.Hour, "1d 0h"},
		{25 * time.Hour, "1d 1h"},
		{6*24*time.Hour + 23*time.Hour, "6d 23h"},
		{7 * 24 * time.Hour, "1w 0d"},
		{4*7*24*time.Hour + 24*time.Hour, "4w 1d"},
		{29 * 24 * time.Hour, "4w 1d"},
		{30 * 24 * time.Hour, "1mo 0w"},
		{38 * 24 * time.Hour, "1mo 1w"},
		{364 * 24 * time.Hour, "12mo 0w"}, // 4d remainder; next unit is w, fits 0
		{365 * 24 * time.Hour, "1y 0mo"},
		{2*365*24*time.Hour + 31*24*time.Hour, "2y 1mo"},
	}
	for _, c := range cases {
		got := HumanDuration(c.in)
		if got != c.want {
			t.Errorf("HumanDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHumanSince_ZeroIsDash(t *testing.T) {
	got := HumanSince(time.Time{})
	if got != "—" {
		t.Errorf("HumanSince(zero) = %q, want —", got)
	}
}

func TestHumanSince_PositiveDelta(t *testing.T) {
	past := time.Now().Add(-2 * time.Hour)
	got := HumanSince(past)
	if got != "2h 0m" {
		t.Errorf("HumanSince(now-2h) = %q, want 2h 0m", got)
	}
}
