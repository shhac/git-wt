package wt

import "testing"

func TestParseDirtyStat(t *testing.T) {
	tests := []struct {
		name      string
		porcelain string
		unborn    bool
		want      DirtyStat
	}{
		{"empty", "", false, DirtyStat{}},
		{"whitespace only", "\n  \n", false, DirtyStat{}},
		{"untracked only", "?? a.txt\n?? b/\n", false, DirtyStat{Untracked: 2}},
		{"unstaged modification", " M main.go\n", false, DirtyStat{Modified: 1}},
		{"staged addition", "A  new.go\n", false, DirtyStat{Modified: 1}},
		{"staged and unstaged", "MM both.go\n", false, DirtyStat{Modified: 1}},
		{"deletion", " D gone.go\n", false, DirtyStat{Modified: 1}},
		{"rename", "R  old.go -> new.go\n", false, DirtyStat{Modified: 1}},
		{"mixed", " M main.go\nA  new.go\n?? scratch\n", false, DirtyStat{Modified: 2, Untracked: 1}},

		// A worktree whose branch ref was deleted has no HEAD to diff
		// against, so git calls every tracked file a staged addition.
		{"unborn HEAD, genuinely clean", "A  a.txt\nA  b.txt\n", true, DirtyStat{}},
		{"unborn HEAD, real edit", "AM a.txt\nA  b.txt\n", true, DirtyStat{Modified: 1}},
		{"unborn HEAD, untracked still counts", "A  a.txt\n?? scratch\n", true, DirtyStat{Untracked: 1}},
		{"unborn HEAD, deleted from worktree", "AD a.txt\n", true, DirtyStat{Modified: 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseDirtyStat(tc.porcelain, tc.unborn); got != tc.want {
				t.Errorf("parseDirtyStat(%q, unborn=%v) = %+v, want %+v", tc.porcelain, tc.unborn, got, tc.want)
			}
		})
	}
}

func TestDirtyStatSummary(t *testing.T) {
	tests := []struct {
		in   DirtyStat
		want string
	}{
		{DirtyStat{}, ""},
		{DirtyStat{Modified: 4}, "4 modified"},
		{DirtyStat{Untracked: 1}, "1 untracked"},
		{DirtyStat{Modified: 4, Untracked: 1}, "4 modified, 1 untracked"},
	}
	for _, tc := range tests {
		if got := tc.in.Summary(); got != tc.want {
			t.Errorf("%+v.Summary() = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDirtyStatAny(t *testing.T) {
	if (DirtyStat{}).Any() {
		t.Error("zero DirtyStat should not be Any()")
	}
	if !(DirtyStat{Untracked: 1}).Any() {
		t.Error("untracked-only should be Any()")
	}
}
