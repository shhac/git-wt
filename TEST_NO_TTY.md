# Testing --no-tty Flag

The `--no-tty` flag has been fixed to work properly with the shell alias. To test it manually:

## Setup

1. Build the project:
   ```bash
   zig build
   ```

2. Set up the shell alias in your current shell:
   ```bash
   eval "$(./zig-out/bin/git-wt alias gwt)"
   ```

3. Create a test worktree:
   ```bash
   gwt new test-no-tty
   ```

## Test the Fix

1. Go back to main:
   ```bash
   gwt go main
   ```

2. Test with `--no-tty` (interactive number selection):
   ```bash
   gwt go --no-tty
   # Enter "1" when prompted
   # You should now be in the test-no-tty worktree
   ```

3. Test direct branch navigation with `--no-tty`:
   ```bash
   gwt go main
   gwt go test-no-tty --no-tty
   # You should now be in the test-no-tty worktree
   ```

## What Was Fixed

Previously, `--no-tty` was being treated the same as `--non-interactive`, causing the command to exit without prompting. Now:

- `--non-interactive`: Lists worktrees and exits (no interaction)
- `--no-tty`: Shows number-based selection prompt (no arrow keys)

The fix also ensures the fd3 mechanism works correctly with `--no-tty`, allowing the shell alias to change directories properly.

## Cleanup

```bash
gwt rm test-no-tty
```