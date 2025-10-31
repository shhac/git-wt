# Configuration

git-wt supports configuration files to customize default behavior without specifying flags every time.

## Configuration Locations

Configuration can be specified at two levels:

### 1. User-level Configuration
**Location:** `~/.config/git-wt/config`

Applies to all repositories for the current user.

```bash
mkdir -p ~/.config/git-wt
cp config.example.toml ~/.config/git-wt/config
# Edit with your preferences
```

### 2. Project-level Configuration
**Location:** `.git-wt.toml` in repository root

Applies only to the specific repository. Overrides user-level config.

```bash
cp config.example.toml .git-wt.toml
# Edit with project-specific settings
```

## Configuration Precedence

Settings are merged in the following order (highest priority first):

1. **Command-line flags** (highest priority)
2. **Environment variables** (e.g., `NO_COLOR`, `GWT_USE_FD3`)
3. **Project config** (`.git-wt.toml` in repo root)
4. **User config** (`~/.config/git-wt/config`)
5. **Built-in defaults** (lowest priority)

## Configuration Format

git-wt uses TOML format for configuration files.

### Example Configuration

See [`config.example.toml`](../config.example.toml) for a complete example with all available options.

## Configuration Options

### `[worktree]` Section

#### `parent_dir`
**Type:** String
**Default:** `"../{repo}-trees"`
**Description:** Default parent directory for worktrees.

**Special features:**
- `{repo}` placeholder: Replaced with repository name
- **Relative paths:** Resolved from repository root (e.g., `"worktrees"` → `/path/to/repo/worktrees`)
- **Absolute paths:** Used as-is (e.g., `"/tmp/worktrees"`)
- **Home expansion:** `~` expands to user home directory (e.g., `"~/code/worktrees"`)

**Examples:**
```toml
# Default behavior - sibling directory with -trees suffix
parent_dir = "../{repo}-trees"

# Centralized worktrees directory
parent_dir = "~/code/worktrees/{repo}"

# Project-specific location
parent_dir = "/tmp/project-worktrees"

# Within repository (relative path)
parent_dir = "worktrees"
```

**Command-line override:**
```bash
git-wt new feature --parent-dir /custom/path
```

---

### `[behavior]` Section

#### `auto_confirm`
**Type:** Boolean
**Default:** `false`
**Description:** Skip confirmation prompts (equivalent to `-f` or `--force` flag).

**Use case:** CI/CD environments where prompts would block execution.

```toml
[behavior]
auto_confirm = true
```

**Command-line override:**
```bash
git-wt rm --force    # Force confirmations
```

#### `non_interactive`
**Type:** Boolean
**Default:** `false`
**Description:** Disable interactive prompts and selections (equivalent to `-n` or `--non-interactive` flag).

**Use case:** Scripting and automation.

```toml
[behavior]
non_interactive = true
```

**Command-line override:**
```bash
git-wt go --non-interactive
```

#### `plain_output`
**Type:** Boolean
**Default:** `false`
**Description:** Disable colors and use plain text output (equivalent to `--plain` flag).

**Use case:** Log files, screen readers, or terminals without color support.

```toml
[behavior]
plain_output = true
```

**Command-line override:**
```bash
git-wt list --plain
```

#### `json_output`
**Type:** Boolean
**Default:** `false`
**Description:** Use JSON output format for `list` command (equivalent to `--json` flag).

**Use case:** Parsing output in scripts or tools.

```toml
[behavior]
json_output = true
```

**Command-line override:**
```bash
git-wt list --json
```

---

### `[ui]` Section

#### `no_color`
**Type:** Boolean
**Default:** `false`
**Description:** Disable ANSI colors (equivalent to `--no-color` flag or `NO_COLOR` environment variable).

**Use case:** Color blindness, terminals without color support.

```toml
[ui]
no_color = true
```

**Command-line override:**
```bash
git-wt list --no-color
```

#### `no_tty`
**Type:** Boolean
**Default:** `false`
**Description:** Force number-based selection instead of arrow keys (equivalent to `--no-tty` flag).

**Use case:** Environments without TTY support (CI/CD, remote shells).

```toml
[ui]
no_tty = true
```

**Command-line override:**
```bash
git-wt go --no-tty
```

---

### `[sync]` Section

#### `extra_files`
**Type:** Array of strings
**Default:** `[]`
**Description:** Additional files/directories to copy when creating worktrees.

**Built-in files (always copied):**
- `.env*` - Environment files
- `.claude` - Claude Code configuration
- `CLAUDE.local.md` - Local Claude instructions
- `.ai-cache` - AI cache directory

**Example:**
```toml
[sync]
extra_files = [
    ".vscode/settings.json",
    ".idea/",
    "docker-compose.override.yml",
    ".tool-versions",
    "Makefile.local"
]
```

**Note:** Paths are relative to repository root.

#### `exclude_files`
**Type:** Array of strings
**Default:** `[]`
**Description:** Files to exclude from syncing (useful for secrets).

**Example:**
```toml
[sync]
exclude_files = [
    ".env.production",
    ".env.secrets",
    ".claude/api-keys.json"
]
```

---

## Common Configuration Scenarios

### Scenario 1: CI/CD Environment

```toml
[behavior]
non_interactive = true
auto_confirm = true
plain_output = true

[ui]
no_color = true
no_tty = true
```

### Scenario 2: Team Project Setup

```toml
# .git-wt.toml in repository root
[worktree]
parent_dir = "../{repo}-worktrees"

[sync]
extra_files = [
    ".tool-versions",
    ".envrc",
    "docker-compose.override.yml"
]
exclude_files = [
    ".env.local"
]
```

### Scenario 3: Personal Centralized Worktrees

```toml
# ~/.config/git-wt/config
[worktree]
parent_dir = "~/dev/worktrees/{repo}"

[ui]
no_color = false
```

### Scenario 4: Accessibility Settings

```toml
[ui]
no_color = true
plain_output = true
```

---

## Environment Variables

In addition to configuration files, git-wt respects these environment variables:

| Variable | Description | Config Equivalent |
|----------|-------------|-------------------|
| `NO_COLOR` | Disable colors | `[ui] no_color = true` |
| `GWT_USE_FD3` | Internal: fd3 mechanism | N/A |
| `NON_INTERACTIVE` | Non-interactive mode | `[behavior] non_interactive = true` |

**Precedence:** Environment variables are overridden by command-line flags but override config files.

---

## Validation and Errors

### Invalid Configuration

If a configuration file has syntax errors, git-wt will:
1. Print a warning to stderr
2. Fall back to built-in defaults
3. Continue execution

**Example:**
```bash
Warning: Failed to parse config at ~/.config/git-wt/config: ParseError
Using default configuration.
```

### Debugging Configuration

Use `--debug` flag to see which config files are loaded:

```bash
git-wt --debug list
```

**Output includes:**
```
[Config]
  User config: ~/.config/git-wt/config (loaded)
  Project config: .git-wt.toml (not found)
  parent_dir: ~/code/worktrees/{repo}
  non_interactive: false
  ...
```

---

## Migration Guide

### From Command-line Flags

If you find yourself using the same flags repeatedly:

**Before:**
```bash
alias gwt-new='git-wt new --parent-dir ~/worktrees'
```

**After:**
```toml
# ~/.config/git-wt/config
[worktree]
parent_dir = "~/worktrees"
```

Now just use:
```bash
git-wt new feature-branch
```

---

## See Also

- [Usage Guide](USAGE.md) - Command reference
- [Advanced Features](ADVANCED.md) - Configuration syncing details
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues
