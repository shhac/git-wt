// Package copyspec parses and applies a list of patterns describing which
// project files git-wt should copy into a freshly-created worktree.
//
// Format: gitignore-ish, top-level matching only.
//   - "# comment" — line is ignored
//   - "" (blank) — line is ignored
//   - "<pattern>" — include; pattern is glob-expanded from the repo root
//     (* ? [...] supported, no **)
//   - "!<pattern>" — exclude; subtracts matches from the include set
//   - trailing "/" — accepted, stripped before matching (informational only)
//
// Matches are top-level only: `.env*` matches files at the repo root that start
// with `.env`, not `subdir/.env`. To copy nested files, name them explicitly.
package copyspec

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Spec is a parsed copy specification.
type Spec struct {
	Includes []string
	Excludes []string
}

// Defaults returns the built-in fallback used when the spec file is absent.
// Mirrors what git-wt has always copied.
func Defaults() *Spec {
	return &Spec{
		Includes: []string{".env", ".env.*", ".claude", "CLAUDE.local.md", ".ai-cache"},
	}
}

// Load reads a spec from path. Returns Defaults() if the file is not present;
// any other error is reported. The caller is responsible for choosing the path
// (typically `<main_repo>/.git-wt-copy-files`).
func Load(path string) (*Spec, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Defaults(), nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return Parse(f)
}

// Parse reads patterns from r, one per line.
func Parse(r io.Reader) (*Spec, error) {
	s := &Spec{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "!") {
			s.Excludes = append(s.Excludes, strings.TrimSuffix(strings.TrimPrefix(line, "!"), "/"))
			continue
		}
		s.Includes = append(s.Includes, strings.TrimSuffix(line, "/"))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return s, nil
}

// Match returns the paths matched by the spec, relative to root, sorted and deduped.
// Glob expansion uses filepath.Glob (no `**`). Non-glob patterns that don't expand
// to anything but exist verbatim are still included (lets you list a literal path
// that has no glob chars).
func (s *Spec) Match(root string) ([]string, error) {
	set := make(map[string]struct{})

	for _, p := range s.Includes {
		paths, err := expand(root, p)
		if err != nil {
			return nil, err
		}
		for _, abs := range paths {
			rel, err := filepath.Rel(root, abs)
			if err != nil {
				continue
			}
			set[rel] = struct{}{}
		}
	}

	for _, p := range s.Excludes {
		paths, err := expand(root, p)
		if err != nil {
			return nil, err
		}
		for _, abs := range paths {
			rel, err := filepath.Rel(root, abs)
			if err != nil {
				continue
			}
			delete(set, rel)
		}
	}

	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

func expand(root, pattern string) ([]string, error) {
	full := filepath.Join(root, pattern)
	matches, err := filepath.Glob(full)
	if err != nil {
		return nil, fmt.Errorf("bad pattern %q: %w", pattern, err)
	}
	if len(matches) == 0 && !containsGlobChar(pattern) {
		if _, err := os.Lstat(full); err == nil {
			matches = []string{full}
		}
	}
	return matches, nil
}

func containsGlobChar(s string) bool {
	for _, r := range s {
		if r == '*' || r == '?' || r == '[' {
			return true
		}
	}
	return false
}
