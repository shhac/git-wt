package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/shhac/git-wt/internal/debug"
	"github.com/shhac/git-wt/internal/fd"
	"github.com/shhac/git-wt/internal/ui"
)

// emitTarget delivers a worktree path to the caller. Used by go/new/rm —
// wrapper mode writes to fd N; bare mode prints to stdout with a copy/paste
// hint on stderr.
//
// fd.Available can only ask the descriptor what access mode it claims, not
// whether it will accept a write. Environments leak descriptors that answer
// O_RDWR and then reject writes — a kqueue, a device that is open but not
// configured — so a failed write falls back to bare mode too. Reporting the
// path some other way always beats failing the command over it.
func emitTarget(path string) (err error) {
	end := debug.Op("emit-target", path)
	defer func() { end(err) }()

	w, ok := fd.Open(flagFD)
	if !ok {
		debug.Logf("wrapper fd %d not writable; falling back to bare mode", flagFD)
		return emitBare(path)
	}
	defer func() { _ = w.Close() }()

	if _, werr := fmt.Fprintln(w, path); werr != nil {
		debug.Logf("wrapper fd %d took no write (%v); falling back to bare mode", flagFD, werr)
		return emitBare(path)
	}
	return nil
}

// emitBare prints the path to stdout with a copy/paste hint on stderr, for
// callers not running under the shell wrapper. Supports `cd "$(git-wt go x)"`.
func emitBare(path string) error {
	if _, err := fmt.Println(path); err != nil {
		return err
	}
	arrow := "→"
	if ui.Plain {
		arrow = "->"
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s cd %s\n", arrow, shellQuote(path))
	return nil
}

// shellQuote single-quotes s for safe inclusion in a POSIX shell command.
// Single quotes inside s are emitted as `'\”`. Used by emitTarget and the
// alias generator.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
