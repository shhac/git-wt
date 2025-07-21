# Known Bugs and Edge Cases

This file tracks known bugs, edge cases, and potential issues in the git-wt codebase.

## Fixed Issues
The following issues have been resolved (sorted numerically):
- ✅ Bug #1: Memory management in listWorktreesWithTime (using arena allocator)
- ✅ Bug #2: Signal handler race condition (using atomic operations)
- ✅ Bug #3: Path Handling (already handled with sanitization and validation)
- ✅ Bug #4: Concurrent Worktree Operations (file-based locking implemented)
- ✅ Bug #5: Repository State Validation (comprehensive state checks added)
- ✅ Bug #6: Better Error Messages (enhanced with git output and contextual tips)
- ✅ Bug #7: Interactive Mode Edge Cases (SIGWINCH handling added)
- ✅ Bug #8: Shell Integration - fd3 mechanism documentation (comprehensive docs added)
- ✅ Bug #9: Command injection vulnerability (enhanced validation)
- ✅ Bug #10: Missing Input Validation (already comprehensive)
- ✅ Bug #11: Resource Cleanup (improved error handling and cleanup)
- ✅ Bug #12: Redundant Git Calls (optimized with git dir caching)
- ✅ Bug #14: Windows Compatibility (WSL2 support - no changes needed)
- ✅ Bug #15: Case-Insensitive Filesystems (conflict detection on macOS/Windows)
- ✅ Bug #16: Argument parsing inconsistency (shared args.zig parser)
- ✅ Bug #17: Resource leak in copyDir (proper allocator usage)
- ✅ Bug #19: Integration Tests (comprehensive inter-module testing with build system integration)
- ✅ Bug #20: Edge Case Testing (extensive boundary value and error condition tests)
- ✅ Bug #21: Path traversal vulnerability (robust validation)
- ✅ Bug #22: Duplicate Code in Interactive Selection (shared utility functions)
- ✅ Bug #23: No cleanup on worktree creation failure (errdefer cleanup)
- ✅ Bug #24: Claude Process Not Detached Properly (fixed with shell exec)
- ✅ Bug #25: Missing Validation in executeRemove (added branch name validation and sanitization handling)
- ✅ Bug #26: Inconsistent Error Return Patterns (unified error handling in main)
- ✅ Bug #28: Time Formatting Edge Cases (handles "just now" and decades)
- ✅ Bug #29: Path Display Inconsistency (standardized to display names for consistency)
- ✅ Bug #30: Missing --version Flag Validation (version generated from build system)


## Edge Cases



## Usability Issues


## Code Quality Issues


## Performance Issues

### 13. Large Repository Handling
- **Issue**: Loading all worktrees into memory at once
- **Impact**: High memory usage with many worktrees
- **Fix**: Implement pagination or streaming

## Platform-Specific Issues

## Documentation Issues

### 18. Undocumented Behavior
- **Issue**: Several features lack documentation (e.g., fd3 mechanism, CLAUDE files)
- **Impact**: Users don't know about features or use them incorrectly
- **Fix**: Comprehensive documentation

## Testing Gaps

## Additional Issues Found


