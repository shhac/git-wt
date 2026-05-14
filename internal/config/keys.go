// Package config is the typed front door for git-wt's user/repo
// settings. Values are stored in git config under the `wt.*` namespace,
// so users can set them with vanilla `git config` if they want — the
// `git-wt config` subcommand layers on validation, template-variable
// resolution, and per-key documentation.
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// FDMin and FDMax bound the wrapper-protocol file descriptor. Used by
// FD.Validator, by the alias command's bake-time check, and by
// applyConfigDefaults' read-time guard — all three reference these
// constants so the bound has a single source of truth.
const (
	FDMin = 3
	FDMax = 9
)

// Type tags a Key's expected value shape.
type Type int

const (
	TypeString Type = iota
	TypeBool
	TypeInt
)

func (t Type) String() string {
	switch t {
	case TypeString:
		return "string"
	case TypeBool:
		return "bool"
	case TypeInt:
		return "int"
	}
	return "unknown"
}

// Key is the schema entry for one config setting.
type Key struct {
	// Name is the unqualified key, e.g. "parentDir" (full git config key
	// is "wt." + Name).
	Name string
	// Type is the expected value shape. Set time validates against this.
	Type Type
	// Default is the string form of the built-in default, shown in help.
	// The actual default may be computed (e.g. parentDir resolves to
	// <repo>/.worktrees when unset) — Default here is for documentation.
	Default string
	// Doc is a one-line description for help output.
	Doc string
	// Templated indicates that ${...} substitution is applied to stored
	// values when read.
	Templated bool
	// Validator is an optional per-key constraint applied after the
	// generic type check. Use this for ranges, enum values, regex
	// matches, etc. — anything beyond "parses as the declared Type".
	// Co-locates the rule with the key declaration so a new key with a
	// new constraint doesn't touch the generic Validate function.
	Validator func(string) error
}

// FullName returns the git-config key, e.g. "wt.parentDir".
func (k *Key) FullName() string { return "wt." + k.Name }

// ParentDir controls where new worktrees go. Default behaviour (unset
// in git config) is the historical `<mainRoot>/.worktrees`. When set, the
// value is template-expanded against the current repo before use.
var ParentDir = &Key{
	Name:      "parentDir",
	Type:      TypeString,
	Default:   "${repoPath}/.worktrees",
	Doc:       "Parent directory for new worktrees. Supports ${...} substitution.",
	Templated: true,
}

// Plain mirrors the --plain flag — useful for users who never want
// colour. NO_COLOR is honored independently.
var Plain = &Key{
	Name:    "plain",
	Type:    TypeBool,
	Default: "false",
	Doc:     "Always run with --plain (also honors NO_COLOR).",
}

// FD is the fallback fd for the wrapper protocol when --fd isn't given.
// The 3-9 range matches POSIX shell limits on numeric fd literals and
// is shared via the FDMin/FDMax constants — see ValidateFD.
var FD = &Key{
	Name:      "fd",
	Type:      TypeInt,
	Default:   strconv.Itoa(FDMin),
	Doc:       fmt.Sprintf("Default fd for the wrapper protocol (%d-%d).", FDMin, FDMax),
	Validator: ValidateFD,
}

// ValidateFD enforces the FD range. Exposed so callers outside the
// config-set path (e.g. the alias command's bake-time check) can use
// the same rule without duplicating the literal.
func ValidateFD(value string) error {
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("not an integer: %q", value)
	}
	if n < FDMin || n > FDMax {
		return fmt.Errorf("fd must be %d-%d (got %d)", FDMin, FDMax, n)
	}
	return nil
}

// All is the registry. Keep alphabetical for stable help output.
var All = []*Key{FD, ParentDir, Plain}

// Find returns the Key matching name (case-insensitive, with or without
// the "wt." prefix). Falls back to nil if unknown — callers should
// surface that as "unknown config key".
func Find(name string) *Key {
	name = strings.TrimPrefix(strings.ToLower(name), "wt.")
	for _, k := range All {
		if strings.EqualFold(k.Name, name) {
			return k
		}
	}
	return nil
}
