#!/bin/bash

echo "=== Final test of gwt go --no-tty ==="

# Clean slate
git branch -D test-final 2>/dev/null || true

# Create worktree
./zig-out/bin/git-wt new test-final -n

# Generate fresh alias
eval "$(./zig-out/bin/git-wt alias gwt)"

# Test 1: Direct navigation with branch name
echo
echo "Test 1: gwt go test-final --no-tty"
BEFORE=$(pwd)
gwt go test-final --no-tty
AFTER=$(pwd)
echo "Before: $BEFORE"
echo "After:  $AFTER"
[ "$BEFORE" != "$AFTER" ] && echo "✅ SUCCESS" || echo "❌ FAILED"

# Go back
cd /Users/paul/projects-personal/git-wt

# Test 2: Interactive with piped input
echo
echo "Test 2: echo '1' | gwt go --no-tty"
BEFORE=$(pwd)
# Explicitly test what the function does
echo "1" | (
    git_wt_bin="./zig-out/bin/git-wt"
    flags=""
    cd_cmd=$(GWT_USE_FD3=1 eval "$git_wt_bin" go --no-tty $flags 3>&1 1>&2)
    exit_code=$?
    echo "Debug: exit_code=$exit_code, cd_cmd='$cd_cmd'"
    if [ $exit_code -eq 0 ] && [ -n "$cd_cmd" ] && echo "$cd_cmd" | grep -q '^cd '; then
        eval "$cd_cmd"
        pwd
    fi
)

# Cleanup
cd /Users/paul/projects-personal/git-wt
./zig-out/bin/git-wt rm test-final -n
git branch -D test-final 2>/dev/null || true