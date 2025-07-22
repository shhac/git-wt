# Interactive Mode Navigation Fix

## Issue Summary

Users reported that `gwt go` in interactive mode was not changing directories as expected. After investigation, we found that arrow-key interactive mode was not properly communicating with the shell alias.

## Root Cause

The issue occurs when:
1. TTY is detected, triggering arrow-key interactive mode
2. Even though fd3 output is generated, the shell's handling of the interactive terminal interferes with capturing the fd3 output
3. The `process.changeCurDir` fallback only affects the git-wt process, not the parent shell

## Solution

Modified the shell alias to force `--no-tty` flag when running `gwt go` without arguments (interactive mode). This ensures:
- Number-based selection is used instead of arrow keys
- fd3 output is reliably captured by the shell
- Directory changes work correctly

## What Changed

In `src/commands/alias.zig`, the generated shell function now includes:

```bash
if [ $# -eq 0 ]; then
    # Interactive mode - force number selection for reliable fd3
    cd_cmd=$(GWT_USE_FD3=1 eval "$git_wt_bin" go --no-tty $flags 3>&1 1>&2)
else
    # Direct branch navigation
    cd_cmd=$(GWT_USE_FD3=1 eval "$git_wt_bin" go "$@" $flags 3>&1 1>&2)
fi
```

## User Impact

- `gwt go` now shows number-based selection instead of arrow keys
- Navigation works reliably in all cases
- Direct branch navigation (`gwt go feature-branch`) is unaffected

## Testing

The fix was verified with:
1. Direct navigation: `gwt go feature-branch` ✅
2. Interactive navigation: `gwt go` then type `1` ✅
3. Piped input: `echo "1" | gwt go` ❌ (expected - pipes create subshells)

## Alternative Approaches Considered

1. **Fixing arrow-key mode**: The issue appears to be fundamental to how shells handle interactive terminal I/O alongside fd3 redirection
2. **Using environment variables**: Would require shell to re-read environment after command completes
3. **Output to stdout**: Would interfere with the interactive UI

The chosen solution (forcing number-based selection) provides the best balance of reliability and user experience.