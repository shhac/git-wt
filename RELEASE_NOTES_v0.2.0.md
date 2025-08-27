# git-wt v0.2.0 Release Notes

## Major Changes

### 🚀 Upgraded to Zig 0.15.1
- Complete migration from Zig 0.14.x to 0.15.1
- Updated all APIs for Zig 0.15 compatibility
- Fixed ArrayList, I/O, and Thread API changes

### ✨ Improved Worktree Management
- **Fixed prunable worktree handling**: `gwt list` now gracefully handles missing worktree directories
- Shows "missing (prunable)" status for worktrees with deleted directories
- Can remove prunable worktrees with `gwt rm`

### 🧪 Comprehensive Interactive Testing
- Added expect-based test suite for interactive CLI features
- Tests cover arrow-key navigation, multi-select, and cancellation
- Human-like timing simulation (25ms between keystrokes)
- Screen capture capability for debugging

### 🔧 Removed Claude Integration
- Removed the Claude assistant integration from `new` command
- Simplified the CLI to focus on core git worktree management

## Bug Fixes

- Fixed `gwt list` error when encountering prunable worktrees
- Handle missing worktree directories without crashing
- Properly display modification time as "unknown" for missing directories

## Testing Improvements

- Added `test-interactive/` directory with expect scripts
- Test coverage for:
  - Navigation with arrow keys
  - Multi-select removal
  - Prunable worktree cleanup
  - ESC cancellation

## Breaking Changes

- Removed `--claude` flag from `new` command
- Removed Claude-related configuration prompts

## Installation

```bash
# Build from source
zig build -Doptimize=ReleaseFast

# Install
cp zig-out/bin/git-wt ~/.local/bin/

# Set up shell alias
eval "$(git-wt --alias gwt)"
```

## Requirements

- Zig 0.15.1 or later
- Git 2.30.0 or later (for worktree support)

## Contributors

- Paul Somers (@shhac)

---

Full commit list: v0.1.1...v0.2.0