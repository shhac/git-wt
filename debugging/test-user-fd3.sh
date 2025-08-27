#!/bin/bash

echo "=== Testing fd3 for the user's issue ==="
echo

# Test in the user's repository path structure
cd /Users/paul/projects/web

echo "Current directory: $(pwd)"
echo "Testing fd3 mechanism..."

echo
echo "Test 1: Direct fd3 check with debug"
GWT_USE_FD3=1 /Users/paul/projects-personal/git-wt/zig-out/bin/git-wt go --debug 2>&1 | grep -E "(GWT_USE_FD3|Environment)" -A 10

echo
echo "Test 2: Check if binary works with your worktrees"
echo "Available worktrees:"
git worktree list

echo
echo "Test 3: Direct fd3 output test"
fd3_output=$(echo "1" | GWT_USE_FD3=1 /Users/paul/projects-personal/git-wt/zig-out/bin/git-wt go --no-tty 3>&1 1>&2 2>/dev/null)
echo "FD3 output: '$fd3_output'"

echo
echo "Test 4: What does the alias capture?"
gwt_output=$(echo "1" | gwt go 2>&1)
echo "Full gwt output:"
echo "$gwt_output"

if [[ "$fd3_output" =~ ^cd ]]; then
    echo "✅ FD3 is working correctly"
else
    echo "❌ FD3 is not working - this is the issue"
fi