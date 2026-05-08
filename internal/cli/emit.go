package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/shhac/git-wt/internal/fd"
	"github.com/shhac/git-wt/internal/ui"
)

// emitTarget delivers a worktree path to the caller. Used by go/new/rm —
// wrapper mode writes to fd N; bare mode prints to stdout with a copy/paste
// hint on stderr.
func emitTarget(path string) error {
	if w, ok := fd.Open(flagFD); ok {
		defer w.Close()
		_, err := fmt.Fprintln(w, path)
		return err
	}
	fmt.Println(path)
	arrow := "→"
	if ui.Plain {
		arrow = "->"
	}
	fmt.Fprintf(os.Stderr, "%s cd %s\n", arrow, shellQuote(path))
	return nil
}

// shellQuote single-quotes s for safe inclusion in a POSIX shell command.
// Single quotes inside s are emitted as `'\''`. Used by emitTarget and the
// alias generator.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
