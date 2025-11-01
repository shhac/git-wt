---
name: git-wt-test
description: This skill should be used when running tests, verifying test coverage, adding new tests, or debugging test failures for git-wt. Invoked when testing code, running test suites, writing tests, or diagnosing test issues.
allowed-tools: Read, Bash, Grep, Glob, Edit, Write
---

# git-wt Testing Workflow

Comprehensive testing guidance for git-wt including unit tests, integration tests, and interactive CLI testing.

## Instructions

### 1. Run Tests

**Recommended: Use build system**
```bash
zig build test              # Run unit tests
zig build test-integration  # Run integration tests
zig build test-all          # Run all tests (unit + integration)
```

**Alternative: Direct test commands**
```bash
zig test src/main.zig              # Run unit tests directly
zig test src/integration_tests.zig # Run integration tests directly
```

**Individual module tests:**
```bash
zig test src/utils/validation.zig  # Test specific module
zig test src/utils/lock.zig        # Test lock functionality
```

**Note:** `zig build test` may hang due to a known Zig issue with `--listen=-`. If this occurs, use the direct `zig test` commands above.

### 2. Test Coverage by Component

**Unit Tests (70+ tests):**
- `src/utils/` - All utility modules
- `src/commands/` - All command modules
- `src/tests/` - Regression tests

**Integration Tests (38 tests):**
- Inter-module interactions
- End-to-end workflows
- Edge case scenarios

**Interactive Tests:**
- Expect-based tests in `test-interactive/`
- Arrow-key navigation
- Multi-select functionality
- TTY interactions

### 3. Interactive CLI Testing

For interactive features (arrow keys, multi-select):

```bash
# Run all interactive tests
./test-interactive/run-all-tests.sh

# Run specific test
./test-interactive/test-navigation.exp   # Arrow-key navigation
./test-interactive/test-removal.exp      # Multi-select removal
./test-interactive/test-prunable.exp     # Prunable worktree handling
```

See `learnings/TESTING_INTERACTIVE_CLIS_WITH_EXPECT.md` for detailed guidance.

### 4. Non-Interactive Testing

For automated/CI testing:

```bash
# Non-interactive mode disables all prompts
git-wt -n new feature-branch      # Create without prompts
git-wt -n rm                      # Remove without confirmation
git-wt -n go                      # List worktrees only
git-wt -n go feature-branch       # Output: cd /path/to/worktree
```

### 5. Shell Integration Testing

Test the shell alias function:

```bash
# Test shell integration and fd3 mechanism
./scripts/test-shell-integration.sh
```

**Important:** The shell alias function does not persist across different `Bash` tool invocations. When testing, set up alias in the same session:

```bash
# WRONG - This won't work across multiple Bash tool calls:
# First call: eval "$(./zig-out/bin/git-wt --alias gwt)"
# Second call: gwt go  # This will fail - alias doesn't exist

# CORRECT - Set up alias in the same session:
eval "$(./zig-out/bin/git-wt --alias gwt)" && gwt go
```

### 6. Adding New Tests

When adding new functionality:

**File naming convention:**
- Unit tests: `src/commands/command_name_test.zig`
- Integration tests: Add to `src/integration_tests.zig`
- Interactive tests: `test-interactive/test-feature.exp`

**Register test file:**
```zig
// In src/commands/test_all.zig
test {
    _ = @import("new_test.zig");  // Add this line
}
```

**Test structure:**
```zig
const std = @import("std");
const testing = std.testing;
const module = @import("module.zig");

test "feature: description of what it tests" {
    const allocator = testing.allocator;

    // Setup
    // ...

    // Test
    // ...

    // Verify
    try testing.expect(condition);
}
```

### 7. Manual Testing Workflow

```bash
# Build
zig build

# Test manually
./zig-out/bin/git-wt new test-branch
cd ../repo-trees/test-branch
./zig-out/bin/git-wt go
./zig-out/bin/git-wt rm test-branch
```

### 8. Test Directory for Development

The `.e2e-test` directory is gitignored and reserved for:
- Creating test repositories during development
- Testing edge cases and experimental features
- Temporary test data in various states

**Never tracked in git** as it may contain incomplete git repositories or broken worktrees.

Example usage:
```bash
cd .e2e-test
git init test-repo
cd test-repo
git add . && git commit -m "initial"
../../zig-out/bin/git-wt new feature/auth
```

## Testing Best Practices

### Before Committing

Always run tests before committing:
```bash
zig test src/main.zig
zig build -Doptimize=ReleaseFast
```

### CI/CD Testing

GitHub Actions automatically runs:
- Unit tests on Ubuntu + macOS
- Integration tests
- Release builds
- Binary verification

### Test-Driven Development

1. Write failing test first
2. Implement feature
3. Verify test passes
4. Refactor if needed
5. Commit with test included

## Common Test Patterns

### Testing Git Operations

```zig
test "git operation: description" {
    const allocator = testing.allocator;

    // Create temporary git repo
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();

    // ... test git operations ...
}
```

### Testing User Input

```zig
test "input validation: description" {
    const valid_input = "feature-branch";
    try validation.validateBranchName(valid_input);

    const invalid_input = "feature branch";
    try testing.expectError(
        error.InvalidBranchName,
        validation.validateBranchName(invalid_input)
    );
}
```

### Testing File Operations

```zig
test "file operation: description" {
    const allocator = testing.allocator;

    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();

    // ... test file operations ...
}
```

## Debugging Test Failures

### Check Test Output

```bash
# Run single test with verbose output
zig test src/utils/validation.zig --summary all
```

### Use Debug Prints

```zig
test "debugging example" {
    const value = computeValue();
    std.debug.print("Debug value: {}\n", .{value});
    try testing.expect(value == expected);
}
```

### Check Memory Leaks

```zig
test "memory check" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer {
        const leaked = gpa.deinit();
        if (leaked == .leak) {
            std.debug.print("MEMORY LEAK DETECTED!\n", .{});
        }
    }
    const allocator = gpa.allocator();

    // ... test code ...
}
```

## Test Maintenance

### Keep Tests Updated

When refactoring:
- Update affected tests
- Ensure tests still validate behavior
- Add tests for new edge cases

### Remove Obsolete Tests

When removing features:
- Remove corresponding tests
- Update test_all.zig imports

### Document Test Purpose

Use clear, descriptive test names:
```zig
// GOOD
test "validation: rejects branch names with spaces"

// BAD
test "test1"
```

## References

For detailed guides:
- **TESTING.md** - Comprehensive testing documentation
- **learnings/TESTING_INTERACTIVE_CLIS_WITH_EXPECT.md** - Interactive testing with expect
- **learnings/HOW_TO_TEST_TTY_INPUTS.md** - Testing TTY interactions
- **learnings/NAVIGATION_AND_FD3.md** - fd3 mechanism technical details
