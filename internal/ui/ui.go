// Package ui owns terminal styling for git-wt: colors, alignment, and the
// global plain/no-color toggle. lipgloss does the heavy lifting.
package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Plain disables colors and bold styling. Set once from the CLI flag handler;
// defaults to off (colored).
var Plain bool

// Initialize applies environment defaults. Currently honors NO_COLOR.
// Call once during command setup, before any Style() use.
func Initialize() {
	if os.Getenv("NO_COLOR") != "" {
		Plain = true
	}
}

// Predefined styles. Each respects Plain by lazily checking the flag.
var (
	current = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true) // green + bold
	dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // bright black
	branch  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))            // cyan
)

// Current renders text in the "current worktree" style.
func Current(s string) string {
	if Plain {
		return s
	}
	return current.Render(s)
}

// Dim renders text in a muted style (used for secondary info like paths/times).
func Dim(s string) string {
	if Plain {
		return s
	}
	return dim.Render(s)
}

// Branch renders text in the "branch name" style (cyan).
func Branch(s string) string {
	if Plain {
		return s
	}
	return branch.Render(s)
}
