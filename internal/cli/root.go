package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/config"
	"github.com/shhac/git-wt/internal/debug"
	"github.com/shhac/git-wt/internal/ui"
)

// Global flag values, populated by Cobra before any command runs.
var (
	flagDebug          bool
	flagPlain          bool
	flagNonInteractive bool
	flagFD             int
)

var rootCmd = &cobra.Command{
	Use:           "git-wt",
	Short:         "Manage git worktrees with enhanced features",
	Long:          "git-wt creates worktrees in a sibling directory, copies project config, and offers interactive navigation between them.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		applyConfigDefaults(cmd)
		if flagPlain {
			ui.Plain = true
		}
		ui.Initialize()
		debug.Enabled = flagDebug
	},
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.BoolVar(&flagDebug, "debug", false, "verbose diagnostic output")
	pf.BoolVar(&flagPlain, "plain", false, "no color, minimal formatting (also honors NO_COLOR)")
	pf.BoolVarP(&flagNonInteractive, "non-interactive", "n", false, "disable interactive prompts (auto-detected when stdin is not a TTY)")
	pf.IntVar(&flagFD, "fd", 3, "file descriptor for shell-wrapper navigation protocol")
}

// Execute runs the root command and exits with a non-zero status on error.
// version is supplied by main (overridden via ldflags at release time).
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// interactive returns true if we should show prompts:
// - explicit --non-interactive always wins
// - otherwise, true iff stdin is a TTY
func interactive() bool {
	if flagNonInteractive {
		return false
	}
	return isatty.IsTerminal(os.Stdin.Fd())
}

// applyConfigDefaults layers git-config values under the global flags
// the user didn't explicitly pass. We use cmd.Flags().Changed because
// the zero value (false / 3) is a valid explicit setting — we can't
// just check "is it the default value?".
//
// Config reads happen here (in PreRun) rather than init time because
// they shell out to git, and we don't want `--help` to pay for that.
// Cobra skips PreRun for --help, so this stays cheap.
//
// Failures are silent: a malformed value in gitconfig shouldn't break
// every git-wt invocation. The user can debug with `git-wt config <key>`.
func applyConfigDefaults(cmd *cobra.Command) {
	applyBoolFlagDefault(cmd, "plain", config.Plain, &flagPlain)
	applyIntFlagDefault(cmd, "fd", config.FD, &flagFD)
}

// applyBoolFlagDefault assigns dest from the effective config value for
// k when the named flag wasn't passed explicitly. Bad config values are
// silently skipped (see applyConfigDefaults' note).
func applyBoolFlagDefault(cmd *cobra.Command, flagName string, k *config.Key, dest *bool) {
	raw, ok := readUnchangedFlagConfig(cmd, flagName, k)
	if !ok {
		return
	}
	v, err := config.ParseBool(raw)
	if err != nil {
		return
	}
	*dest = v
}

// applyIntFlagDefault is the int equivalent of applyBoolFlagDefault.
// We route the value through config.Validate so per-key constraints
// (e.g. FD's 3-9 range) live in one place — the Key declaration —
// rather than being re-derived here.
func applyIntFlagDefault(cmd *cobra.Command, flagName string, k *config.Key, dest *int) {
	raw, ok := readUnchangedFlagConfig(cmd, flagName, k)
	if !ok {
		return
	}
	if err := config.Validate(k, raw); err != nil {
		return
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return
	}
	*dest = n
}

// readUnchangedFlagConfig returns the raw config value for k iff the
// user did not pass the named flag and the key is set somewhere. It's
// the shared guard that both flag-default helpers above branch on.
func readUnchangedFlagConfig(cmd *cobra.Command, flagName string, k *config.Key) (string, bool) {
	if cmd.Flags().Changed(flagName) {
		return "", false
	}
	e, err := config.GetEffective(cmd.Context(), k)
	if err != nil || !e.IsSet {
		return "", false
	}
	return e.Value, true
}
