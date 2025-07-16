# git-wt

A Zig-based CLI tool for managing git worktrees with enhanced features like automatic setup, configuration copying, and interactive navigation.

## Features

- **Create worktrees** with automatic branch creation and setup
- **Remove worktrees** safely with branch cleanup options
- **Navigate between worktrees** interactively or directly
- Automatic copying of configuration files (.env, .claude, etc.)
- Node.js project support (nvm, yarn detection)
- Colored terminal output for better UX

## Installation

### Build from source

Requirements:
- Zig 0.14.1 or later

```bash
git clone https://github.com/yourusername/git-wt.git
cd git-wt
zig build -Doptimize=ReleaseFast
cp zig-out/bin/git-wt ~/.local/bin/
```

## Usage

### Create a new worktree

```bash
git-wt new feature-branch
```

This will:
1. Create a new worktree at `../repo-trees/feature-branch`
2. Create and checkout the new branch
3. Copy configuration files from the main repository
4. Run `nvm use` if .nvmrc exists
5. Run `yarn install` if package.json with yarn is detected
6. Optionally start Claude

### Remove current worktree

```bash
git-wt rm
```

This will:
1. Confirm you're in a worktree (not main repository)
2. Navigate back to the main repository
3. Remove the worktree
4. Optionally delete the associated branch

### Navigate to a worktree

```bash
# Interactive mode - shows all worktrees sorted by modification time
git-wt go

# Direct navigation
git-wt go main              # Go to main repository
git-wt go feature-branch    # Go to specific worktree
```

## Configuration Files

The following files are automatically copied when creating new worktrees:
- `.claude` - Claude Code configuration
- `.env`, `.env.local`, `.env.development`, `.env.test`, `.env.production`
- `CLAUDE.local.md` - Local Claude instructions
- `.ai-cache` - AI cache directory

## Development

```bash
# Run tests
zig build test

# Build debug version
zig build

# Run directly
zig build run -- new test-branch
```

## License

MIT