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
		{60 * time.Second, "1m"},
		{59*time.Minute + 59*time.Second, "59m"},
		{60 * time.Minute, "1h"},
		{23 * time.Hour, "23h"},
		{24 * time.Hour, "1d"},
		{6*24*time.Hour + 23*time.Hour, "6d"},
		{7 * 24 * time.Hour, "1w"},
		{29 * 24 * time.Hour, "4w"},
		{30 * 24 * time.Hour, "1mo"},
		{364 * 24 * time.Hour, "12mo"},
		{365 * 24 * time.Hour, "1y"},
		{2 * 365 * 24 * time.Hour, "2y"},
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
	if got != "2h" {
		t.Errorf("HumanSince(now-2h) = %q, want 2h", got)
	}
}
