package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shhac/git-wt/internal/config"
	"github.com/shhac/git-wt/internal/git"
)

func TestResolveParentDir_FlagEmptyConfigUnsetUsesDefault(t *testing.T) {
	repo := setupTempGitconfig(t)
	got, err := resolveParentDir(context.Background(), repo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(repo, ".worktrees")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveParentDir_FlagEmptyConfigLiteral(t *testing.T) {
	repo := setupTempGitconfig(t)
	custom := filepath.Join(repo, "configured")
	if err := config.Set(context.Background(), config.ParentDir, custom, config.ScopeLocal); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := resolveParentDir(context.Background(), repo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != custom {
		t.Errorf("got %q, want %q", got, custom)
	}
}

func TestResolveParentDir_FlagEmptyConfigTemplate(t *testing.T) {
	repo := setupTempGitconfig(t)
	if err := config.Set(context.Background(), config.ParentDir, "${repoParent}/${repo}.worktrees", config.ScopeLocal); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := resolveParentDir(context.Background(), repo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(filepath.Dir(repo), filepath.Base(repo)+".worktrees")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveParentDir_FlagWinsOverConfig(t *testing.T) {
	repo := setupTempGitconfig(t)
	if err := config.Set(context.Background(), config.ParentDir, "/from-config", config.ScopeLocal); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := resolveParentDir(context.Background(), repo, "/from-flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/from-flag" {
		t.Errorf("flag should win; got %q", got)
	}
}

func TestResolveParentDir_BadTemplateViaRawGitConfigSurfacesError(t *testing.T) {
	repo := setupTempGitconfig(t)
	// Bypass our validation by writing the bad template directly via
	// git, which is what would happen if a user manually edited
	// .git/config. The error must surface here at use time — silently
	// falling back would mean the worktree lands somewhere unexpected.
	if _, err := git.Run(context.Background(), "config", "--local", "wt.parentDir", "${nope}/wt"); err != nil {
		t.Fatalf("raw git config set: %v", err)
	}
	_, err := resolveParentDir(context.Background(), repo, "")
	if err == nil {
		t.Fatal("expected error from bad template, got nil")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error should name the bad var; got: %v", err)
	}
}
