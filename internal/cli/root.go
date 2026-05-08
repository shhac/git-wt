package cli

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/ui"
	"github.com/shhac/git-wt/internal/version"
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
	Version:       version.Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if flagPlain {
			ui.Plain = true
		}
		ui.Initialize()
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
func Execute() {
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
