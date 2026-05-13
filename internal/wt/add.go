package wt

import (
	"context"
	"fmt"
	"strings"

	"github.com/shhac/git-wt/internal/git"
)

// AddRefKind distinguishes how a user-supplied ref for `gwt add` resolves.
type AddRefKind int

const (
	// AddRefLocal: an existing local branch under refs/heads/.
	AddRefLocal AddRefKind = iota
	// AddRefRemote: an existing remote-tracking branch under refs/remotes/.
	// `git worktree add` will create a local branch tracking it.
	AddRefRemote
)

// AddRefResolution describes how a user-supplied ref was resolved.
type AddRefResolution struct {
	Kind AddRefKind
	// SourceRef is the exact string to pass through to `git worktree add`.
	SourceRef string
	// LocalName is the resulting local branch name — equal to the user's
	// input for local refs, and the rest-after-remote-prefix for remote refs.
	// Used to derive the default leaf when the user didn't supply one.
	LocalName string
}

// WorktreeAddArgs returns the argv to pass to `git worktree add` to
// materialise this resolution at path. For local refs that's a plain
// checkout; for remote refs we pass `--track -b <LocalName>` explicitly,
// because `git worktree add origin/foo` on its own creates a detached
// worktree rather than a tracking branch — git's DWIM only fires when
// the start-point is a bare branch name that doesn't yet exist locally.
func (r *AddRefResolution) WorktreeAddArgs(path string) []string {
	if r.Kind == AddRefRemote {
		return []string{"worktree", "add", "--track", "-b", r.LocalName, path, r.SourceRef}
	}
	return []string{"worktree", "add", path, r.SourceRef}
}

// ResolveAddRef classifies a user-supplied ref for `gwt add`.
//
// Rules, in order:
//  1. If ref is "<prefix>/<rest>" AND <prefix> matches an existing remote
//     AND refs/remotes/<prefix>/<rest> exists → remote (DWIM tracking).
//     The local branch git creates will be named <rest>.
//  2. Else if refs/heads/<ref> exists → local. LocalName == ref.
//  3. Else error.
//
// This means a slash-bearing local branch (e.g. "paul/auth-bug") resolves
// as local when no remote named "paul" exists. A no-slash arg can never
// hit rule 1, so local-only branches always win for those.
func ResolveAddRef(ctx context.Context, dir, ref string) (*AddRefResolution, error) {
	if ref == "" {
		return nil, fmt.Errorf("ref is empty")
	}

	if prefix, rest, ok := strings.Cut(ref, "/"); ok && prefix != "" && rest != "" {
		remoteOK, err := remoteExists(ctx, dir, prefix)
		if err != nil {
			return nil, err
		}
		if remoteOK {
			refOK, err := remoteRefExists(ctx, dir, prefix, rest)
			if err != nil {
				return nil, err
			}
			if refOK {
				return &AddRefResolution{
					Kind:      AddRefRemote,
					SourceRef: ref,
					LocalName: rest,
				}, nil
			}
		}
	}

	exists, err := BranchExists(ctx, dir, ref)
	if err != nil {
		return nil, err
	}
	if exists {
		return &AddRefResolution{
			Kind:      AddRefLocal,
			SourceRef: ref,
			LocalName: ref,
		}, nil
	}

	return nil, fmt.Errorf("no such branch or remote ref: %q", ref)
}

func remoteExists(ctx context.Context, dir, name string) (bool, error) {
	out, err := git.RunIn(ctx, dir, "remote")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == name {
			return true, nil
		}
	}
	return false, nil
}

func remoteRefExists(ctx context.Context, dir, remote, rest string) (bool, error) {
	_, err := git.RunIn(ctx, dir, "show-ref", "--verify", "--quiet", "refs/remotes/"+remote+"/"+rest)
	return err == nil, nil
}
