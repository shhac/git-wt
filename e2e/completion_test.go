package e2e

import (
	"strings"
	"testing"
)

// TestCompletion_BashScriptRenders verifies `git-wt completion bash`
// emits something that looks like the standard Cobra v2 bash output.
func TestCompletion_BashScriptRenders(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "completion", "bash")
	if res.ExitCode != 0 {
		t.Fatalf("completion bash exit %d: %s", res.ExitCode, res.Stderr)
	}
	// Cobra v2 bash output starts with this header.
	if !strings.Contains(res.Stdout, "bash completion V2 for git-wt") {
		t.Errorf("expected V2 bash header; got:\n%s", res.Stdout[:min(200, len(res.Stdout))])
	}
	if !strings.Contains(res.Stdout, "__start_git-wt") {
		t.Errorf("expected __start_git-wt function; got:\n%s", res.Stdout[:min(400, len(res.Stdout))])
	}
}

// TestCompletion_ZshScriptRenders smoke-tests the zsh path.
func TestCompletion_ZshScriptRenders(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "completion", "zsh")
	if res.ExitCode != 0 {
		t.Fatalf("completion zsh exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "#compdef git-wt") {
		t.Errorf("expected zsh compdef header; got:\n%s", res.Stdout[:min(200, len(res.Stdout))])
	}
}

// TestCompletion_UnsupportedShellRejected — Cobra's OnlyValidArgs
// guard should kick in before we reach the switch.
func TestCompletion_UnsupportedShellRejected(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "completion", "tcsh")
	if res.ExitCode == 0 {
		t.Fatal("expected non-zero exit for unsupported shell")
	}
}

// TestCompletion_GoOffersWorktreeBranches uses Cobra's hidden
// `__complete` subcommand to exercise the live completion path,
// then asserts that an existing worktree's branch is offered.
func TestCompletion_GoOffersWorktreeBranches(t *testing.T) {
	repo := newRepo(t)
	// Create a worktree on a non-main branch.
	res := runWT(t, repo, "new", "feat-comp", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d: %s", res.ExitCode, res.Stderr)
	}
	// Run from the main worktree — `go` should offer the OTHER worktree.
	res = runWT(t, repo, "__complete", "go", "")
	if res.ExitCode != 0 {
		t.Fatalf("__complete go exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "feat-comp") {
		t.Errorf("expected feat-comp in completions; got:\n%s", res.Stdout)
	}
	// The current worktree's branch (main) MUST NOT appear — `go` filters
	// it because you can't navigate to where you already are. Lines look
	// like `branch\tdescription` now, so check the branch prefix.
	for _, line := range strings.Split(res.Stdout, "\n") {
		branch, _, _ := strings.Cut(strings.TrimSpace(line), "\t")
		if branch == "main" {
			t.Errorf("current worktree branch `main` should be excluded; got:\n%s", res.Stdout)
		}
	}
}

// TestCompletion_GoIncludesDescription confirms the new metadata
// (location + recency) ride along with each candidate so bash-v2 /
// zsh / fish render them next to the branch name.
func TestCompletion_GoIncludesDescription(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "new", "feat-desc", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d: %s", res.ExitCode, res.Stderr)
	}
	res = runWT(t, repo, "__complete", "go", "")
	if res.ExitCode != 0 {
		t.Fatalf("__complete go exit %d: %s", res.ExitCode, res.Stderr)
	}
	// A candidate line shaped like `feat-desc\t#feat-desc  <recency>`.
	// We only check that feat-desc carries a tab + something starting
	// with `#feat-desc`, which is what DisplayPath returns for trees-dir
	// children.
	var matched bool
	for _, line := range strings.Split(res.Stdout, "\n") {
		branch, desc, ok := strings.Cut(line, "\t")
		if !ok || branch != "feat-desc" {
			continue
		}
		if !strings.Contains(desc, "#feat-desc") {
			t.Errorf("expected feat-desc description to contain `#feat-desc`; got %q", desc)
		}
		matched = true
	}
	if !matched {
		t.Errorf("did not find a feat-desc completion line in:\n%s", res.Stdout)
	}
}

// TestCompletion_RmExcludesMainWorktree confirms `rm` never offers
// the main worktree as a removal target.
func TestCompletion_RmExcludesMainWorktree(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "new", "feat-rm", "--non-interactive", "--no-copy")
	if res.ExitCode != 0 {
		t.Fatalf("new exit %d: %s", res.ExitCode, res.Stderr)
	}
	res = runWT(t, repo, "__complete", "rm", "")
	if res.ExitCode != 0 {
		t.Fatalf("__complete rm exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "feat-rm") {
		t.Errorf("expected feat-rm in completions; got:\n%s", res.Stdout)
	}
	for _, line := range strings.Split(res.Stdout, "\n") {
		branch, _, _ := strings.Cut(strings.TrimSpace(line), "\t")
		if branch == "main" {
			t.Errorf("main worktree must never be offered for rm; got:\n%s", res.Stdout)
		}
	}
}

// TestAlias_IncludesCompletionBindingByDefault verifies the new
// alias-generator behaviour: completion glue is on by default.
func TestAlias_IncludesCompletionBindingByDefault(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "alias", "gwt")
	if res.ExitCode != 0 {
		t.Fatalf("alias gwt exit %d: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "compdef gwt=git-wt") {
		t.Errorf("expected zsh compdef line; got:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "complete -o default -F __start_git-wt gwt") {
		t.Errorf("expected bash complete line; got:\n%s", res.Stdout)
	}
}

// TestAlias_NoCompletionFlagSuppressesBinding verifies the opt-out.
func TestAlias_NoCompletionFlagSuppressesBinding(t *testing.T) {
	repo := newRepo(t)
	res := runWT(t, repo, "alias", "gwt", "--no-completion")
	if res.ExitCode != 0 {
		t.Fatalf("alias gwt --no-completion exit %d: %s", res.ExitCode, res.Stderr)
	}
	if strings.Contains(res.Stdout, "compdef") {
		t.Errorf("--no-completion should suppress compdef line; got:\n%s", res.Stdout)
	}
	if strings.Contains(res.Stdout, "complete -o default") {
		t.Errorf("--no-completion should suppress complete line; got:\n%s", res.Stdout)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
