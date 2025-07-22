#!/bin/bash
set -e

echo "=== Shell Integration Test Suite ==="
echo "Testing git-wt shell alias functionality"
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to run a test
run_test() {
    local test_name="$1"
    local expected="$2"
    local actual="$3"
    
    if [ "$expected" = "$actual" ]; then
        echo -e "${GREEN}✓${NC} $test_name"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC} $test_name"
        echo "  Expected: $expected"
        echo "  Actual: $actual"
        ((TESTS_FAILED++))
    fi
}

# Build the project
echo "Building git-wt..."
zig build -Doptimize=ReleaseFast >/dev/null 2>&1

# Clean up any existing test worktrees
git branch -D test-shell-1 test-shell-2 2>/dev/null || true
rm -rf ../git-wt-trees/test-shell-* 2>/dev/null || true

# Create test worktrees
echo "Setting up test environment..."
./zig-out/bin/git-wt new test-shell-1 -n >/dev/null 2>&1
./zig-out/bin/git-wt new test-shell-2 -n >/dev/null 2>&1

# Test 1: Direct fd3 output for branch navigation
echo
echo "Test Group 1: FD3 Mechanism"
cd_cmd=$(GWT_USE_FD3=1 ./zig-out/bin/git-wt go test-shell-1 3>&1 1>&2 2>/dev/null)
run_test "Direct branch navigation with fd3" "cd $(pwd)-trees/test-shell-1" "$cd_cmd"

# Test 2: FD3 output for main
cd_cmd=$(GWT_USE_FD3=1 ./zig-out/bin/git-wt go main 3>&1 1>&2 2>/dev/null)
expected_main="cd $(pwd)"
run_test "Main navigation with fd3" "$expected_main" "$cd_cmd"

# Test 3: Interactive selection with fd3
cd_cmd=$(echo "1" | GWT_USE_FD3=1 ./zig-out/bin/git-wt go 3>&1 1>&2 2>/dev/null | grep "^cd")
if [[ "$cd_cmd" =~ ^cd\ .*test-shell ]]; then
    run_test "Interactive selection with fd3" "pass" "pass"
else
    run_test "Interactive selection with fd3" "cd command" "$cd_cmd"
fi

# Test 4: --no-tty with fd3
cd_cmd=$(echo "1" | GWT_USE_FD3=1 ./zig-out/bin/git-wt go --no-tty 3>&1 1>&2 2>/dev/null | grep "^cd")
if [[ "$cd_cmd" =~ ^cd\ .*test-shell ]]; then
    run_test "--no-tty flag with fd3" "pass" "pass"
else
    run_test "--no-tty flag with fd3" "cd command" "$cd_cmd"
fi

# Test 5: Show command mode
echo
echo "Test Group 2: Show Command Mode"
output=$(./zig-out/bin/git-wt go --show-command test-shell-1 2>/dev/null)
expected_show="cd $(pwd)-trees/test-shell-1"
run_test "Show command outputs cd" "$expected_show" "$output"

# Test 6: Non-interactive mode (should not output cd commands)
echo
echo "Test Group 3: Non-Interactive Mode"
output=$(./zig-out/bin/git-wt go --non-interactive 2>&1 | grep -c "^cd" || true)
run_test "Non-interactive doesn't output cd" "0" "$output"

# Test 7: Alias function behavior
echo
echo "Test Group 4: Shell Alias Function"
eval "$(./zig-out/bin/git-wt alias gwt)"

# Capture what the function would do
cd_output=$(
    git_wt_bin="./zig-out/bin/git-wt"
    flags=""
    cd_cmd=$(GWT_USE_FD3=1 eval "$git_wt_bin" go test-shell-1 $flags 3>&1 1>&2 2>/dev/null)
    echo "$cd_cmd"
)
run_test "Alias function captures fd3 output" "cd $(pwd)-trees/test-shell-1" "$cd_output"

# Clean up
echo
echo "Cleaning up test worktrees..."
./zig-out/bin/git-wt rm test-shell-1 -n >/dev/null 2>&1
./zig-out/bin/git-wt rm test-shell-2 -n >/dev/null 2>&1
git branch -D test-shell-1 test-shell-2 2>/dev/null || true

# Summary
echo
echo "=== Test Summary ==="
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi