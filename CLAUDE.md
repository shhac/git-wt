# git-wt Project

A Zig CLI tool for managing git worktrees with enhanced features like automatic setup, configuration copying, and interactive navigation.

## Project Overview

This is a Zig implementation of the git-wt shell script, providing:
- `git-wt new <branch>` - Create a new worktree with automated setup
- `git-wt rm` - Remove current worktree with safety checks
- `git-wt go [branch]` - Navigate between worktrees

## Development Guidelines

### Code Style
- Use Zig standard library conventions
- Prefer explicit error handling over panics
- Keep functions focused and testable

### Architecture
- `src/main.zig` - Entry point and command routing
- `src/commands/` - Command implementations
- `src/utils/` - Shared utilities (git, terminal, fs, node)

### Building
```bash
zig build
```

### Testing
```bash
zig build test
```

### Installation
```bash
zig build -Doptimize=ReleaseFast
cp zig-out/bin/git-wt ~/.local/bin/
```

## Next Steps
- Implement terminal color utilities
- Add git command wrapper
- Implement the three main commands
- Add interactive prompt handling