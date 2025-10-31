# Debugging Scripts

This directory contains debugging and testing scripts used during development. These are not part of the main application but are kept for reference and troubleshooting.

## Scripts

### Shell Integration Debugging
- `check-user-alias.sh` - Verifies shell alias configuration
- `debug-interactive-fd3.sh` - Debug file descriptor 3 mechanism for shell integration
- `test-user-fd3.sh` - Test fd3 functionality
- `example-debug-usage.sh` - Example usage patterns for debugging

### Testing Scripts
- `test-interactive.sh` - Manual testing of interactive features
- `test-claude-script.sh` - Claude's test script for complex bash commands

### Rendering Issue Tests
Scripts to reproduce and verify UI rendering bugs:
- `test-all-rendering.sh` - **Master script** to run all rendering tests
- `test-rendering-exit-cleanup.sh` - Test for ghost menu items after selection
- `test-rendering-color-bleed.sh` - Test for ANSI color bleeding and attribute persistence
- `test-rendering-flicker.sh` - Test for output buffering and flicker issues

**Quick start**: Run `./debugging/test-all-rendering.sh` for interactive testing suite.

For detailed analysis and fix plans, see:
- `.ai-cache/plan-fix-ui-rendering.md` - Implementation plan for all rendering fixes
- `.ai-cache/deep-dive-rendering-investigation.md` - Technical deep dive on rendering issues

## Note

These scripts are primarily for development and debugging purposes. For actual testing, use:
- Unit tests: `zig test src/main.zig`
- Interactive tests: `test-interactive/run-all-tests.sh`