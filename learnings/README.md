# Learnings

This directory contains development learnings and discoveries from the git-wt project.

## Contents

- **NAVIGATION_AND_FD3.md** - How the fd3 mechanism works and common pitfalls (pipes, eval)
- **EVAL_FD3_FIX.md** - Discovery of how eval broke fd3 and the proper fix (2025-01-24)
- **TEST_NO_TTY.md** - Documentation of --no-tty flag behavior and testing
- **HOW_TO_TEST_TTY_INPUTS.md** - Testing methodology for TTY-related features

## Key Takeaways

1. **Avoid eval in shell scripts** - It adds complexity and can break environment variable passing
2. **Understand shell behavior** - Pipes create subshells, which affects commands like `cd`
3. **Test the actual mechanism** - Don't just test if a feature works, test HOW it works
4. **Document discoveries** - Future developers (including yourself) will thank you