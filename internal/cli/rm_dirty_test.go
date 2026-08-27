package cli

import (
	"strings"
	"testing"

	"github.com/shhac/git-wt/internal/wt"
)

func dirtyFixture() []rmTarget {
	return []rmTarget{
		{Worktree: wt.Worktree{Path: "/p/a", Branch: "a"}},
		{Worktree: wt.Worktree{Path: "/p/b", Branch: "b"}, dirty: wt.DirtyStat{Modified: 4, Untracked: 1}},
		{Worktree: wt.Worktree{Path: "/p/c", Branch: "c"}},
		{Worktree: wt.Worktree{Path: "/p/d", Branch: "d"}, dirty: wt.DirtyStat{Untracked: 2}},
	}
}

func paths(ts []rmTarget) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Path
	}
	return out
}

func TestDirtyTargets(t *testing.T) {
	got := paths(dirtyTargets(dirtyFixture()))
	want := []string{"/p/b", "/p/d"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDirtyTargets_NoneDirty(t *testing.T) {
	clean := []rmTarget{{Worktree: wt.Worktree{Path: "/p/a"}}}
	if got := dirtyTargets(clean); len(got) != 0 {
		t.Errorf("got %v, want empty", paths(got))
	}
}

func TestApplyDirtyChoice_SkipsUnforced(t *testing.T) {
	kept := applyDirtyChoice(dirtyFixture(), map[string]bool{"/p/b": true}, forceChosen)
	if got, want := strings.Join(paths(kept), ","), "/p/a,/p/b,/p/c"; got != want {
		t.Fatalf("kept %q, want %q (order preserved, /p/d skipped)", got, want)
	}
	for _, k := range kept {
		switch k.Path {
		case "/p/b":
			if k.force != forceChosen {
				t.Errorf("/p/b force = %v, want forceChosen", k.force)
			}
		default:
			if k.force != forceNone {
				t.Errorf("%s force = %v, want forceNone", k.Path, k.force)
			}
		}
	}
}

func TestApplyDirtyChoice_NothingForced(t *testing.T) {
	kept := applyDirtyChoice(dirtyFixture(), nil, forceChosen)
	if got, want := strings.Join(paths(kept), ","), "/p/a,/p/c"; got != want {
		t.Errorf("kept %q, want %q", got, want)
	}
}

func TestApplyDirtyChoice_ForceAll(t *testing.T) {
	in := dirtyFixture()
	kept := applyDirtyChoice(in, forceAll(dirtyTargets(in)), forceFlag)
	if len(kept) != 4 {
		t.Fatalf("kept %v, want all four", paths(kept))
	}
	for _, k := range kept {
		want := forceNone
		if k.dirty.Any() {
			want = forceFlag
		}
		if k.force != want {
			t.Errorf("%s force = %v, want %v", k.Path, k.force, want)
		}
	}
}

func TestApplyDirtyChoice_DoesNotMutateInput(t *testing.T) {
	in := dirtyFixture()
	applyDirtyChoice(in, forceAll(dirtyTargets(in)), forceFlag)
	for _, tgt := range in {
		if tgt.force != forceNone {
			t.Errorf("%s mutated to force = %v", tgt.Path, tgt.force)
		}
	}
}

func TestResolveDirty_NoDirtyIsPassthrough(t *testing.T) {
	in := []rmTarget{{Worktree: wt.Worktree{Path: "/p/a"}}, {Worktree: wt.Worktree{Path: "/p/c"}}}
	got, err := resolveDirty(in, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %v, want both targets", paths(got))
	}
}

func TestResolveDirty_ForceKeepsEverything(t *testing.T) {
	got, err := resolveDirty(dirtyFixture(), true)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(got) != 4 {
		t.Errorf("got %v, want all four", paths(got))
	}
}

// Tests run with stdin detached, so interactive() is false and resolveDirty
// takes the non-interactive branch.
func TestResolveDirty_NonInteractiveRefusesAndNamesAll(t *testing.T) {
	_, err := resolveDirty(dirtyFixture(), false)
	if err == nil {
		t.Fatal("want an error naming the dirty worktrees")
	}
	msg := err.Error()
	for _, want := range []string{"b", "d", "4 modified, 1 untracked", "2 untracked", "--force"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error missing %q\n--- got ---\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "/p/a") || strings.Contains(msg, "/p/c") {
		t.Errorf("clean worktrees should not appear\n--- got ---\n%s", msg)
	}
}

func TestDirtyRows_AlignsCounts(t *testing.T) {
	rows := dirtyRows([]rmTarget{
		{Worktree: wt.Worktree{Path: "/p/x", Branch: "short"}, dirty: wt.DirtyStat{Modified: 1}},
		{Worktree: wt.Worktree{Path: "/p/y", Branch: "a-much-longer-branch"}, dirty: wt.DirtyStat{Untracked: 3}},
	})
	if len(rows) != 2 {
		t.Fatalf("len = %d, want 2", len(rows))
	}
	if rows[0].Value != "/p/x" || rows[1].Value != "/p/y" {
		t.Errorf("values should be paths, got %q / %q", rows[0].Value, rows[1].Value)
	}
	if strings.Index(rows[0].Display, "1 modified") != strings.Index(rows[1].Display, "3 untracked") {
		t.Errorf("counts not aligned:\n%q\n%q", rows[0].Display, rows[1].Display)
	}
}

func TestSkipDirty(t *testing.T) {
	var buf strings.Builder
	kept := skipDirty(&buf, dirtyFixture())
	if got, want := strings.Join(paths(kept), ","), "/p/a,/p/c"; got != want {
		t.Errorf("kept %q, want %q", got, want)
	}
	out := buf.String()
	for _, want := range []string{"skipping", "4 modified, 1 untracked", "2 untracked"} {
		if !strings.Contains(out, want) {
			t.Errorf("warning missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestDirtyByPath(t *testing.T) {
	got := dirtyByPath(dirtyFixture())
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got["/p/b"].Modified != 4 {
		t.Errorf("/p/b = %+v", got["/p/b"])
	}
	if _, ok := got["/p/a"]; ok {
		t.Error("clean worktree should be absent")
	}
}

func TestWarnDiscarding(t *testing.T) {
	tests := []struct {
		name string
		in   rmTarget
		want string
	}{
		{
			"flag-forced dirty target warns",
			rmTarget{Worktree: wt.Worktree{Path: "/p/b", Branch: "b"}, dirty: wt.DirtyStat{Modified: 4}, force: forceFlag},
			"4 modified",
		},
		{
			"prompt-chosen target stays silent",
			rmTarget{Worktree: wt.Worktree{Path: "/p/b", Branch: "b"}, dirty: wt.DirtyStat{Modified: 4}, force: forceChosen},
			"",
		},
		{
			"clean target stays silent",
			rmTarget{Worktree: wt.Worktree{Path: "/p/a", Branch: "a"}, force: forceFlag},
			"",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			warnDiscarding(&buf, tc.in)
			got := buf.String()
			if tc.want == "" {
				if got != "" {
					t.Errorf("want silence, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("got %q, want it to contain %q", got, tc.want)
			}
		})
	}
}
