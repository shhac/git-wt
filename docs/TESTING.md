# Testing Guide

## Unit Tests

```bash
# Run all unit tests
zig build test
```

## Non-Interactive Mode

The tool supports a `--non-interactive` (or `-n`) flag for testing and automation:

```bash
# Create worktree without prompts
git-wt --non-interactive new feature-branch

# Remove worktree without confirmation
git-wt --non-interactive rm feature-branch

# List worktrees without interactive selection
git-wt --non-interactive go

# Navigate directly to a worktree (outputs cd command)
git-wt --non-interactive go feature-branch
```

## End-to-End Testing

Multiple test scripts are provided:

```bash
# Run non-interactive tests
./test-non-interactive.sh

# Test shell integration (requires shell alias setup)
./test-shell-integration.sh

# Run integration tests
zig build test-integration
```

## Test Scripts

The test scripts will:
- Build the binary
- Create temporary git repositories in `.e2e-test` directory
- Test all commands in various modes
- Validate actual outcomes (not just command execution)
- Clean up after themselves

### Test Directory

The `.e2e-test` directory is used for all test data and is gitignored to prevent test artifacts from being committed.

## Development Testing

```bash
# Run all tests
zig build test

# Run integration tests specifically
zig build test-integration

# Build debug version
zig build

# Run directly without installing
zig build run -- new test-branch

# Enable debug output
git-wt --debug new test-branch

# Build with custom version
zig build -Dversion="dev-1.0.0"
```

## Debug Mode

Enable debug output with the `--debug` flag to see detailed information about:
- Git operations and their output
- File system operations
- Lock acquisition and release
- Configuration file copying
- Process execution details

## Testing Concurrent Operations

The tool uses file-based locking to prevent concurrent worktree operations. Test this with:

```bash
# Run the concurrent test script
./test-concurrent.sh
```

This tests:
- Lock file creation in `.git/git-wt.lock`
- Automatic stale lock cleanup
- 30-second timeout for lock acquisition
- Clean error messages for lock conflicts