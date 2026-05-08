// Package git is a thin subprocess wrapper around the git CLI.
package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Run executes git with args from the current working directory.
// Returns trimmed stdout; on failure, returns stderr in the error.
func Run(ctx context.Context, args ...string) (string, error) {
	return RunIn(ctx, "", args...)
}

// RunIn runs git in dir (use "" for the current working directory).
func RunIn(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			msg := strings.TrimSpace(string(ee.Stderr))
			if msg == "" {
				msg = err.Error()
			}
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}
