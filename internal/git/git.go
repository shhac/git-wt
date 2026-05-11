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
			err = fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
			return "", err
		}
		err = fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		return "", err
	}
	return strings.TrimRight(string(stdout), "\n"), nil
}
