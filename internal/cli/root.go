package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/version"
)

var rootCmd = &cobra.Command{
	Use:           "git-wt",
	Short:         "Manage git worktrees with enhanced features",
	Long:          "git-wt creates worktrees in a sibling directory, copies project config, and offers interactive navigation between them.",
	Version:       version.Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and exits with a non-zero status on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
