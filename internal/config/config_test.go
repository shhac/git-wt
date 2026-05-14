package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shhac/git-wt/internal/testutil"
)

func TestParseBool(t *testing.T) {
	cases := []struct {
		in   string
		want bool
		err  bool
	}{
		{"true", true, false},
		{"yes", true, false},
		{"on", true, false},
		{"1", true, false},
		{"TRUE", true, false},
		{"false", false, false},
		{"no", false, false},
		{"off", false, false},
		{"0", false, false},
		{"", false, false},
		{"maybe", false, true},
		{"2", false, true},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := ParseBool(c.in)
			if (err != nil) != c.err {
				t.Fatalf("err=%v want err=%v", err, c.err)
			}
			if !c.err && got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestFind(t *testing.T) {
	cases := []struct {
		in   string
		want *Key
	}{
		{"parentDir", ParentDir},
		{"parentdir", ParentDir},
		{"wt.parentDir", ParentDir},
		{"WT.PARENTDIR", ParentDir},
		{"plain", Plain},
		{"fd", FD},
		{"nope", nil},
		{"", nil},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := Find(c.in)
			if got != c.want {
				t.Errorf("Find(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name  string
		key   *Key
		value string
		err   bool
	}{
		{"plain accepts true", Plain, "true", false},
		{"plain rejects garbage", Plain, "maybe", true},
		{"fd accepts 3", FD, "3", false},
		{"fd accepts 9", FD, "9", false},
		{"fd rejects 2", FD, "2", true},
		{"fd rejects 10", FD, "10", true},
		{"fd rejects non-int", FD, "three", true},
		{"parentDir accepts plain path", ParentDir, "/tmp/wt", false},
		{"parentDir accepts known template", ParentDir, "${repoParent}/${repo}.wt", false},
		{"parentDir rejects unknown template", ParentDir, "${repoo}/wt", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Validate(c.key, c.value)
			if (err != nil) != c.err {
				t.Fatalf("got err=%v want err=%v", err, c.err)
			}
		})
	}
}

// setupTempGitconfig is a thin alias for testutil.SetupTempGitconfig
// so existing call sites read naturally. The real helper lives in
// internal/testutil so multiple packages can share it.
func setupTempGitconfig(t *testing.T) string {
	t.Helper()
	return testutil.SetupTempGitconfig(t)
}

func TestSetGetRoundTrip_Local(t *testing.T) {
	setupTempGitconfig(t)
	ctx := context.Background()

	if err := Set(ctx, ParentDir, "/tmp/wt", ScopeLocal); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok, err := GetScoped(ctx, ParentDir, ScopeLocal)
	if err != nil || !ok {
		t.Fatalf("GetScoped: ok=%v err=%v", ok, err)
	}
	if got != "/tmp/wt" {
		t.Errorf("got %q, want %q", got, "/tmp/wt")
	}
}

func TestUnset_Idempotent(t *testing.T) {
	setupTempGitconfig(t)
	ctx := context.Background()

	// Unsetting something that was never set must not error.
	if err := Unset(ctx, Plain, ScopeLocal); err != nil {
		t.Fatalf("Unset of missing key should succeed, got: %v", err)
	}

	// Set then unset, twice — second unset still no error.
	if err := Set(ctx, Plain, "true", ScopeLocal); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := Unset(ctx, Plain, ScopeLocal); err != nil {
		t.Fatalf("first Unset: %v", err)
	}
	if err := Unset(ctx, Plain, ScopeLocal); err != nil {
		t.Fatalf("second Unset: %v", err)
	}
}

func TestGetEffective_PrefersLocalOverGlobal(t *testing.T) {
	setupTempGitconfig(t)
	ctx := context.Background()

	if err := Set(ctx, ParentDir, "/from-global", ScopeGlobal); err != nil {
		t.Fatalf("Set global: %v", err)
	}
	e, err := GetEffective(ctx, ParentDir)
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if !e.IsSet || e.Value != "/from-global" || e.Source != ScopeGlobal {
		t.Errorf("expected global value, got %+v", e)
	}

	if err := Set(ctx, ParentDir, "/from-local", ScopeLocal); err != nil {
		t.Fatalf("Set local: %v", err)
	}
	e, err = GetEffective(ctx, ParentDir)
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if !e.IsSet || e.Value != "/from-local" || e.Source != ScopeLocal {
		t.Errorf("expected local value to win, got %+v", e)
	}
}

func TestGetScoped_PropagatesNonExit1Errors(t *testing.T) {
	repo := testutil.SetupTempGitconfig(t)
	// Corrupt .git/config so any read returns a real error (not exit 1).
	// `git config --get` returns exit 3 ("malformed file") in this case.
	cfgPath := filepath.Join(repo, ".git", "config")
	if err := os.WriteFile(cfgPath, []byte("[wt\nbroken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := GetScoped(context.Background(), ParentDir, ScopeLocal)
	if err == nil {
		t.Fatal("expected GetScoped to propagate the malformed-config error, got nil")
	}
}

func TestGetEffective_UnsetReportsNotSet(t *testing.T) {
	setupTempGitconfig(t)
	ctx := context.Background()
	e, err := GetEffective(ctx, FD)
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if e.IsSet {
		t.Errorf("expected unset, got %+v", e)
	}
}

func TestList_IncludesAllRegisteredKeys(t *testing.T) {
	setupTempGitconfig(t)
	ctx := context.Background()
	entries, err := List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != len(All) {
		t.Fatalf("got %d entries, want %d", len(entries), len(All))
	}
	seen := map[string]bool{}
	for _, e := range entries {
		seen[e.Key.Name] = true
	}
	for _, k := range All {
		if !seen[k.Name] {
			t.Errorf("missing key %s in list", k.Name)
		}
	}
}

func TestScopeStrings(t *testing.T) {
	if !strings.Contains(ScopeLocal.String(), "local") {
		t.Error("ScopeLocal stringer")
	}
	if !strings.Contains(ScopeGlobal.String(), "global") {
		t.Error("ScopeGlobal stringer")
	}
}
