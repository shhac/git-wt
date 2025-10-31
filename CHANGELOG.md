# Changelog

All notable changes to git-wt will be documented in this file.

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