# git-wt Project

A Zig CLI tool for managing git worktrees with enhanced features like automatic setup, configuration copying, and interactive navigation.

## Project Overview

This is a Zig implementation of the git-wt shell script, providing:
- `git-wt new <branch>` - Create a new worktree with automated setup
- `git-wt rm <branch>` - Remove worktree by branch name with safety checks
- `git-wt go [branch]` - Navigate between worktrees
- `git-wt list` - List all worktrees with current indicator

See [DESIGN.md](DESIGN.md) for the design principles and patterns used in this project.

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
│   └── go.zig        # Navigate between worktrees (interactive/direct)
└── utils/
    ├── git.zig       # Git command wrapper, repository info
    ├── fs.zig        # File operations, config copying
    ├── colors.zig    # ANSI color codes and formatted printing
    ├── input.zig     # User input utilities (confirmations, line reading)
    └── process.zig   # External command execution helpers

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
zig build test
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
- Requires Zig 0.14.1 or later
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
- Run `zig build test` to execute unit tests
- Manual testing in git repositories is essential
- Consider edge cases like being in main repo vs worktree

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
# Run all unit tests
zig build test
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
Created `test-non-interactive.sh` for automated testing:
```bash
./test-non-interactive.sh
```

The test script:
- Builds the binary
- Creates a temporary git repository
- Tests all commands in non-interactive mode
- Validates worktree creation/removal
- Cleans up automatically

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

## Future Improvements

See `TODO.md` for the current list of planned features and enhancements. Major items include:
- Custom worktree parent directory via command line flag
- Additional commands (clean)
- Configuration file support

Recently completed features:
- ✅ Support for branch names with slashes (creating subdirectory structures)
- ✅ List command to show all worktrees with current indicator
- ✅ Force flag for rm command (skip uncommitted changes check)
- ✅ Rm command now requires branch name argument