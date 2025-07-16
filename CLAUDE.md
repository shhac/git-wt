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

## Dependencies

Using established Zig libraries to minimize custom code:
- **zig-clap** (v0.10.0) - Command line argument parsing
- **ansi_term** - Terminal colors and ANSI escape sequences

## Implementation Notes

### Zig Version
- Requires Zig 0.14.1 or later
- Uses modern build.zig.zon for dependency management

### Design Philosophy
- Minimize custom code by using established libraries
- Focus on clear, maintainable implementation
- Match general features rather than exact shell script behavior

## Implementation Learnings

### Dependencies
- Started with zig-clap for CLI parsing but simplified to basic arg parsing due to compatibility issues
- Removed ansi_term dependency in favor of simple ANSI escape constants
- Lesson: Sometimes simpler is better - don't over-engineer with dependencies

### Code Organization
- Extracted common utilities (colors, input, process) to reduce duplication
- Used command table pattern in main.zig for cleaner command dispatch
- Helper functions like `trimNewline` and `fileExists` eliminate repeated patterns

### Zig-Specific Patterns
- Use `defer` for cleanup consistently
- Handle const-correctness carefully (e.g., `openDir` returns const Dir)
- Arena allocators work well for CLI tools
- Error unions and explicit error handling make code robust

### Testing
- Run `zig build test` to execute unit tests
- Manual testing in git repositories is essential
- Consider edge cases like being in main repo vs worktree

## Future Improvements
- Add shell completion scripts
- Consider configuration file support
- Add dry-run mode for commands
- Improve error messages with more context