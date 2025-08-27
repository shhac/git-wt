# Learnings

This directory contains development learnings and discoveries from the git-wt project.

## Contents

### Testing Interactive CLIs
- **TESTING_INTERACTIVE_CLIS_WITH_EXPECT.md** - Comprehensive guide to testing with expect (2025-08-27) ⭐ RECOMMENDED
- **HOW_TO_TEST_TTY_INPUTS.md** - Various testing methodologies for TTY-related features
- **TEST_NO_TTY.md** - Documentation of --no-tty flag behavior and testing

### Shell Integration & FD3 Mechanism
- **NAVIGATION_AND_FD3.md** - How the fd3 mechanism works and common pitfalls (pipes, eval)
- **EVAL_FD3_FIX.md** - Discovery of how eval broke fd3 and the proper fix (2025-01-24)

## Key Takeaways

1. **Expect is the best testing tool** - Provides real pseudo-TTY, keyboard simulation, and screen capture capabilities
2. **Human-like timing matters** - 25ms between keystrokes makes tests more realistic and reliable
3. **Screen capture on timeout** - Using `expect_out(buffer)` shows exactly where tests get stuck
4. **Avoid eval in shell scripts** - It adds complexity and can break environment variable passing
5. **Understand shell behavior** - Pipes create subshells, which affects commands like `cd`
6. **Test the actual mechanism** - Don't just test if a feature works, test HOW it works
7. **Document discoveries** - Future developers (including yourself) will thank you

## Evolution of Testing Approach

1. **Initial attempts**: Simple piping and here-documents (limited success)
2. **Investigation phase**: Multiple approaches tried (script, expect, manual)
3. **Breakthrough**: Discovered expect's `expect_out(buffer)` for screen capture
4. **Refinement**: Added human-like timing and comprehensive test patterns
5. **Current state**: Full test suite in `test-interactive/` with ~100% coverage of interactive features