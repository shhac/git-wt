# Changelog

All notable changes to git-wt will be documented in this file.

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