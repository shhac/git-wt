package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Vars holds the substitution values for ${...} expansion in path-shaped
// config values. They're derived from the repository the command is
// running against; see VarsFor.
type Vars struct {
	Repo       string // basename of MainRoot, e.g. "git-wt"
	RepoPath   string // absolute MainRoot, e.g. "/Users/paul/projects-personal/git-wt"
	RepoParent string // filepath.Dir(MainRoot)
	Home       string // $HOME (sidesteps `~` quirks for stored values)
}

// VarsFor builds the substitution vars for a given main-worktree root.
func VarsFor(mainRoot string) Vars {
	return Vars{
		Repo:       filepath.Base(mainRoot),
		RepoPath:   mainRoot,
		RepoParent: filepath.Dir(mainRoot),
		Home:       os.Getenv("HOME"),
	}
}

// known returns the canonical list of template-variable names for help
// text and error messages. Kept in one place to stay in sync with the
// switch below.
var known = []string{"repo", "repoPath", "repoParent", "home"}

// KnownVars returns the names of variables ExpandPath understands.
func KnownVars() []string {
	out := make([]string, len(known))
	copy(out, known)
	return out
}

// ExpandPath substitutes ${...} variables in template against vars and
// returns the result. Unknown variables make this return an error
// listing every offender plus the supported set — we deliberately fail
// loud rather than leaving `${typo}` literal in the resulting path,
// where it would only blow up at worktree-creation time.
//
// Use `$$` to embed a literal `$`.
func ExpandPath(template string, vars Vars) (string, error) {
	bad := map[string]struct{}{}
	out := os.Expand(template, func(k string) string {
		switch k {
		case "repo":
			return vars.Repo
		case "repoPath":
			return vars.RepoPath
		case "repoParent":
			return vars.RepoParent
		case "home":
			return vars.Home
		case "$":
			// `$$` in the template → literal `$` in the output.
			// os.Expand parses the second `$` as a one-char name; we
			// echo it back rather than treating it as an unknown var.
			return "$"
		}
		bad[k] = struct{}{}
		return ""
	})
	if len(bad) > 0 {
		names := make([]string, 0, len(bad))
		for k := range bad {
			names = append(names, k)
		}
		sort.Strings(names)
		return "", fmt.Errorf("unknown template variable(s): %s (known: %s)",
			strings.Join(names, ", "), strings.Join(known, ", "))
	}
	return out, nil
}
