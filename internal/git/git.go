// Package git is a thin subprocess wrapper around the git CLI.
package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/shhac/git-wt/internal/debug"
)

// ExitError is what Run/RunIn returns when git exits non-zero. It
// preserves the underlying *exec.ExitError (via Unwrap) so callers
// can use errors.As to inspect the exit code — git uses meaningful
// codes for some commands (e.g. `git config --unset` returns 5 when
// the key is missing, 1 for most other errors).
type ExitError struct {
	Args   []string
	Stderr string // trimmed stderr from git
	inner  *exec.ExitError
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), e.Stderr)
}

// ExitCode returns the underlying process exit code, or -1 if the
// inner ExitError is nil (shouldn't happen in practice).
func (e *ExitError) ExitCode() int {
	if e.inner == nil {
		return -1
	}
	return e.inner.ExitCode()
}

// Unwrap exposes the underlying *exec.ExitError so callers can also
// reach it via errors.As(err, &execErr).
func (e *ExitError) Unwrap() error { return e.inner }

// Run executes git with args from the current working directory.
// Returns trimmed stdout; on failure, returns stderr in the error.
func Run(ctx context.Context, args ...string) (string, error) {
	return RunIn(ctx, "", args...)
}

// RunIn runs git in dir (use "" for the current working directory).
func RunIn(ctx context.Context, dir string, args ...string) (out string, err error) {
	end := debug.Op("git", args)
	defer func() { end(err) }()

	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	stdout, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			msg := strings.TrimSpace(string(ee.Stderr))
			if msg == "" {
				msg = err.Error()
			}
			err = &ExitError{Args: args, Stderr: msg, inner: ee}
			return "", err
		}
		err = fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		return "", err
	}
	return strings.TrimRight(string(stdout), "\n"), nil
}
