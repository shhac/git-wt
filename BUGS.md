# Known Bugs and Edge Cases

This file tracks known bugs, edge cases, and potential issues in the git-wt codebase.

## Fixed Issues
The following issues have been resolved:
- ✅ Memory management in listWorktreesWithTime (using arena allocator)
- ✅ Signal handler race condition (using atomic operations)
- ✅ Command injection vulnerability (enhanced validation)
- ✅ Path traversal vulnerability (robust validation)
- ✅ Argument parsing inconsistency (shared args.zig parser)
- ✅ Resource leak in copyDir (proper allocator usage)
- ✅ No cleanup on worktree creation failure (errdefer cleanup)
- ✅ Bug #5: Repository State Validation (comprehensive state checks added)
- ✅ Bug #6: Better Error Messages (enhanced with git output and contextual tips)
- ✅ Bug #28: Time Formatting Edge Cases (handles "just now" and decades)
- ✅ Bug #4: Concurrent Worktree Operations (file-based locking implemented)
- ✅ Bug #14: Windows Compatibility (WSL2 support - no changes needed)
- ✅ Bug #3: Path Handling (already handled with sanitization and validation)
- ✅ Bug #7: Interactive Mode Edge Cases (SIGWINCH handling added)
- ✅ Bug #10: Missing Input Validation (already comprehensive)
- ✅ Bug #24: Claude Process Not Detached Properly (fixed with shell exec)
- ✅ Bug #11: Resource Cleanup (improved error handling and cleanup)

## Edge Cases



## Usability Issues


### 8. Shell Integration
- **Issue**: The fd3 mechanism for shell integration is fragile and undocumented
- **Impact**: Users may not understand why commands behave differently
- **Fix**: Better documentation and error handling

## Code Quality Issues

### 9. Inconsistent Error Handling
- **Issue**: Mix of try/catch patterns and error returns without clear strategy
- **Impact**: Makes code harder to maintain and reason about
- **Fix**: Establish consistent error handling patterns


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


### 15. Case-Insensitive Filesystems
- **Issue**: No handling of case-insensitive filesystem issues
- **Impact**: Could create conflicting worktrees on macOS/Windows
- **Fix**: Add filesystem capability detection

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

### 22. Duplicate Code in Interactive Selection
- **Issue**: Both remove.zig and go.zig have nearly identical interactive selection logic
- **Impact**: Code duplication, harder to maintain
- **Fix**: Extract shared interactive selection functionality


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

### 29. Path Display Inconsistency
- **Issue**: Some commands show absolute paths, others show relative paths
- **Impact**: Confusing user experience
- **Fix**: Standardize path display format

### 30. Missing --version Flag Validation
- **Issue**: Version string is hardcoded in main.zig
- **Impact**: Version may not match actual build
- **Fix**: Generate version from build system