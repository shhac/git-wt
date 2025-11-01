# git-wt Project

A Zig CLI tool for managing git worktrees with enhanced features like automatic setup, configuration copying, and interactive navigation.

## Project Overview

This is a Zig implementation providing enhanced git worktree management:
- `git-wt new <branch>` - Create worktree with automated setup
- `git-wt rm [branch...]` - Remove worktree(s) with safety checks
- `git-wt go [branch]` - Navigate between worktrees interactively
- `git-wt list` - List all worktrees with current indicator
- `git-wt alias <name>` - Generate shell function wrapper
- `git-wt clean` - Remove worktrees for deleted branches

**Version:** 0.4.2
**Zig Version:** 0.15.1+
**Platform Support:** macOS (Intel/ARM/Universal), Linux (x86_64/ARM64), Windows (via WSL2)

## Quick Reference

### Global Flags
- `-n, --non-interactive` - Run without prompts (for testing/scripting)
- `--no-tty` - Force number-based selection (disable arrow keys)
- `--no-color` - Disable colored output
- `--plain` - Plain output format (no colors, minimal formatting)
- `--debug` - Show diagnostic information
- `-h, --help` - Show help message
- `-v, --version` - Show version information

### Essential Documentation

**User Guides** (`docs/` directory):
- [INSTALLATION.md](docs/INSTALLATION.md) - Installation and setup
- [USAGE.md](docs/USAGE.md) - Basic usage with examples
- [CONFIGURATION.md](docs/CONFIGURATION.md) - Config file setup
- [SHELL-INTEGRATION.md](docs/SHELL-INTEGRATION.md) - Shell alias setup
- [ADVANCED.md](docs/ADVANCED.md) - Advanced features
- [TESTING.md](docs/TESTING.MD) - Testing guide
- [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) - Common issues

**Project Files:**
- [DESIGN.md](DESIGN.md) - Design principles and patterns
- [BUGS.md](BUGS.md) - Bug tracking (currently all resolved!)
- [TODO.md](TODO.md) - Planned features
- [CHANGELOG.md](CHANGELOG.md) - Version history

## Development Guidelines

### Git Workflow
- **Commit early and often** - Make commits as you complete logical units
- Use conventional commits: `gm feat cli "add argument parsing"`
- Run tests before committing: `zig test src/main.zig`
- Follow the workflow in Paul's global CLAUDE.local.md

### Code Style
- Use Zig standard library conventions
- Prefer explicit error handling over panics
- Keep functions focused and testable
- Extract common patterns into utility modules

### Architecture Overview

```
src/
├── main.zig              # Entry point, command dispatch
├── commands/             # Command implementations
│   ├── new.zig          # Create worktree with config copy
│   ├── remove.zig       # Remove worktrees with safety
│   ├── go.zig           # Navigate interactively
│   ├── list.zig         # List worktrees
│   ├── alias.zig        # Generate shell wrapper
│   └── clean.zig        # Clean deleted branches
└── utils/               # 14 utility modules
    ├── config.zig       # Configuration file support
    ├── git.zig          # Git command wrapper
    ├── interactive.zig  # Terminal UI (arrow keys, etc.)
    ├── validation.zig   # Input validation
    ├── lock.zig         # Concurrent operation locking
    └── ... (9 more utilities)
```

**For detailed architecture:** Use the `git-wt-architecture` skill or read inline code comments.

### Design Principles

From [DESIGN.md](DESIGN.md):

1. **Separation of Concerns** - Each command in own module, focused utilities
2. **Explicit Over Implicit** - All errors handled explicitly, memory management visible
3. **User Experience First** - Colors, prompts, progress indicators
4. **Safety by Default** - Confirmations, validation, comprehensive errors
5. **Testability** - Pure functions, non-interactive mode, 70+ tests
6. **Zero Runtime Dependencies** - Only git required, single binary

## Common Development Tasks

### Building and Testing

```bash
# Build
zig build -Doptimize=ReleaseFast

# Run tests
zig test src/main.zig              # Unit tests (70+)
zig test src/integration_tests.zig # Integration tests (38)

# Manual testing
./zig-out/bin/git-wt new test-branch
./zig-out/bin/git-wt --debug list
```

**For detailed testing:** Use the `git-wt-test` skill.

### Debugging

```bash
# Enable debug output
./zig-out/bin/git-wt --debug go

# Test specific scenarios
./debugging/test-claude-script.sh
```

**For debugging help:** Use the `git-wt-debug` skill.

### Creating Releases

**For release process:** Use the `git-wt-release` skill.

Quick reference:
1. Update CHANGELOG.md with version section
2. Update build.zig version
3. Run all tests
4. Commit: `gm chore release "bump version to X.Y.Z"`
5. Tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
6. Push: `git push origin main vX.Y.Z`
7. GitHub Actions automatically builds and publishes

## Specialized Skills Available

Claude has access to these git-wt-specific skills:

- **`git-wt-release`** - Complete release process automation
- **`git-wt-test`** - Testing workflows and procedures
- **`git-wt-debug`** - Debugging and troubleshooting
- **`git-wt-architecture`** - Codebase navigation and understanding
- **`git-wt-bugtrack`** - Bug tracking with BUGS.md
- **`git-wt-skill-validator`** - Validate and update skills to match current codebase state

These skills are automatically invoked when relevant. You can also ask Claude to use them explicitly.

**Skill Maintenance:** Use `git-wt-skill-validator` after significant refactoring or architecture changes to ensure skills stay synchronized with the codebase.

## Key Features

### Configuration File Support (v0.4.2)
- User-level: `~/.config/git-wt/config`
- Project-level: `.git-wt.toml`
- Precedence: CLI flags > env vars > project > user > defaults
- See [CONFIGURATION.md](docs/CONFIGURATION.md)

### GitHub Actions CI/CD (v0.4.2)
- Automated testing on push/PR
- Manual build artifacts workflow
- Automated releases on version tags
- All platforms built automatically

### Shell Integration
- fd3 mechanism for directory navigation
- Generated shell function wrapper
- Setup: `eval "$(git-wt alias gwt)"`
- See [SHELL-INTEGRATION.md](docs/SHELL-INTEGRATION.md)

### Safety Features
- Branch name validation
- Uncommitted changes detection
- File-based locking for concurrent operations
- Repository state validation (merge, rebase, bisect)
- Confirmation prompts with sensible defaults

### Interactive Features
- Arrow-key navigation (with number fallback)
- Multi-select removal with Space/Enter
- Smart sorting by modification time
- Terminal resize handling
- Graceful interrupt handling

## Dependencies and Tools

### Required
- **Zig 0.15.1+** - Programming language
- **Git** - Version control (intentional dependency)

### Build System
- `build.zig` - Zig build configuration
- `build.zig.zon` - Dependency management
- No external runtime dependencies

### CI/CD
- GitHub Actions workflows in `.github/workflows/`
- Automated testing, building, and releases
- Multi-platform binary generation

## Implementation Notes

### Zig Version Compatibility
- **Current:** Zig 0.15.1
- **Minimum:** Zig 0.15.1 (uses modern APIs)
- **Breaking Changes:** ArrayList.writer() now requires allocator parameter

### Code Patterns
- **Command Table:** Clean dispatch in main.zig
- **GitResult Union:** Explicit success/failure handling
- **Resource Management:** Consistent `defer` cleanup
- **Error Context:** Meaningful error propagation

### Testing Philosophy
- **70 unit tests** covering all modules
- **38 integration tests** for workflows
- **Expect-based tests** for interactive features
- **Non-interactive mode** for CI/CD

## Recent Changes

### v0.4.2 (Current - 2025-10-31)
- ✅ GitHub Actions CI/CD (testing, builds, releases)
- ✅ Configuration file support (user + project level)
- ✅ Added missing test file for clean command
- ✅ Fixed Zig version requirement in README

### v0.4.0-0.4.1
- ✅ Clean command for deleted branches
- ✅ JSON output for list command
- ✅ Bug fixes for duplicate symbols

See [CHANGELOG.md](CHANGELOG.md) for complete history.

## Getting Help

### For Development Issues
1. **Check skills:** Ask Claude to use relevant skill (release, test, debug, etc.)
2. **Check docs:** Comprehensive guides in `docs/` directory
3. **Check BUGS.md:** Known issues and solutions
4. **Enable debug:** Use `--debug` flag for diagnostics

### For User Issues
- See [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
- Use `--debug` flag
- Check GitHub Issues

## Project Status

**Production Ready:** ✅
**Test Coverage:** 70 unit tests + 38 integration tests
**Known Bugs:** 0 (46 fixed)
**CI/CD:** Automated via GitHub Actions
**Documentation:** Comprehensive (11 markdown files)

**Active Features:**
- All core commands fully functional
- Configuration file support
- Shell integration (fd3)
- Multi-platform releases
- Comprehensive error handling
- Interactive UI with fallbacks

**Planned Enhancements:** See [TODO.md](TODO.md)
