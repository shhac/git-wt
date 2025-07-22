#!/bin/bash

echo "=== Testing fd3 mechanism ==="

# Test 1: Direct fd3 test
echo
echo "Test 1: Direct fd3 output"
cd_cmd=$(GWT_USE_FD3=1 ./zig-out/bin/git-wt go main 3>&1 1>&2)
echo "Captured from fd3: '$cd_cmd'"

# Test 2: Test with existing worktree
echo
echo "Test 2: With test worktree"
./zig-out/bin/git-wt new test-fd3 -n 2>/dev/null || true
cd_cmd=$(echo "1" | GWT_USE_FD3=1 ./zig-out/bin/git-wt go --no-tty 3>&1 1>&2)
echo "Captured from fd3: '$cd_cmd'"

# Test 3: Debug environment
echo
echo "Test 3: Check environment in go command"
GWT_USE_FD3=1 ./zig-out/bin/git-wt go --debug 2>&1 | grep -A5 "Environment" | head -10

# Cleanup
./zig-out/bin/git-wt rm test-fd3 -n 2>/dev/null || true
git branch -D test-fd3 2>/dev/null || true