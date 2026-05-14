package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/shhac/git-wt/internal/config"
)

// TestResolvedLineFor_BadTemplateSurfacesError exercises the load-bearing
// promise of resolvedLineFor: even when a bad template made it into
// storage (e.g. via raw `git config` bypassing Validate), the show
// command must mention the error in the resolved: line rather than
// silently dropping the value.
func TestResolvedLineFor_BadTemplateSurfacesError(t *testing.T) {
	setupTempGitconfig(t) // chdirs into a real repo so wt.Inspect works
	e := config.Entry{
		Key:    config.ParentDir,
		Value:  "${nope}/wt",
		Source: config.ScopeLocal,
		IsSet:  true,
	}
	got := resolvedLineFor(context.Background(), config.ParentDir, e)
	if !strings.Contains(got, "error:") {
		t.Errorf("expected `error:` in line for bad template; got %q", got)
	}
	if !strings.Contains(got, "nope") {
		t.Errorf("expected bad var name `nope` in line; got %q", got)
	}
}

// TestResolvedLineFor_HappyPath: a valid template resolves to a real path.
func TestResolvedLineFor_HappyPath(t *testing.T) {
	setupTempGitconfig(t)
	e := config.Entry{
		Key:    config.ParentDir,
		Value:  "${repoPath}/wt",
		Source: config.ScopeLocal,
		IsSet:  true,
	}
	got := resolvedLineFor(context.Background(), config.ParentDir, e)
	if !strings.HasPrefix(got, "resolved: ") {
		t.Errorf("expected `resolved:` prefix; got %q", got)
	}
	if !strings.HasSuffix(got, "/wt") {
		t.Errorf("expected resolved path to end with /wt; got %q", got)
	}
}

// TestResolvedLineFor_NotTemplatedReturnsEmpty: a non-templated key
// shouldn't produce a resolved line at all.
func TestResolvedLineFor_NotTemplatedReturnsEmpty(t *testing.T) {
	setupTempGitconfig(t)
	e := config.Entry{Key: config.Plain, Value: "true", Source: config.ScopeLocal, IsSet: true}
	if got := resolvedLineFor(context.Background(), config.Plain, e); got != "" {
		t.Errorf("non-templated key should produce empty line; got %q", got)
	}
}

// TestResolvedLineFor_UnsetReturnsEmpty: unset key, even if templated,
// has nothing to resolve.
func TestResolvedLineFor_UnsetReturnsEmpty(t *testing.T) {
	setupTempGitconfig(t)
	e := config.Entry{Key: config.ParentDir, IsSet: false}
	if got := resolvedLineFor(context.Background(), config.ParentDir, e); got != "" {
		t.Errorf("unset key should produce empty line; got %q", got)
	}
}
