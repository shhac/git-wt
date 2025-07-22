# Usage Guide

## Basic Commands

### Create a new worktree

```bash
git-wt new feature-branch

# Branch names with slashes are supported and create subdirectories
git-wt new feature/auth-system
git-wt new bugfix/issue-123
```

This will:
1. Create a new worktree at `../repo-trees/feature-branch` (or subdirectories for slashes)
2. Create and checkout the new branch
3. Copy configuration files from the main repository
4. Optionally start Claude

### Remove a worktree

```bash
# Remove a specific worktree by branch name
git-wt rm feature-branch

# Remove multiple worktrees at once
git-wt rm branch1 branch2 branch3

# Force removal (skip uncommitted changes check)
git-wt rm feature-branch --force

# Non-interactive mode (skip confirmation prompts)
git-wt rm feature-branch --non-interactive
```

### Navigate to a worktree

```bash
# Interactive mode - shows all worktrees sorted by modification time
git-wt go

# Direct navigation
git-wt go main              # Go to main repository
git-wt go feature-branch    # Go to specific worktree

# Additional options
git-wt go --no-tty          # Force number-based selection
git-wt go --show-command    # Output shell cd commands
git-wt go --no-color        # Disable colored output
git-wt go --plain           # Output plain paths only
```

### List all worktrees

```bash
# List all worktrees with details
git-wt list

# Plain output (machine-readable)
git-wt list --plain

# Without colors
git-wt list --no-color
```

## Interactive Features

### Navigation Controls
- **Arrow keys (↑/↓)**: Navigate between options
- **Enter**: Confirm selection
- **ESC/Q**: Cancel operation
- **Space**: 
  - In `rm` command: Toggle selection ([*]/[ ])
  - In `go` command: No action (single-select only)

### Multi-Select (rm command)
- Use **Space** to toggle multiple selections
- Selected items show `[*]`, unselected show `[ ]`
- **Enter** confirms all selected items
- If nothing selected when pressing Enter, current item is selected

### Terminal Features
- Automatic fallback to number-based selection when TTY unavailable
- Terminal resize handling (SIGWINCH support)
- Real-time display updates
- Graceful handling of interrupted operations

## Global Options

Available for all commands:

```bash
git-wt --help                    # Show general help
git-wt --version, -v             # Show version information
git-wt --debug                   # Enable debug output
git-wt --non-interactive, -n     # Disable all interactive prompts
git-wt --no-tty                  # Force number-based selection
```

## Command-Specific Options

### `git-wt new`
```bash
git-wt new <branch-name>
  -h, --help                     # Show command help
  -n, --non-interactive          # Skip all prompts (no Claude startup)
  -p, --parent-dir <path>        # Custom parent directory for worktree
```

### `git-wt rm`
```bash
git-wt rm [branch-name...]
  -h, --help                     # Show command help
  -n, --non-interactive          # Skip confirmation prompts
  -f, --force                    # Skip uncommitted changes check
```

### `git-wt go`
```bash
git-wt go [branch-name]
  -h, --help                     # Show command help
  -n, --non-interactive          # List worktrees without interaction
  --no-tty                       # Force number-based selection
  --show-command                 # Output shell cd commands
  --no-color                     # Disable colored output
  --plain                        # Output plain paths only
```

### `git-wt alias`
```bash
git-wt alias <name> [options]
  -h, --help                     # Show command help
  --no-tty                       # Forward --no-tty flag to all commands
  -n, --non-interactive          # Forward --non-interactive flag
  --plain                        # Forward --plain flag
  -p, --parent-dir <path>        # Set default parent directory
```