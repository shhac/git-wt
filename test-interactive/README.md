# Interactive Tests for git-wt

This directory contains expect-based interactive tests for the git-wt CLI tool. These tests verify the arrow-key navigation, multi-select functionality, and other interactive features.

## Prerequisites

- `expect` must be installed:
  - macOS: `brew install expect`
  - Linux: `apt-get install expect` or `yum install expect`

## Running Tests

### Run all tests
```bash
./run-all-tests.sh
```

### Run individual test suites
```bash
./test-navigation.exp  # Test arrow-key navigation in `go` command
./test-removal.exp     # Test multi-select removal in `rm` command
```

## Test Coverage

### Navigation Tests (`test-navigation.exp`)
- ✅ Arrow-key navigation (up/down arrows)
- ✅ Selection with Enter key
- ✅ Cancellation with ESC key
- ✅ Navigation from within a worktree
- ✅ --show-command flag behavior

### Removal Tests (`test-removal.exp`)
- ✅ Multi-select with arrow keys and space bar
- ✅ Selective removal (keeping some worktrees)
- ✅ Confirmation prompts
- ✅ ESC cancellation in multi-select mode
- ✅ --force flag skipping uncommitted changes check

## Test Design

The tests use expect scripts with human-like timing:
- 25ms delay between keystrokes
- 200-500ms reaction time before actions
- Screen capture capability for debugging

### Key Features
- **Human-like timing**: Tests simulate real user interaction speeds
- **Screen capture**: Can capture terminal state at any point for debugging
- **Fallback handling**: Tests handle both arrow-key and number-based modes
- **Cleanup**: All tests clean up their temporary repositories

## Adding New Tests

1. Create a new `.exp` file following the existing patterns
2. Use the helper procedures:
   - `capture_screen` - Capture and display terminal state
   - `send_key` - Send a single key with human delay
   - `send_human` - Type text with human-like speed
   - `test_failed` - Mark a test as failed

3. Add the test to `run-all-tests.sh`

## Debugging Failed Tests

Run individual test files directly to see detailed output:
```bash
./test-navigation.exp  # Shows all screen captures and step-by-step progress
```

The screen capture functionality shows exactly what the terminal displayed at each step, making it easy to debug UI issues.