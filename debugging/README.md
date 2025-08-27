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

## Note

These scripts are primarily for development and debugging purposes. For actual testing, use:
- Unit tests: `zig test src/main.zig`
- Interactive tests: `test-interactive/run-all-tests.sh`