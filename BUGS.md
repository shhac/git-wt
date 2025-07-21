# Known Bugs and Edge Cases

This file tracks known bugs, edge cases, and potential issues in the git-wt codebase.

## Critical Issues

### 1. Memory Management
- **Issue**: In `git.zig:listWorktreesWithTime()`, we duplicate worktree strings but if any allocation fails in the loop, the errdefer only cleans up items in the ArrayList, not any partially created WorktreeWithTime that failed
- **Impact**: Potential memory leak on allocation failure
- **Fix**: Need better error handling during the construction loop

### 2. Signal Handler Race Condition
- **Issue**: In `interactive.zig`, the signal handler modifies global state without proper synchronization
- **Impact**: Potential race condition if signal arrives during terminal state modification
- **Fix**: Need atomic operations or better locking strategy

## Edge Cases

### 3. Path Handling
- **Issue**: Branch names with special characters (spaces, quotes, etc.) may not be handled correctly
- **Impact**: Could create worktrees with unexpected names or fail
- **Example**: `git-wt new "branch with spaces"`
- **Fix**: Need proper escaping/validation

### 4. Concurrent Worktree Operations
- **Issue**: No locking mechanism when creating/removing worktrees
- **Impact**: Race conditions if multiple git-wt instances run simultaneously
- **Fix**: Implement file-based locking or check git's own locking

### 5. Repository State Validation
- **Issue**: Limited checking for repository states (bare repos, submodules, etc.)
- **Impact**: Unexpected behavior in non-standard git setups
- **Fix**: Add more comprehensive repository state checks

## Usability Issues

### 6. Error Messages
- **Issue**: Some error messages don't provide enough context (e.g., "Failed to create worktree")
- **Impact**: Users can't diagnose issues easily
- **Fix**: Add more detailed error messages with suggestions

### 7. Interactive Mode Edge Cases
- **Issue**: Terminal size changes during interactive selection not handled
- **Impact**: Display corruption if terminal is resized
- **Fix**: Handle SIGWINCH signal

### 8. Shell Integration
- **Issue**: The fd3 mechanism for shell integration is fragile and undocumented
- **Impact**: Users may not understand why commands behave differently
- **Fix**: Better documentation and error handling

## Code Quality Issues

### 9. Inconsistent Error Handling
- **Issue**: Mix of try/catch patterns and error returns without clear strategy
- **Impact**: Makes code harder to maintain and reason about
- **Fix**: Establish consistent error handling patterns

### 10. Missing Input Validation
- **Issue**: Several commands don't validate input thoroughly
- **Impact**: Cryptic errors or unexpected behavior
- **Examples**:
  - No validation for --parent-dir paths
  - No validation for branch name length limits
  - No handling of relative paths in some cases

### 11. Resource Cleanup
- **Issue**: Some file handles and processes may not be cleaned up on early returns
- **Impact**: Resource leaks
- **Fix**: Audit all resource allocations for proper cleanup

## Performance Issues

### 12. Redundant Git Calls
- **Issue**: Multiple calls to git for information that could be cached
- **Impact**: Slower performance, especially on large repositories
- **Fix**: Cache git information within a single command execution

### 13. Large Repository Handling
- **Issue**: Loading all worktrees into memory at once
- **Impact**: High memory usage with many worktrees
- **Fix**: Implement pagination or streaming

## Platform-Specific Issues

### 14. Windows Compatibility
- **Issue**: Path handling assumes Unix-style paths in several places
- **Impact**: May not work correctly on Windows
- **Fix**: Use std.fs.path functions consistently

### 15. Case-Insensitive Filesystems
- **Issue**: No handling of case-insensitive filesystem issues
- **Impact**: Could create conflicting worktrees on macOS/Windows
- **Fix**: Add filesystem capability detection

## Security Concerns

### 16. Command Injection
- **Issue**: Building shell commands with string concatenation
- **Impact**: Potential command injection with malicious branch names
- **Fix**: Use proper command argument arrays everywhere

### 17. Path Traversal
- **Issue**: While validateParentDir checks for `..`, it may not catch all traversal attempts
- **Impact**: Could potentially create worktrees outside intended directories
- **Fix**: More robust path validation

## Documentation Issues

### 18. Undocumented Behavior
- **Issue**: Several features lack documentation (e.g., fd3 mechanism, CLAUDE files)
- **Impact**: Users don't know about features or use them incorrectly
- **Fix**: Comprehensive documentation

## Testing Gaps

### 19. Integration Tests
- **Issue**: No integration tests for full command workflows
- **Impact**: Regressions in command behavior may go unnoticed
- **Fix**: Add integration test suite

### 20. Edge Case Testing
- **Issue**: Tests mostly cover happy paths
- **Impact**: Edge cases may break in production
- **Fix**: Add negative test cases and edge case tests

## Additional Issues Found

### 21. Argument Parsing Inconsistency
- **Issue**: Each command implements its own argument parsing logic with different patterns
- **Impact**: Inconsistent behavior, harder to maintain, more code duplication
- **Examples**:
  - `executeNew` uses a while loop with index manipulation
  - `executeRemove` uses a simple for loop
  - `executeGo` uses a for loop with different flag checks
  - `executeList` uses another for loop variant
- **Fix**: Create a shared argument parser utility

### 22. Duplicate Code in Interactive Selection
- **Issue**: Both remove.zig and go.zig have nearly identical interactive selection logic
- **Impact**: Code duplication, harder to maintain
- **Fix**: Extract shared interactive selection functionality

### 23. Resource Leak in copyDir
- **Issue**: In fs.zig copyDir function, using page_allocator without cleanup in a loop
- **Impact**: Memory leak when copying large directories
- **Fix**: Use proper allocator with cleanup

### 24. Claude Process Not Detached Properly
- **Issue**: In new.zig, claude process is spawned but not properly detached
- **Impact**: May become zombie process or interfere with terminal
- **Fix**: Properly detach process or use system shell to launch

### 25. Missing Validation in executeRemove
- **Issue**: No validation for branch names with special characters in remove command
- **Impact**: Could fail to find worktrees with encoded branch names
- **Fix**: Use sanitization consistently

### 26. Inconsistent Error Return Patterns
- **Issue**: Some functions return error unions, others use catch blocks with process.exit
- **Impact**: Makes error handling unpredictable
- **Examples**:
  - main.zig uses process.exit in some paths
  - Commands sometimes return errors, sometimes exit
- **Fix**: Establish consistent error propagation

### 27. No Cleanup on Worktree Creation Failure
- **Issue**: If worktree creation fails after directory creation, no cleanup occurs
- **Impact**: Leaves orphaned directories
- **Fix**: Add cleanup on failure

### 28. Time Formatting Edge Cases
- **Issue**: formatDuration doesn't handle edge cases like 0 seconds or very large values
- **Impact**: May show confusing output like "0s ago" or overflow
- **Fix**: Add bounds checking and special cases

### 29. Path Display Inconsistency
- **Issue**: Some commands show absolute paths, others show relative paths
- **Impact**: Confusing user experience
- **Fix**: Standardize path display format

### 30. Missing --version Flag Validation
- **Issue**: Version string is hardcoded in main.zig
- **Impact**: Version may not match actual build
- **Fix**: Generate version from build system