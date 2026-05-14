package cli

import (
	"context"
	"testing"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/config"
	"github.com/shhac/git-wt/internal/git"
	"github.com/shhac/git-wt/internal/testutil"
)

// setupTempGitconfig is a thin alias for testutil.SetupTempGitconfig
// so existing call sites read naturally.
func setupTempGitconfig(t *testing.T) string {
	t.Helper()
	return testutil.SetupTempGitconfig(t)
}

// makeCmdWithFlags returns a cobra.Command shaped like the real
// rootCmd: bool flag "plain", int flag "fd". Tests bind these to
// local destinations so we can avoid mutating the package globals.
func makeCmdWithFlags(t *testing.T, dest *bool, fdDest *int) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().BoolVar(dest, "plain", false, "")
	cmd.Flags().IntVar(fdDest, "fd", 3, "")
	cmd.SetContext(context.Background())
	return cmd
}

func TestApplyBoolFlagDefault_UsesConfigWhenFlagUnchanged(t *testing.T) {
	setupTempGitconfig(t)
	if err := config.Set(context.Background(), config.Plain, "true", config.ScopeLocal); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var plain bool
	var fd int
	cmd := makeCmdWithFlags(t, &plain, &fd)

	applyBoolFlagDefault(cmd, "plain", config.Plain, &plain)

	if !plain {
		t.Errorf("expected plain=true from config, got false")
	}
}

func TestApplyBoolFlagDefault_FlagWinsWhenChanged(t *testing.T) {
	setupTempGitconfig(t)
	if err := config.Set(context.Background(), config.Plain, "true", config.ScopeLocal); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var plain bool
	var fd int
	cmd := makeCmdWithFlags(t, &plain, &fd)
	// Simulate the user explicitly passing --plain=false.
	if err := cmd.Flags().Set("plain", "false"); err != nil {
		t.Fatal(err)
	}

	applyBoolFlagDefault(cmd, "plain", config.Plain, &plain)

	if plain {
		t.Errorf("explicit flag should win over config; got plain=true")
	}
}

func TestApplyBoolFlagDefault_UnsetConfigLeavesFlagAlone(t *testing.T) {
	setupTempGitconfig(t)
	// No config set.
	plain := false
	fd := 3
	cmd := makeCmdWithFlags(t, &plain, &fd)

	applyBoolFlagDefault(cmd, "plain", config.Plain, &plain)

	if plain {
		t.Errorf("unset config should leave dest at CLI default (false), got true")
	}
}

func TestApplyBoolFlagDefault_GarbageConfigIgnoredSilently(t *testing.T) {
	setupTempGitconfig(t)
	// Bypass Validate by writing directly with raw git — this is how
	// a malformed gitconfig would actually arrive in practice.
	ctx := context.Background()
	if _, err := git.Run(ctx, "config", "--local", "wt.plain", "maybe"); err != nil {
		t.Fatalf("raw git config set: %v", err)
	}
	plain := false
	fd := 3
	cmd := makeCmdWithFlags(t, &plain, &fd)

	applyBoolFlagDefault(cmd, "plain", config.Plain, &plain)

	// Garbage value → silently skipped (load-bearing UX promise).
	if plain {
		t.Errorf("malformed bool should be silently ignored, got plain=true")
	}
}

func TestApplyIntFlagDefault_UsesConfigWhenFlagUnchanged(t *testing.T) {
	setupTempGitconfig(t)
	if err := config.Set(context.Background(), config.FD, "7", config.ScopeLocal); err != nil {
		t.Fatalf("Set: %v", err)
	}
	plain := false
	fd := 3
	cmd := makeCmdWithFlags(t, &plain, &fd)

	applyIntFlagDefault(cmd, "fd", config.FD, &fd)

	if fd != 7 {
		t.Errorf("expected fd=7 from config, got %d", fd)
	}
}

func TestApplyIntFlagDefault_OutOfRangeIgnored(t *testing.T) {
	setupTempGitconfig(t)
	// Validate would reject 99, so we have to bypass it via raw git
	// to reach the read-side guard.
	ctx := context.Background()
	if _, err := git.Run(ctx, "config", "--local", "wt.fd", "99"); err != nil {
		t.Fatalf("raw git config set: %v", err)
	}
	plain := false
	fd := 3
	cmd := makeCmdWithFlags(t, &plain, &fd)

	applyIntFlagDefault(cmd, "fd", config.FD, &fd)

	if fd != 3 {
		t.Errorf("out-of-range value should be ignored, kept at 3; got %d", fd)
	}
}

func TestApplyIntFlagDefault_NonIntIgnored(t *testing.T) {
	setupTempGitconfig(t)
	ctx := context.Background()
	if _, err := git.Run(ctx, "config", "--local", "wt.fd", "three"); err != nil {
		t.Fatalf("raw git config set: %v", err)
	}
	plain := false
	fd := 3
	cmd := makeCmdWithFlags(t, &plain, &fd)

	applyIntFlagDefault(cmd, "fd", config.FD, &fd)

	if fd != 3 {
		t.Errorf("non-int value should be ignored; got %d", fd)
	}
}

func TestApplyIntFlagDefault_FlagWinsWhenChanged(t *testing.T) {
	setupTempGitconfig(t)
	if err := config.Set(context.Background(), config.FD, "7", config.ScopeLocal); err != nil {
		t.Fatalf("Set: %v", err)
	}
	plain := false
	fd := 3
	cmd := makeCmdWithFlags(t, &plain, &fd)
	if err := cmd.Flags().Set("fd", "5"); err != nil {
		t.Fatal(err)
	}

	applyIntFlagDefault(cmd, "fd", config.FD, &fd)

	if fd != 5 {
		t.Errorf("explicit --fd 5 should beat config fd=7; got %d", fd)
	}
}
