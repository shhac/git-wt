# Changelog

All notable changes to git-wt will be documented in this file.

## [0.6.2] - 2026-03-11

### Fixed
- **Nested worktree current detection** — when a worktree is nested inside another worktree (e.g., `.conductor/sofia` inside the main repo), `list` previously marked both as `(current)`; now only the most specific (deepest) path is marked current

## [0.6.1] - 2026-02-27

### Fixed
- **Namespaced branch display names** — branches with slashes (e.g., `aaaa/foo` and `bbbb/foo`) previously both displayed as `foo` in list, go, and rm pickers; now shows the path relative to the trees dir (`aaaa/foo`, `bbbb/foo`)

## [0.6.0] - 2026-02-27

### Added
- **Bare mode (dual-mode support)** — when running as `git wt` (without the shell alias), the tool now outputs worktree paths instead of attempting to change directories. At a TTY, shows a copy-paste hint (`→ cd '/path'`); when piped, outputs the raw path for scripting (`cd "$(git-wt go branch)"`)
- **Mode detection module** — new `mode.zig` detects wrapper vs bare mode once at startup based on `GWT_FD` presence, passed through all commands
- **`isStderrTty` / `isWriterTty` helpers** — interactive module can now check TTY status of any output writer
- **Bare-mode documentation** — SHELL-INTEGRATION.md and USAGE.md now document the scripting pattern and bare-mode behavior

### Fixed
- **`--no-color` respected in bare-mode hints** — the `→ cd '/path'` hint lines previously had hardcoded ANSI color codes that ignored `--no-color`; now properly gated
- **UTF-8 arrow fallback** — the `→` character in bare-mode hints now falls back to `->` on non-UTF-8 terminals (matches existing `✓`/`[OK]` pattern)
- **Signal handler async-signal-safety** — replaced mutex with atomic operations to prevent potential deadlock when SIGINT arrives during terminal state changes; uses `_exit(130)` instead of `exit(130)` to bypass atexit handlers
- **`new` command respects `--no-color`** — added `no_color` parameter to `new` command (previously only `go` and `list` supported it)

### Changed
- **`process.changeCurDir()` removed** — all 5 call sites (4 in `go`, 1 in `new`) removed; these were no-ops in bare mode and misleading in wrapper mode
- **Help text updated** — `go`, `new`, and top-level help now accurately describe both wrapper and bare mode behavior instead of claiming to "change the current working directory"
- **Interactive picker accepts output writer** — `selectFromList` / `selectMultipleFromList` / `selectFromListUnified` now take an explicit writer parameter, enabling future stderr rendering

### Performance
- **fd/TTY checks cached** — `isStdinTty()`, `isStdoutTty()`, and `fd.isEnabled()` results cached at function entry in `go` command instead of repeated syscalls (was 4-9 calls per invocation)
- **Duplicate computation eliminated** — `will_use_interactive` / `use_interactive` collapsed to single computation; dead `ui_writer` branch simplified

## [0.5.1] - 2026-02-21

### Added
- **Configurable fd number** — `git-wt alias gwt --fd 5` generates a shell function using fd 5 instead of the default fd 3; useful when fd3 is already in use by another tool
- **`GWT_FD` env var** — replaces `GWT_USE_FD3`; carries both "enabled" and "which fd" in one value (e.g., `GWT_FD=3`)

### Fixed
- **`gwt new` auto-navigation** — creating a worktree via the shell alias now navigates directly instead of showing the interactive picker (arg parser was consuming the branch name as the value of `--show-command`)

### Changed
- **`GWT_USE_FD3` removed** — replaced by `GWT_FD=N`; re-eval your alias to update: `eval "$(git-wt alias gwt)"`

## [0.5.0] - 2026-02-21

### Changed
- **`--help` and `--version` now output to stdout** — previously went to stderr, breaking `git-wt --help | less` and similar pipelines
- **Unknown flags now produce an error** — previously silently ignored; typos like `--josn` are now caught with a helpful message
- **Multi-select (rm) requires explicit selection** — pressing Enter without Space-toggling no longer auto-selects the highlighted item; prevents accidental deletion
- **Shell alias uses bash arrays for flags** — `$flags` string replaced with `"${flags[@]}"` array; fixes `--parent-dir` with paths containing spaces
- **Shell alias uses POSIX-compatible syntax** — replaced `[[ ]]` bashisms with `[ ]` and `case` for broader shell compatibility

### Fixed
- **Critical: `go` fast-path fd3 bug** — the `findWorktreeByBranch` optimization unconditionally wrote `cd` to stdout when fd3 was disabled; user saw raw text instead of navigating
- **Shell cd paths now single-quoted** — prevents shell expansion of `$`, backticks, and other metacharacters in worktree paths
- **`go --help` had wrong alias syntax** — said `git-wt --alias gwt` instead of `git-wt alias gwt`
- **`go` without shell alias now shows setup hint** — instead of silently changing only the subprocess directory, prints a tip about setting up the shell alias

### Added
- **Unknown flag validation** — all commands now validate flags and error on unrecognized ones
- **`--json`/`-j` documented in `list --help`** — was a supported but undiscoverable feature
- **`NO_COLOR` support in `clean` command** — respects the standard `NO_COLOR` environment variable
- **Configuration paths shown in `--help`** — `~/.config/git-wt/config` and `.git-wt.toml` now mentioned in help output
- **Removed unused dependencies** — `clap` and `ansi_term` removed from `build.zig.zon` (neither was imported)
- **Updated CI** — migrated to `mlugg/setup-zig`, Zig 0.15.2, `action-gh-release` v2

### Developer Notes
- Fixed `build.zig.zon` metadata: version now matches release, minimum_zig_version set to 0.15.1
- Updated `docs/INSTALLATION.md` to require Zig 0.15.1+

## [0.4.4] - 2026-02-21

### Fixed
- **Critical: `clean` command** — worktree removal was broken due to double "git" prefix in subprocess args (always failed silently)
- **Critical: Signal termination** — accessing process exit code without checking termination type caused undefined behavior when git was killed by signal (SIGKILL, SIGPIPE)
- **Critical: Shell navigation** — unquoted paths in fd3 `cd` commands broke `gwt go` for worktree paths containing spaces
- **`-n` flag conflict** — global `--non-interactive` intercepted `-n` before `clean` could use it for `--dry-run`; changed dry-run shortcut to `-d`
- **`findWorktreeByBranch`** — optimization path never matched because `refs/heads/` prefix wasn't stripped; function now works as intended
- **Timestamp underflow** — future-dated file timestamps (clock skew, NFS) caused panic; now clamped to 0 across 8 locations
- **Config boolean override** — project config couldn't override user config booleans back to `false`; config booleans now use three-state (`?bool`)
- **TOML inline comments** — unquoted config values included trailing `# comment` text; now stripped correctly
- **Memory leak** — `listWorktrees` missing errdefer for partial allocation cleanup on error
- **Missing `clean` in help** — `printUsage()` now includes the `clean` command
- **Dead code cleanup** — removed unused `withLock` function (had latent compile error), duplicate validation check

## [0.4.3] - 2025-11-01

### Added
- **Skill Validator** - New `git-wt-skill-validator` skill for maintaining documentation accuracy
  - Validates that project skills match current codebase state
  - Detects drift in architecture, test counts, command lists, and workflows
  - Supports quick validation, deep validation, and auto-update modes
  - Ensures documentation stays synchronized after refactoring

### Fixed
- **CI/CD Dependency Hash** - Updated `clap` dependency hash in `build.zig.zon` for version 0.11.0
  - Fixes GitHub Actions build failures caused by outdated package hash
  - All CI workflows now passing on Ubuntu and macOS

### Developer Notes
- Extracted detailed documentation from CLAUDE.md into specialized skills
- Skills now provide progressive disclosure of information
- Added `.gitignore` rules for `.claude/` directory with exceptions for skills/agents/commands

## [0.4.2] - 2025-10-31

### Added
- **GitHub Actions CI/CD**
  - Automated testing workflow runs on every push and pull request
  - Tests on Ubuntu and macOS with Zig 0.15.1
  - Manual build artifacts workflow allows on-demand builds for all platforms
  - Automated release workflow creates GitHub releases with platform binaries when version tags are pushed
  - All workflows build for: macOS Universal, macOS x86_64, macOS ARM64, Linux x86_64, Linux ARM64
- **Configuration File Support**
  - User-level configuration: `~/.config/git-wt/config`
  - Project-level configuration: `.git-wt.toml` in repository root
  - TOML format with comprehensive options
  - `[worktree]` section: `parent_dir` with `{repo}` substitution, relative/absolute path support
  - `[behavior]` section: `auto_confirm`, `non_interactive`, `plain_output`, `json_output`
  - `[ui]` section: `no_color`, `no_tty`
  - `[sync]` section: `extra_files`, `exclude_files` (arrays)
  - Precedence: CLI flags > environment variables > project config > user config > defaults
  - Graceful fallback to defaults if config files are missing or invalid
- **Documentation**
  - New `docs/CONFIGURATION.md` with comprehensive configuration guide
  - New `config.example.toml` with fully commented example configuration
  - Common configuration scenarios for CI/CD, teams, and personal use

### Fixed
- Added missing test file for `clean` command (`src/commands/clean_test.zig`)
- Updated README to correctly require Zig 0.15.1+ (was incorrectly showing 0.14.1+)

### Changed
- All command wrappers now accept configuration and merge with command-line flags
- Command-line flags always override configuration file settings

### Developer Notes
- Test coverage increased from 62 to 70 tests (8 new tests)
- Config module includes 5 unit tests for parsing and path resolution
- Clean command now has 3 unit tests

## [0.4.1] - 2025-10-31

### Fixed
- **Duplicate Symbols in Output**
  - Removed duplicate checkmarks in `clean` command output
  - Fixed 5 instances of duplicate symbols in `remove` command
  - Fixed duplicate "Error:" prefix in `new` command
  - `printSuccess()` and `printError()` already add symbols/prefixes automatically

## [0.4.0] - 2025-10-31

### Added
- **New `clean` Command**
  - Removes all worktrees for deleted branches
  - Lists worktrees to be removed before confirmation
  - Supports `--dry-run` flag to show what would be cleaned without removing
  - Supports `--force` flag to skip confirmation prompt
  - Properly handles memory management and error cases
- **JSON Output Format**
  - Added `--json` (or `-j`) flag to `list` command
  - Outputs structured JSON with branch, path, display_name, is_current, and last_modified fields
  - Properly escapes JSON strings for safe output
  - Returns empty array `[]` when no worktrees found

### Changed
- **Code Quality Improvements (Phase 1)**
  - Refactored `selectFromListUnified` function in interactive.zig (reduced from 291 to 212 lines, 27% reduction)
  - Extracted `renderInstructions` helper to eliminate code duplication
  - Consolidated lock acquisition error handling with `acquireWithUserFeedback` helper
  - Removed code smells: replaced `catch unreachable` with `try`, removed unused `execWithError` function
  - Enhanced maintainability through better code organization

## [0.3.1] - 2025-10-31

### Fixed
- **Display Name Bug in Interactive Navigation**
  - Fixed incorrect display names for worktrees in `gwt go` command
  - Previously used flawed "-trees" path heuristic to identify main repository
  - Now properly compares paths with repository root for accurate identification
  - Worktrees outside standard "-trees" directory now display correctly

### Technical Details
- Updated `listWorktreesWithTime` and `listWorktreesWithTimeSmart` functions
- Now use `getRepoInfo()` to get actual repository root path
- Exact path comparison replaces unreliable substring matching
- Deprecated `extractDisplayPath` function with documentation of limitations

## [0.3.0] - 2025-10-31

### Fixed
- **Interactive UI Rendering Improvements**
  - Fixed ghost menu items appearing after selection by simplifying exit cleanup
  - Fixed output flicker and progressive rendering by adding proper flush operations
  - Fixed instruction line redrawing issues during navigation
  - Improved window resize handling to preserve terminal context above menu

### Added
- **Terminal Compatibility Enhancements**
  - Added centralized terminal capability detection system (`terminal.zig`)
  - UTF-8 detection with automatic fallback to ASCII alternatives
  - Arrow key instructions now show "Up/Down" on non-UTF-8 terminals
  - Checkmark emoji (✓) now shows "[OK]" fallback on non-UTF-8 terminals
  - Support for NO_COLOR environment variable

### Changed
- **Code Quality Improvements**
  - Replaced inline ANSI escape codes with named constants from `colors.zig`
  - Simplified complex ANSI nesting by breaking into separate print statements
  - Refactored git module to use `GitResult` for better error handling
  - Added comprehensive rendering issue test suite in `debugging/` directory

### Developer Notes
- Added `src/utils/terminal.zig` for terminal capability detection
- New ANSI constants: `bold_off`, `dim`, `reverse`, `reverse_off`, `bright_green`
- Test scripts for rendering issues: `debugging/test-all-rendering.sh`
- Documentation updates to reflect accurate architecture

## [0.2.0] - 2025-08-27

### Changed
- **BREAKING**: Upgraded to Zig 0.15.1 (from 0.14.x)
  - Complete API migration for ArrayList, I/O, and Thread APIs
  - Updated build system for Zig 0.15 compatibility
- **BREAKING**: Removed Claude assistant integration from `new` command
  - Removed `--claude` flag and related configuration prompts
  - Simplified CLI to focus on core git worktree management

### Fixed
- Fixed `gwt list` error when encountering prunable/missing worktrees
  - Gracefully handles worktree directories that have been deleted
  - Shows "missing (prunable)" status for worktrees with missing directories
  - Displays "unknown" for modification time when directory doesn't exist
- Prunable worktrees can now be removed with `gwt rm`

### Added
- Comprehensive interactive testing suite using expect
  - Tests for arrow-key navigation, multi-select, and cancellation
  - Human-like timing simulation (25ms between keystrokes)
  - Screen capture capability for debugging
  - Test coverage for prunable worktree scenarios

### Developer Notes
- Added `test-interactive/` directory with expect-based test scripts
- Migration guide for Zig 0.15 API changes documented in codebase
- Improved error handling for file system operations

## [0.1.1] - 2025-01-24

### Fixed
- **Critical**: Fixed fd3 mechanism failure in shell alias due to incorrect `eval` usage
  - Shell alias now properly passes `GWT_USE_FD3` environment variable to subprocess
  - Navigation now works correctly in all interactive modes
  - Restored arrow-key navigation support (no longer forces `--no-tty`)

### Added
- Enhanced debug logging for fd3 mechanism (`--debug` flag in alias command)
- Regression tests to prevent eval-related issues from reoccurring
- Comprehensive learnings documentation about shell integration pitfalls

### Developer Notes
- Removed unnecessary `eval` from shell function generation
- Added debug output showing `cd_cmd` value even when empty
- Improved fd3 debugging with environment variable detection logs

## [0.1.0] - 2025-01-22

### Initial Release

#### Features
- **Core Commands**
  - `new` - Create worktrees with automatic branch creation and configuration copying
  - `rm` - Remove worktrees safely with multi-select support
  - `go` - Navigate between worktrees interactively with smart sorting
  - `list` - List all worktrees with details
  - `alias` - Generate shell functions for directory navigation

- **Interactive Features**
  - Arrow key navigation with automatic fallback to number selection
  - Multi-select removal with Space/Enter keys
  - Smart sorting by modification time (most recent first)
  - Terminal resize and interrupt handling
  - `--no-tty` flag for environments without TTY support

- **Configuration Syncing**
  - Automatically copies `.env*`, `.claude`, `CLAUDE.local.md`, `.ai-cache`
  - Preserves development environment across worktrees
  - Optional Claude auto-start after worktree creation

- **Safety Features**
  - Confirmation prompts for destructive operations
  - Uncommitted changes detection
  - Concurrent operation locking
  - Branch name validation
  - Process cleanup on worktree removal

- **Performance Optimizations**
  - Early-exit branch search for large repositories
  - Threshold-based loading (200+ worktrees)
  - Smart caching for interactive selections
  - O(1) memory usage for direct lookups

- **Shell Integration**
  - File descriptor 3 (fd3) mechanism for clean shell integration
  - Works with any shell (bash, zsh, fish, etc.)
  - Custom parent directory support with `{repo}` template

#### Platform Support
- macOS (Intel and Apple Silicon) with universal binary
- Linux (x86_64 and ARM64)
- Windows via WSL2

#### Technical Details
- Written in Zig 0.14.1 for zero runtime dependencies
- ~550KB optimized binary size
- Comprehensive test coverage
- Memory-safe with explicit error handling