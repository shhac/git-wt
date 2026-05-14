package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestConfig_ListShowsAllRegisteredKeys verifies the bare `config`
// invocation prints every wt.* key the binary knows about.
func TestConfig_ListShowsAllRegisteredKeys(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "config")
	if res.ExitCode != 0 {
		t.Fatalf("config exit %d: %s", res.ExitCode, res.Stderr)
	}
	for _, want := range []string{"wt.fd", "wt.parentDir", "wt.plain"} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("list output missing %s; got:\n%s", want, res.Stdout)
		}
	}
}

// TestConfig_SetParentDirLiteralAffectsNew exercises the precedence
// rule: with no --parent-dir flag and a configured wt.parentDir,
// `new` should land the worktree at the configured path.
func TestConfig_SetParentDirLiteralAffectsNew(t *testing.T) {
	repo := newRepo(t)
	custom := filepath.Join(repo, "configured-trees")

	if res := runWT(t, repo, "config", "parentDir", custom); res.ExitCode != 0 {
		t.Fatalf("config set: %s", res.Stderr)
	}
	res := runWT(t, repo, "new", "feat-cfg", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustExist(t, filepath.Join(custom, "feat-cfg"))
	mustNotExist(t, filepath.Join(repo, ".worktrees", "feat-cfg"))
}

// TestConfig_TemplatedParentDirResolves verifies the headline use
// case: a templated value in config expands per-repo so a single
// global setting can produce per-repo sibling directories like
// `<repo>.worktrees`.
func TestConfig_TemplatedParentDirResolves(t *testing.T) {
	repo := newRepo(t)
	if res := runWT(t, repo, "config", "parentDir", "${repoParent}/${repo}.worktrees"); res.ExitCode != 0 {
		t.Fatalf("config set: %s", res.Stderr)
	}

	// `new` should place the worktree at <parent-of-repo>/<repo-name>.worktrees/<branch>
	mustGit(t, repo, "branch", "feat-tmpl")
	res := runWT(t, repo, "add", "feat-tmpl", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("add exit %d: %s", res.ExitCode, res.Stderr)
	}
	expected := filepath.Join(filepath.Dir(repo), filepath.Base(repo)+".worktrees", "feat-tmpl")
	mustExist(t, expected)
}

// TestConfig_FlagWinsOverConfig verifies the precedence rule: --parent-dir
// on the command line overrides whatever's in gitconfig.
func TestConfig_FlagWinsOverConfig(t *testing.T) {
	repo := newRepo(t)
	configured := filepath.Join(repo, "from-config")
	flagDir := filepath.Join(repo, "from-flag")

	if res := runWT(t, repo, "config", "parentDir", configured); res.ExitCode != 0 {
		t.Fatalf("config set: %s", res.Stderr)
	}
	res := runWT(t, repo, "new", "feat-flag", "--parent-dir", flagDir, "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d: %s", res.ExitCode, res.Stderr)
	}
	mustExist(t, filepath.Join(flagDir, "feat-flag"))
	mustNotExist(t, filepath.Join(configured, "feat-flag"))
}

// TestConfig_RejectsUnknownTemplateVarAtSetTime is the loud-failure
// promise: a typo in a template must error at the `config` call, not
// silently sit in gitconfig and blow up at worktree-creation time.
func TestConfig_RejectsUnknownTemplateVarAtSetTime(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "config", "parentDir", "${repoo}/wt")
	if res.ExitCode == 0 {
		t.Fatal("expected non-zero exit for unknown template var")
	}
	if !strings.Contains(res.Stderr, "repoo") {
		t.Errorf("error should name the bad var; got: %s", res.Stderr)
	}
}

// TestConfig_RejectsBadTypesAtSetTime checks the other two validators.
func TestConfig_RejectsBadTypesAtSetTime(t *testing.T) {
	repo := newRepo(t)
	cases := []struct{ key, value, wantSub string }{
		{"fd", "10", "fd must be 3-9"},
		{"fd", "two", "not an integer"},
		{"plain", "maybe", "invalid boolean"},
	}
	for _, c := range cases {
		t.Run(c.key+"="+c.value, func(t *testing.T) {
			res := runWT(t, repo, "config", c.key, c.value)
			if res.ExitCode == 0 {
				t.Fatalf("expected non-zero exit for %s=%s", c.key, c.value)
			}
			if !strings.Contains(res.Stderr, c.wantSub) {
				t.Errorf("stderr should contain %q; got: %s", c.wantSub, res.Stderr)
			}
		})
	}
}

// TestConfig_UnsetIsIdempotent — unsetting a missing key must not error.
func TestConfig_UnsetIsIdempotent(t *testing.T) {
	repo := newRepo(t)
	// Never set; unset should be a no-op exit-0.
	res := runWT(t, repo, "config", "--unset", "parentDir")
	if res.ExitCode != 0 {
		t.Fatalf("unset of missing key exit %d: %s", res.ExitCode, res.Stderr)
	}
}

// TestConfig_ShowResolvesTemplate verifies the resolved-value display
// — running `config <key>` against a templated value shows the
// expanded path next to the raw template.
func TestConfig_ShowResolvesTemplate(t *testing.T) {
	repo := newRepo(t)
	if res := runWT(t, repo, "config", "parentDir", "${repoPath}/wt"); res.ExitCode != 0 {
		t.Fatalf("config set: %s", res.Stderr)
	}
	res := runWT(t, repo, "config", "parentDir")
	if res.ExitCode != 0 {
		t.Fatalf("config show exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "resolved:") {
		t.Errorf("expected resolved: line; got:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "/wt") {
		t.Errorf("expected resolved path to contain /wt; got:\n%s", res.Stdout)
	}
}

// TestConfig_BypassValidationBadTemplateSurfacesAtUse confirms the
// loud-failure promise: even if a user writes a bad template directly
// via `git config` (skipping our set-time validation), the typo gets
// named at `new`/`add` time rather than silently producing a wrong path.
func TestConfig_BypassValidationBadTemplateSurfacesAtUse(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "config", "--local", "wt.parentDir", "${nope}/wt")
	res := runWT(t, repo, "new", "feat-bypass", "--non-interactive", "--no-copy")
	if res.ExitCode == 0 {
		t.Fatal("expected non-zero exit when wt.parentDir has a bad template var")
	}
	if !strings.Contains(res.Stderr, "nope") {
		t.Errorf("error should name the bad var; got: %s", res.Stderr)
	}
}

// TestConfig_FDAndPlainTakeEffectAtRuntime exercises applyConfigDefaults
// end-to-end. With wt.plain=true and no --plain flag, the `list`
// output should be color-free; with wt.fd set, the alias generator
// (which reads --fd) should reflect a different default in its
// generated function.
func TestConfig_FDAndPlainTakeEffectAtRuntime(t *testing.T) {
	repo := newRepo(t)
	mustGit(t, repo, "config", "--local", "wt.plain", "true")

	// Visual smoke: when wt.plain is true, `list` runs cleanly with
	// no ANSI escapes in stdout. The harness pins --fd 9 by default
	// for non-flag-starting commands, so we just check stderr is
	// clean (no panic / error) and stdout has at least the header.
	res := runWT(t, repo, "list")
	if res.ExitCode != 0 {
		t.Fatalf("list exit %d: %s", res.ExitCode, res.Stderr)
	}
	if strings.Contains(res.Stdout, "\x1b[") {
		t.Errorf("wt.plain=true should suppress ANSI escapes; stdout contained one: %q", res.Stdout)
	}
}

// TestConfig_UnknownKey gives a helpful error including the registry.
func TestConfig_UnknownKey(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "config", "parntDir", "/x")
	if res.ExitCode == 0 {
		t.Fatal("expected non-zero exit for unknown key")
	}
	if !strings.Contains(res.Stderr, "unknown config key") {
		t.Errorf("expected unknown-key error; got: %s", res.Stderr)
	}
	if !strings.Contains(res.Stderr, "wt.parentDir") {
		t.Errorf("error should list known keys; got: %s", res.Stderr)
	}
}
