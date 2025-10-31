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
- ✅ Bug #27: fd3 mechanism broken by eval in shell alias (removed unnecessary eval usage)
- ✅ Bug #28: Time Formatting Edge Cases (handles "just now" and decades)
- ✅ Bug #29: Path Display Inconsistency (standardized to display names for consistency)
- ✅ Bug #30: Missing --version Flag Validation (version generated from build system)
- ✅ Bug #18: Undocumented Behavior (comprehensive documentation added for all features including fd3, CLAUDE files, locking, validation, flags, and troubleshooting)
- ✅ Bug #31: Missing Test Coverage for Commands (added comprehensive test coverage for all command files)
- ✅ Bug #32: Memory Leak in git.zig Error Handling (error storage is now cleared when retrieved)
- ✅ Bug #33: Race Condition in Interactive Signal Handling (using global termios storage instead of pointer)
- ✅ Bug #36: Incorrect Current Worktree Detection (added proper path separator check)
- ✅ Bug #37: Input Buffer Overflow Risk (increased buffer size and added overflow handling)
- ✅ Bug #38: Insufficient Path Validation (added comprehensive Unicode validation)
- ✅ Bug #39: Missing Null Checks (improved bounds checking and safer optional handling)
- ✅ Bug #40: Windows Lock File Handling (added WSL2 support with fallback for native Windows)
- ✅ Bug #34: Command Injection in Claude Process Spawning (replaced shell execution with direct process spawning)
- ✅ Bug #35: TOCTOU Race in Lock File Implementation (implemented atomic rename operation)
- ✅ Bug #13: Large Repository Handling (implemented smart threshold-based loading and early-exit branch search)


## Critical Issues

(None currently identified)

## Edge Cases

(None currently identified)

## Usability Issues

(None currently identified)

## Code Quality Issues

(None currently identified)

## Performance Issues

(None currently identified)

## Platform-Specific Issues

(None currently identified)

## Documentation Issues

(None currently identified)

## Testing Gaps

(None currently identified)

## Security Issues

(None currently identified)

