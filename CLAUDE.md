# git-wt Project

A Zig CLI tool for managing git worktrees with enhanced features like automatic setup, configuration copying, and interactive navigation.

## Project Overview

This is a Zig implementation of the git-wt shell script, providing:
- `git-wt new <branch>` - Create a new worktree with automated setup
- `git-wt rm [branch...]` - Remove worktree(s) by branch name with safety checks
- `git-wt go [branch]` - Navigate between worktrees (interactive or direct)
- `git-wt list` - List all worktrees with current indicator
- `git-wt alias <name>` - Generate shell function wrapper for directory navigation

See [DESIGN.md](DESIGN.md) for the design principles and patterns used in this project.
See [docs/](docs/) for comprehensive user guides and advanced documentation.

### Global Flags
All commands support these global flags:
- `-n, --non-interactive` - Run without prompts (for testing/scripting)
- `--no-tty` - Force number-based selection (disable arrow keys)
- `--no-color` - Disable colored output
- `--plain` - Plain output format (no colors, minimal formatting)
- `--debug` - Show diagnostic information
- `-h, --help` - Show help message for command
- `-v, --version` - Show version information

## Documentation

Comprehensive guides available in the `docs/` directory:
- **[INSTALLATION.md](docs/INSTALLATION.md)** - Installation and setup instructions
- **[USAGE.md](docs/USAGE.md)** - Basic usage guide with examples
- **[ADVANCED.md](docs/ADVANCED.md)** - Advanced features and workflows
- **[SHELL-INTEGRATION.md](docs/SHELL-INTEGRATION.md)** - Shell integration setup (gwt alias)
- **[TESTING.md](docs/TESTING.md)** - Testing guide for developers
- **[TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)** - Common issues and solutions

Additional learning resources in `learnings/`:
- **[TESTING_INTERACTIVE_CLIS_WITH_EXPECT.md](learnings/TESTING_INTERACTIVE_CLIS_WITH_EXPECT.md)** - Guide to testing interactive CLIs
- **[NAVIGATION_AND_FD3.md](learnings/NAVIGATION_AND_FD3.md)** - Technical details on fd3 mechanism
- **[HOW_TO_TEST_TTY_INPUTS.md](learnings/HOW_TO_TEST_TTY_INPUTS.md)** - Testing TTY interactions

See also:
- **[DESIGN.md](DESIGN.md)** - Design principles and patterns
- **[BUGS.md](BUGS.md)** - Known bugs and edge cases (currently all resolved!)
- **[TODO.md](TODO.md)** - Planned features and improvements
- **[CHANGELOG.md](CHANGELOG.md)** - Version history and release notes

## Development Guidelines

### Git Workflow
- **Commit early and often** - Make commits as you complete logical units of work
- Use conventional commits with `gm` (e.g., `gm feat cli "add argument parsing"`)
- Don't wait until the end to commit - commit after each feature/fix is working
- Run tests before committing to ensure changes don't break existing functionality

### Code Style
- Use Zig standard library conventions
- Prefer explicit error handling over panics
- Keep functions focused and testable
- Extract common patterns into utility modules

### Architecture

```
src/
├── main.zig           # Entry point, command dispatch using command table pattern
├── commands/
│   ├── new.zig       # Create worktree with setup (config copy)
│   ├── remove.zig    # Remove worktree with safety checks
│   ├── go.zig        # Navigate between worktrees (interactive/direct)
│   ├── list.zig      # List all worktrees
│   └── alias.zig     # Generate shell function wrappers
└── utils/
    ├── args.zig      # Command-line argument parsing
    ├── colors.zig    # ANSI color codes and formatted printing
    ├── debug.zig     # Debug logging functionality
    ├── env.zig       # Environment variable utilities
    ├── fd.zig        # File descriptor 3 (fd3) shell integration
    ├── fs.zig        # File operations, config copying
    ├── git.zig       # Git command wrapper, repository info
    ├── input.zig     # User input utilities (confirmations, line reading)
    ├── interactive.zig # Interactive terminal control (arrow keys, raw mode)
    ├── io.zig        # File I/O wrappers (Zig 0.15+ compatibility)
    ├── lock.zig      # File locking for concurrent operations
    ├── process.zig   # External command execution helpers
    ├── time.zig      # Time formatting utilities
    └── validation.zig # Branch name validation

### Typical Development Workflow

1. Make changes to the code
2. Run `zig build` to check compilation
3. Run `zig build test` to ensure tests pass
4. Test manually with `./zig-out/bin/git-wt [command]`
5. Commit with `gm [type] [scope] "description"`:
   ```bash
   gm feat new "add branch validation"
   gm fix go "handle missing worktrees"
   gm test utils "add tests for trimNewline"
   gm refactor cli "simplify argument parsing"
   ```

### Building
```bash
zig build
```

### Testing
```bash
# Recommended: Use the build system
zig build test              # Run unit tests
zig build test-integration  # Run integration tests
zig build test-all          # Run all tests (unit + integration)

# Alternative: Direct test commands (bypasses build options)
zig test src/main.zig              # Run unit tests directly
zig test src/integration_tests.zig # Run integration tests directly

# Individual module tests
zig test src/utils/validation.zig  # Test specific module
zig test src/utils/lock.zig        # Test lock functionality

# Note: `zig build test` may hang due to a known Zig issue with `--listen=-`
# If this occurs, use the direct `zig test` commands above
```

### Installation
```bash
zig build -Doptimize=ReleaseFast
cp zig-out/bin/git-wt ~/.local/bin/
```

## Dependencies

Originally planned to use external libraries but simplified:
- **zig-clap** - Added to build.zig.zon but not actively used (simplified to basic arg parsing)
- **ansi_term** - Removed in favor of simple ANSI constants

## Implementation Notes

### Zig Version
- Requires Zig 0.13.0 or later (actively developed and tested with 0.15.1)
- Uses modern build.zig.zon for dependency management

### Design Philosophy
- Minimize custom code by using established libraries
- Focus on clear, maintainable implementation
- Match general features rather than exact shell script behavior

## Implementation Learnings

### Version Control Best Practices
- Commit after each working feature, not at the end of the session
- Use conventional commits to maintain clear history
- Small, focused commits are easier to review and debug
- Run tests before committing to catch issues early

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
- Always free memory from exec() calls in git.zig
- Watch for memory leaks with GeneralPurposeAllocator in debug mode

### Testing
- **Important**: `zig build test` may hang due to a known Zig issue with `--listen=-`
- Use direct test commands instead (see Testing section above)
- Manual testing in git repositories is essential
- Consider edge cases like being in main repo vs worktree
- All tests use temporary directories to avoid side effects

## Robustness and Safety Features

### Input Validation
- **Branch name validation**: Rejects invalid characters, reserved names, and improper formatting
- **Path validation**: Checks for existing worktree paths to prevent conflicts
- **Branch existence checks**: Prevents creating worktrees for branches that already exist

### Git State Validation
- **Repository state checks**: Detects ongoing merges, rebases, cherry-picks, and bisects
- **Uncommitted changes detection**: Warns when removing worktrees with uncommitted work
- **Clean operation guarantees**: Ensures git operations happen in clean repository states

### Error Handling
- **Graceful failure modes**: All errors provide clear, actionable feedback to users
- **Process isolation**: Claude spawning uses detached processes to prevent hangs
- **Resource cleanup**: Proper memory management and file handle cleanup throughout

### Testing and Verification
- **Comprehensive test suite**: 13 automated tests covering normal and edge cases
- **Outcome verification**: Tests actually verify that operations succeed (not just that commands run)
- **Non-interactive mode**: Full CLI functionality available for scripting and automation

## Key Features Implemented

### Shell Integration
- **--alias command** generates shell functions to handle directory changes
- Since CLI tools can't change the parent shell's pwd, the alias wraps git-wt
- Enables commands like `gwt go branch` to actually change directories
- Setup: `eval "$(git-wt --alias gwt)"` in .zshrc or .bashrc

### File Descriptor 3 (fd3) Integration
The shell integration uses a clever fd3 mechanism to enable directory changes:

- **fd3 mechanism** enables the CLI subprocess to communicate directory changes back to the parent shell
- Environment variable `GWT_USE_FD3=1` signals that fd3 is available and should be used
- Commands write shell commands (like `cd /path`) to file descriptor 3
- The shell wrapper evaluates captured commands to change directories
- Implementation documented in: `src/utils/fd.zig` with detailed technical explanation

**How it works:**
1. Shell function opens fd3 for writing: `3>&1`
2. Sets environment variable: `GWT_USE_FD3=1`
3. Runs git-wt binary with fd3 available
4. Binary writes `cd /path` to fd3
5. Shell evaluates the captured command
6. Parent shell's directory changes

This mechanism is necessary because subprocess commands cannot change the parent shell's working directory.

### Worktree Management
- Creates worktrees in `../repo-trees/branch-name` structure
- Automatic branch creation with `-b` flag
- Safe removal with confirmation prompts
- Interactive navigation with modification time sorting

### Configuration Copying
When creating a new worktree, automatically copies:
- `.claude` - Claude Code configuration
- `.env*` - All environment files
- `CLAUDE.local.md` - Local Claude instructions  
- `.ai-cache` - AI cache directory


### User Experience
- **Colored output** for better visibility and status clarity
- **Interactive prompts** with sensible defaults (Y/n patterns)
- **Comprehensive help system** with per-command help (`git-wt new --help`)
- **Clear error messages** with context and suggested actions
- **Progress indicators** for long operations like yarn install

## Testing Approach

### Unit Tests
All utility modules have comprehensive unit tests:
```bash
# Run all unit tests (avoiding zig build test hang issue)
zig test src/main.zig

# This runs tests for all modules imported by main.zig
# which includes all utility modules via test_all.zig
```

### Non-Interactive Mode
Added `--non-interactive` (or `-n`) flag for automated testing:
- Skips all prompts and confirmations
- Disables interactive selection in `go` command
- Returns machine-readable output where appropriate
- Essential for CI/CD and scripting

```bash
# Examples
git-wt -n new feature-branch      # Create without prompts
git-wt -n rm                      # Remove without confirmation
git-wt -n go                      # List worktrees only
git-wt -n go feature-branch       # Output: cd /path/to/worktree
```

### End-to-End Testing

**Interactive tests** using expect:
```bash
# Run all interactive tests
./test-interactive/run-all-tests.sh

# Run specific test
./test-interactive/test-navigation.exp   # Arrow-key navigation
./test-interactive/test-removal.exp      # Multi-select removal
./test-interactive/test-prunable.exp     # Prunable worktree handling
```

**Shell integration tests:**
```bash
# Test shell integration and fd3 mechanism
./scripts/test-shell-integration.sh
```

**Debugging scripts** (in `debugging/` directory):
- `debug-interactive-fd3.sh` - Debug fd3 mechanism interactively
- `test-claude-script.sh` - Scratch script for quick testing
- `test-user-fd3.sh` - Test user's actual shell setup

### Manual Testing
```bash
# Build and test manually
zig build
./zig-out/bin/git-wt new test-branch
cd ../repo-trees/test-branch
./zig-out/bin/git-wt go
./zig-out/bin/git-wt rm test-branch
```

### Testing Shell Alias Function
**Important**: The shell alias function does not persist across different `Bash` tool invocations. When testing the alias function, you must set it up in the same session:

```bash
# WRONG - This won't work across multiple Bash tool calls:
# First call: eval "$(./zig-out/bin/git-wt --alias gwt)"
# Second call: gwt go  # This will fail - alias doesn't exist

# CORRECT - Set up alias in the same session:
eval "$(./zig-out/bin/git-wt --alias gwt)" && gwt go
```

This is a limitation of the Bash tool execution model where each tool call runs in a separate shell session.

### Test Directory for Development
The `.e2e-test` directory is gitignored and reserved for:
- Creating test repositories during development
- Testing edge cases and experimental features
- Any temporary test data that might be in a broken state

This directory should NEVER be tracked in git as it may contain:
- Incomplete git repositories
- Broken worktrees
- Test data in various states of completion

Use this directory freely for manual testing during development:
```bash
# Example: Testing branch with slashes
cd .e2e-test
git init test-repo
cd test-repo
git add . && git commit -m "initial"
../../zig-out/bin/git-wt new feature/auth
```

## Bug Tracking

We maintain a `BUGS.md` file that tracks known bugs, edge cases, and potential issues in the codebase. When reviewing code or encountering issues:

1. **Check BUGS.md first** - The issue might already be documented
2. **Add new bugs** - Document any new issues you find with:
   - Clear description of the problem
   - Impact on users or system
   - Example scenarios that trigger the bug
   - Suggested fix approach
3. **Categorize appropriately** - Use categories like Critical Issues, Edge Cases, Usability Issues, etc.
4. **Reference DESIGN.md** - When fixing bugs, ensure solutions conform to our design principles:
   - Zero runtime dependencies
   - Clear, maintainable code
   - Proper error handling
   - Cross-platform compatibility

To create or update BUGS.md:
```bash
# Review code systematically
grep -r "TODO\|FIXME\|XXX" src/
# Look for error handling patterns
grep -r "catch\|error\|panic" src/
# Check for memory management
grep -r "allocator\|free\|defer" src/
```

## TODO Management

There is a `TODO.md` file that tracks planned features and improvements. When implementing items from the TODO:

1. **Before starting**: Review the TODO.md to understand the full scope
2. **During implementation**: Update the TODO item with progress notes if needed
3. **After completion**: Remove the completed item from TODO.md
4. **Important**: Always commit the TODO.md update as part of the feature implementation

This ensures the TODO list stays current and reflects actual work remaining.

## Testing Interactive CLI Features

For comprehensive guidance on testing interactive CLI features, see:
- **[learnings/TESTING_INTERACTIVE_CLIS_WITH_EXPECT.md](learnings/TESTING_INTERACTIVE_CLIS_WITH_EXPECT.md)** - Complete guide to testing with expect
- **[test-interactive/](test-interactive/)** - Actual test suite with examples

### Quick Summary

**Best approach**: Use `expect` with screen capture and human-like timing (25ms between keystrokes).

```bash
# Run all interactive tests
./test-interactive/run-all-tests.sh

# Test specific features
./test-interactive/test-navigation.exp   # Arrow-key navigation
./test-interactive/test-removal.exp      # Multi-select removal
./test-interactive/test-prunable.exp     # Prunable worktree handling
```

The project supports two interactive modes:
1. **Arrow-key navigation** (default with TTY) - Uses ANSI escape codes
2. **Number-based selection** (`--no-tty` or fallback) - Works everywhere

For non-interactive testing: `./zig-out/bin/git-wt --non-interactive [command]`

## Terminal Compatibility

### Supported Terminals

**Full Support (colors + UTF-8 + interactive):**
- macOS Terminal.app
- iTerm2
- GNOME Terminal
- Konsole
- Windows Terminal
- xterm (modern)

**Partial Support (colors + ANSI, limited UTF-8):**
- Linux console (Ctrl+Alt+F2)
- Older xterm versions
- tmux/screen multiplexers

**Minimal Support (basic text only):**
- Non-TTY output (pipes, redirects)
- TERM=dumb
- Very old terminals

### Fallback Modes

The tool automatically detects terminal capabilities and adjusts:

**Interactive Mode:**
- **Arrow-key navigation** - Default with full TTY support
- **Number selection** - Automatic fallback with `--no-tty` or when arrow keys unavailable

**Visual Elements:**
- **Colors** - Disabled with `--no-color` or when TERM=dumb
- **UTF-8 symbols** - Fallback to ASCII on terminals without UTF-8 support
- **Emojis** - Display when UTF-8 available, use text alternatives otherwise

### Environment Variables

- `TERM` - Terminal type detection (xterm-256color, dumb, etc.)
- `LANG` / `LC_CTYPE` - UTF-8 support detection
- `GWT_USE_FD3` - Enable fd3 shell integration mechanism (set automatically by alias)

### Testing Terminal Compatibility

```bash
# Test with minimal terminal
TERM=dumb ./zig-out/bin/git-wt list

# Test without UTF-8
LANG=C ./zig-out/bin/git-wt go

# Test with explicit flags
./zig-out/bin/git-wt --no-color --no-tty go
```

## Debugging

### Debug Mode

Enable debug output with the `--debug` flag:
```bash
# See fd3 mechanism details
./zig-out/bin/git-wt --debug go

# See git command execution
./zig-out/bin/git-wt --debug new feature-branch

# Combine with other flags
./zig-out/bin/git-wt --debug --show-command go
```

### Debugging Scripts

The `debugging/` directory contains helpful scripts:

**Interactive debugging:**
```bash
# Debug fd3 mechanism with your actual shell setup
./debugging/test-user-fd3.sh

# Debug interactive selection
./debugging/debug-interactive-fd3.sh

# Check user's alias configuration
./debugging/check-user-alias.sh
```

**Scratch testing:**
```bash
# Edit this script for quick experimentation
./debugging/test-claude-script.sh
```

### Common Issues

**Arrow keys not working:**
- Check if `--no-tty` flag is set (forces number selection)
- Verify terminal supports raw mode input
- Try explicit TTY mode by running directly (not via pipe)

**Colors not showing:**
- Check `TERM` environment variable is set
- Use `--debug` to see capability detection
- Try explicit color mode with `--color` (when implemented)

**fd3 mechanism not working:**
- Ensure using the shell alias: `eval "$(git-wt --alias gwt)"`
- Check `GWT_USE_FD3` is set when running via alias
- Run `./debugging/test-user-fd3.sh` to diagnose

**Performance issues:**
- Check repository size (very large repos may be slow)
- Use `--debug` to see which operations take time
- Consider using direct branch name instead of interactive selection

See **[TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)** for comprehensive troubleshooting guide.

## Future Improvements

See `TODO.md` for the current list of planned features and enhancements. Major items include:
- Additional commands (clean, sync, prune)
- Configuration file support (.git-wt.toml)
- Better error recovery for interrupted operations

Recently completed features:
- ✅ Support for branch names with slashes (creating subdirectory structures)
- ✅ List command to show all worktrees with current indicator
- ✅ Force flag for rm command (skip uncommitted changes check)
- ✅ Custom worktree parent directory via --parent-dir flag
- ✅ Upgraded to Zig 0.15.1 (from 0.13.0 minimum)
- ✅ Removed Claude integration from new command
- ✅ Comprehensive test suite (13 tests + interactive expect tests)
- ✅ File-based locking for concurrent operations
- ✅ Repository state validation (merge, rebase, bisect detection)
- ✅ Interactive multi-select removal
- ✅ Arrow-key navigation with fallback to number selection