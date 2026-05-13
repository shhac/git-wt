package wt

// Stash helpers for commands that need to safely move work between
// worktrees / branches. Wraps `git stash` with the conventions git-wt
// relies on: capture by SHA (not @{N}, which shifts under concurrent
// pushes), include untracked files, restore index state on apply.

import (
	"context"
	"fmt"
	"strings"

	"github.com/shhac/git-wt/internal/git"
)

// StashApplyOutcome distinguishes the three observable end-states of a
// stash apply, so callers can tailor messaging without parsing errors.
type StashApplyOutcome int

const (
	// StashApplied: working tree + index restored, stash entry dropped.
	StashApplied StashApplyOutcome = iota
	// StashAppliedKeptStash: working tree + index restored, but the
	// follow-up drop failed (e.g. the stash list changed unexpectedly).
	// The user's work is in place; the stash entry just lingers.
	StashAppliedKeptStash
	// StashApplyFailed: apply returned non-zero. May be conflicts
	// (working tree has merge markers) or a harder failure. The stash
	// entry is preserved for recovery.
	StashApplyFailed
)

// StashPush captures the current working tree (incl. untracked) as a
// stash with the given message and returns the stash commit SHA. The
// SHA is used by [StashApply] and [StashDropBySHA] so that the operation
// is independent of the `stash@{N}` index, which can shift if another
// process pushes a stash before we apply.
//
// Caller invariant: nothing between `git stash push` and `git rev-parse
// stash@{0}` is allowed to yield to a concurrent git process that could
// push another stash. The two commands run back-to-back here.
func StashPush(ctx context.Context, message string) (string, error) {
	if _, err := git.Run(ctx, "stash", "push", "--include-untracked", "-m", message); err != nil {
		return "", err
	}
	sha, err := git.Run(ctx, "rev-parse", "stash@{0}")
	if err != nil {
		return "", fmt.Errorf("capture stash ref: %w", err)
	}
	return strings.TrimSpace(sha), nil
}

// StashApply restores the stash identified by sha into the working tree
// at dir (use "" for the caller's cwd). Both working tree contents and
// index state are restored (apply --index). On a clean apply, the stash
// entry is dropped by SHA. On any failure, the entry is preserved so the
// user can recover.
//
// The returned outcome encodes whether (a) the apply succeeded and (b)
// the drop succeeded. The error is only meaningful when outcome is
// StashAppliedKeptStash (drop failed — usually harmless) or
// StashApplyFailed (apply failed — caller should warn the user).
func StashApply(ctx context.Context, dir, sha string) (StashApplyOutcome, error) {
	var err error
	if dir == "" {
		_, err = git.Run(ctx, "stash", "apply", "--index", sha)
	} else {
		_, err = git.RunIn(ctx, dir, "stash", "apply", "--index", sha)
	}
	if err != nil {
		return StashApplyFailed, err
	}
	if dropErr := StashDropBySHA(ctx, sha); dropErr != nil {
		return StashAppliedKeptStash, dropErr
	}
	return StashApplied, nil
}

// StashDropBySHA scans `git stash list` for the entry matching sha and
// drops it. Matching by SHA (not by @{N}) avoids the race where another
// process pushes a stash between our apply and our drop, which would
// otherwise shift @{N} and cause us to drop the wrong entry.
func StashDropBySHA(ctx context.Context, sha string) error {
	out, err := git.Run(ctx, "stash", "list", "--format=%H %gd")
	if err != nil {
		return err
	}
	ref, ok := findStashRefBySHA(out, sha)
	if !ok {
		return fmt.Errorf("stash %s not found in stash list", ShortStashRef(sha))
	}
	_, err = git.Run(ctx, "stash", "drop", ref)
	return err
}

// findStashRefBySHA parses `git stash list --format=%H %gd` output and
// returns the `stash@{N}` ref whose commit SHA matches sha. Pure parser
// for table-driven testing — extracted so the SHA-to-ref lookup can be
// exercised independently of git.
func findStashRefBySHA(listOutput, sha string) (string, bool) {
	for _, line := range strings.Split(listOutput, "\n") {
		fields := strings.SplitN(line, " ", 2)
		if len(fields) != 2 {
			continue
		}
		if fields[0] == sha {
			return fields[1], true
		}
	}
	return "", false
}

// ShortStashRef returns the first 8 chars of a stash commit SHA for
// user-facing messages. Returns the input unchanged if it's shorter.
func ShortStashRef(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
