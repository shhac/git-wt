# Advanced Features

## Configuration Files

The following files and directories are automatically copied when creating new worktrees:
- `.claude` - Claude Code configuration
- `.env*` - All environment files (`.env`, `.env.local`, `.env.development`, `.env.test`, `.env.production`)
- `CLAUDE.local.md` - Local Claude instructions
- `.ai-cache` - AI cache directory (entire directory)

### CLAUDE Files Support
The tool has special support for Claude Code integration:
- Automatically detects Claude configuration files
- Preserves AI cache for faster Claude startup
- Maintains local instructions across worktrees
- Optional Claude auto-start after worktree creation (interactive mode only)

## How It Works

### Worktree Structure
Creates worktrees in a parallel directory structure:
```
parent-dir/
├── my-repo/          (main repository)
└── my-repo-trees/    (worktrees)
    ├── feature-a/
    ├── feature-b/
    ├── bugfix-123/
    └── feature/      (subdirectories for slash branches)
        ├── auth/
        └── ui/
            └── dark-mode/
```

### Configuration Syncing
Automatically copies important files that are typically gitignored but needed for development (env vars, editor configs, etc.)

### Smart Navigation
The `go` command sorts worktrees by modification time, making it easy to jump to recently used branches.

## Locking Mechanism

Protects against concurrent worktree operations:
- **Lock file location**: `.git/git-wt.lock`
- **Lock timeout**: 30 seconds
- **Stale lock detection**: Automatically cleans up locks from dead processes
- **Process ID tracking**: Stores PID and timestamp in lock files

## Branch Name Validation

Comprehensive validation following git standards:
- **Invalid characters**: No spaces, control characters, or `~^:?*[`
- **Path components**: No `.` or `..` components
- **Reserved names**: Rejects `HEAD`, `-`, `@`
- **Format rules**: No consecutive dots, proper start/end characters
- **Case sensitivity**: Detects conflicts on case-insensitive filesystems

## Path Sanitization

Handles complex branch names safely:
- **Slash conversion**: `feature/auth` → `feature-auth` for filesystem compatibility
- **Directory creation**: Supports subdirectory structures when using `--parent-dir`
- **Display consistency**: Shows relative names consistently across commands

## Performance Optimizations

### Large Repository Handling
For repositories with many worktrees (200+):
- **Early-exit branch search**: O(1) memory usage for direct branch lookup
- **Threshold-based loading**: Limits interactive selections to 50 items
- **Smart caching**: Avoids loading full worktree lists when unnecessary

### Memory Management
- **Arena allocators**: Used for command-scoped memory management
- **Explicit cleanup**: All memory allocations have corresponding `defer` statements
- **Resource isolation**: Each command manages its own memory lifecycle

## Project Structure

```
src/
├── main.zig                  # CLI entry point and command dispatch
├── commands/                 # Command implementations
│   ├── new.zig              # Create worktree with setup
│   ├── remove.zig           # Remove worktree with safety checks
│   └── go.zig               # Navigate between worktrees
├── utils/                   # Shared utilities
│   ├── git.zig              # Git operations and repository info
│   ├── fs.zig               # Filesystem helpers and config copying
│   ├── colors.zig           # Terminal colors and formatted output
│   ├── input.zig            # User input handling and confirmations
│   ├── process.zig          # External command execution
│   ├── validation.zig       # Branch name and path validation
│   ├── lock.zig             # File-based locking for concurrent operations
│   ├── fd.zig               # File descriptor 3 (fd3) shell integration
│   ├── interactive.zig      # Interactive UI with arrow key navigation
│   ├── time.zig             # Time formatting utilities
│   └── debug.zig            # Debug output utilities
└── integration_tests.zig    # Integration tests without git dependencies
```