# git-wt Project

A Zig CLI tool for managing git worktrees with enhanced features like automatic setup, configuration copying, and interactive navigation.

## Project Overview

This is a Zig implementation of the git-wt shell script, providing:
- `git-wt new <branch>` - Create a new worktree with automated setup
- `git-wt rm` - Remove current worktree with safety checks
- `git-wt go [branch]` - Navigate between worktrees

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
│   ├── new.zig       # Create worktree with setup (config copy, nvm, yarn)
│   ├── remove.zig    # Remove worktree with safety checks
│   └── go.zig        # Navigate between worktrees (interactive/direct)
└── utils/
    ├── git.zig       # Git command wrapper, repository info
    ├── fs.zig        # File operations, config copying, Node.js detection
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

### Node.js Integration
- Detects and runs `nvm use` if `.nvmrc` exists
- Detects yarn in package.json and runs `yarn install`
- Proper PATH rehashing after nvm changes

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
./zig-out/bin/git-wt rm
./zig-out/bin/git-wt go
```

## Future Improvements
- Add shell completion scripts
- Consider configuration file support  
- Add dry-run mode for commands
- Improve error messages with more context
- Add `--force` flag for rm command
- Support for custom worktree locations
- Integration with git aliases