package cli

import (
	"context"

	"github.com/shhac/git-wt/internal/config"
	"github.com/shhac/git-wt/internal/wt"
)

// resolveParentDir returns the parent directory for a creation command,
// applying the precedence:
//
//  1. explicit --parent-dir flag wins
//  2. git config wt.parentDir (local beats global)
//  3. built-in default (<mainRoot>/.worktrees)
//
// Template expansion happens in wt.ResolveParentDir whenever the value
// is non-empty, so values from either step 1 or step 2 can use the
// ${...} substitutions.
func resolveParentDir(ctx context.Context, mainRoot, flagValue string) (string, error) {
	if flagValue != "" {
		return wt.ResolveParentDir(mainRoot, flagValue)
	}
	e, err := config.GetEffective(ctx, config.ParentDir)
	if err != nil {
		return "", err
	}
	return wt.ResolveParentDir(mainRoot, e.Value) // empty Value → default path
}
