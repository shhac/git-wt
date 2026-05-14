package config

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/shhac/git-wt/internal/git"
)

// Scope selects which gitconfig layer a read/write operates against.
// We deliberately omit --worktree for v1 — adding it is a future
// extension if anyone asks.
type Scope int

const (
	// ScopeLocal targets .git/config of the current repo.
	ScopeLocal Scope = iota
	// ScopeGlobal targets ~/.gitconfig (or $XDG_CONFIG_HOME/git/config).
	ScopeGlobal
)

func (s Scope) String() string {
	switch s {
	case ScopeLocal:
		return "local"
	case ScopeGlobal:
		return "global"
	}
	return "unknown"
}

// scopeFlag returns the git-config CLI flag for a scope.
func scopeFlag(s Scope) string {
	switch s {
	case ScopeGlobal:
		return "--global"
	default:
		return "--local"
	}
}

// Entry is one row in the effective-config table — the value, where it
// came from, and whether it's set anywhere at all.
type Entry struct {
	Key    *Key
	Value  string // raw value as stored (or "" if unset; consult IsSet)
	Source Scope  // only meaningful when IsSet is true
	IsSet  bool
}

// GetScoped reads the raw value at the given scope. Returns
// (value, true, nil) when set, ("", false, nil) when unset (git
// signals this with exit code 1), and a real error for any other
// failure (permission denied, malformed config file, missing git
// binary, etc.) so real problems surface instead of looking unset.
func GetScoped(ctx context.Context, k *Key, s Scope) (string, bool, error) {
	out, err := git.Run(ctx, "config", scopeFlag(s), "--get", k.FullName())
	if err != nil {
		var ee *git.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(out), true, nil
}

// GetEffective returns the merged value for a key, walking scopes
// most-specific-first and returning the first hit. Built-in defaults
// are NOT filled in here — callers that need a default should check
// Entry.IsSet.
func GetEffective(ctx context.Context, k *Key) (Entry, error) {
	for _, s := range []Scope{ScopeLocal, ScopeGlobal} {
		v, ok, err := GetScoped(ctx, k, s)
		if err != nil {
			return Entry{Key: k}, err
		}
		if ok {
			return Entry{Key: k, Value: v, Source: s, IsSet: true}, nil
		}
	}
	return Entry{Key: k}, nil
}

// Set writes value into the given scope after type-validating it.
// Templated string keys are checked for unknown ${...} vars with a
// dummy expansion — we don't want bad templates to sit silently in
// config and only blow up at command time.
func Set(ctx context.Context, k *Key, value string, s Scope) error {
	if err := Validate(k, value); err != nil {
		return err
	}
	_, err := git.Run(ctx, "config", scopeFlag(s), k.FullName(), value)
	return err
}

// Unset removes the key from the given scope. It's not an error if the
// key wasn't set — `git config --unset` exits 5 in that case, which we
// swallow so the UX is idempotent.
func Unset(ctx context.Context, k *Key, s Scope) error {
	_, err := git.Run(ctx, "config", scopeFlag(s), "--unset", k.FullName())
	if err == nil {
		return nil
	}
	var ee *git.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 5 {
		return nil
	}
	return err
}

// List returns one Entry per registered key with its effective value.
// Used by `git-wt config` to render the table.
func List(ctx context.Context) ([]Entry, error) {
	out := make([]Entry, 0, len(All))
	for _, k := range All {
		e, err := GetEffective(ctx, k)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// Validate type-checks a string value against a key's declared type
// and runs the key's optional per-key Validator. The type switch only
// enforces generic shape (parses as bool/int, templated strings have
// no unknown ${...} vars); anything narrower — ranges, enums, regex —
// lives on Key.Validator so each key's constraint is co-located with
// its declaration.
func Validate(k *Key, value string) error {
	switch k.Type {
	case TypeBool:
		if _, err := ParseBool(value); err != nil {
			return fmt.Errorf("config %s: %w", k.FullName(), err)
		}
	case TypeInt:
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("config %s: not an integer: %q", k.FullName(), value)
		}
	case TypeString:
		if k.Templated {
			// Surface unknown ${...} vars at set time with a dummy expansion.
			// We don't care about the resolved path here, only the error.
			_, err := ExpandPath(value, Vars{Repo: "_", RepoPath: "_", RepoParent: "_", Home: "_"})
			if err != nil {
				return fmt.Errorf("config %s: %w", k.FullName(), err)
			}
		}
	}
	if k.Validator != nil {
		if err := k.Validator(value); err != nil {
			return fmt.Errorf("config %s: %w", k.FullName(), err)
		}
	}
	return nil
}

// ParseBool follows git's accepted set of boolean strings.
func ParseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "yes", "on", "1":
		return true, nil
	case "false", "no", "off", "0", "":
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean %q (use true/false/yes/no/on/off/1/0)", s)
}
