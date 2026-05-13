package cli

import (
	"strings"
	"testing"
)

func TestValidIdentifier(t *testing.T) {
	good := []string{"gwt", "_g", "gwt2", "MY_ALIAS", "a"}
	for _, s := range good {
		if !validIdentifier(s) {
			t.Errorf("validIdentifier(%q) = false, want true", s)
		}
	}
	bad := []string{"", "1leading", "with-hyphen", "with space", "dot.alias", "ünicode"}
	for _, s := range bad {
		if validIdentifier(s) {
			t.Errorf("validIdentifier(%q) = true, want false", s)
		}
	}
}

func TestShellQuote(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/usr/bin/git-wt", `'/usr/bin/git-wt'`},
		{"", `''`},
		{"with spaces", `'with spaces'`},
		{"with'quote", `'with'\''quote'`},
		{"weird $`!", "'weird $`!'"},
	}
	for _, c := range cases {
		got := shellQuote(c.in)
		if got != c.want {
			t.Errorf("shellQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRenderAlias_Defaults(t *testing.T) {
	got := renderAlias("gwt", "/bin/git-wt", 3, false, false, false)

	mustContain(t, got, "gwt() {")
	mustContain(t, got, `local bin='/bin/git-wt'`)
	mustContain(t, got, `--fd 3`)
	mustContain(t, got, `3>&1 1>&2`)
	mustContain(t, got, "go|new|add|eject|rm)")
	// The wrapper walks args so leading global flags (--debug, --plain, etc.)
	// don't bypass the cd-capable branch.
	mustContain(t, got, `for _arg in "$@"`)
	mustContain(t, got, `case "$_sub" in`)
	mustNotContain(t, got, `case "$1" in`)
	mustNotContain(t, got, "echo \"[gwt]") // no debug
	mustNotContain(t, got, "--plain")
	mustNotContain(t, got, "--non-interactive")
}

func TestRenderAlias_BakesPlainAndNonInteractive(t *testing.T) {
	got := renderAlias("gwt", "/bin/git-wt", 3, true, true, false)
	// Both flags appear in the wrapper invocation
	if !strings.Contains(got, " --plain --non-interactive ") {
		t.Errorf("expected ` --plain --non-interactive ` baked into invocation, got:\n%s", got)
	}
}

func TestRenderAlias_CustomFD(t *testing.T) {
	got := renderAlias("gwt", "/bin/git-wt", 7, false, false, false)
	mustContain(t, got, `--fd 7`)
	mustContain(t, got, `7>&1 1>&2`)
	mustNotContain(t, got, `--fd 3`)
}

func TestRenderAlias_DebugAddsThreeEchoLines(t *testing.T) {
	got := renderAlias("gwt", "/bin/git-wt", 3, false, false, true)
	count := strings.Count(got, `echo "[gwt]`)
	if count != 3 {
		t.Errorf("expected 3 debug echo lines, got %d\n--- output ---\n%s", count, got)
	}
}

func TestRenderAlias_BinPathWithSingleQuote(t *testing.T) {
	got := renderAlias("gwt", `/Users/paul's/bin/git-wt`, 3, false, false, false)
	// The path should be quoted with the standard single-quote escape.
	mustContain(t, got, `'/Users/paul'\''s/bin/git-wt'`)
}

func mustContain(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("expected output to contain %q\n--- output ---\n%s", sub, s)
	}
}

func mustNotContain(t *testing.T, s, sub string) {
	t.Helper()
	if strings.Contains(s, sub) {
		t.Errorf("expected output NOT to contain %q\n--- output ---\n%s", sub, s)
	}
}
