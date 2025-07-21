#!/bin/bash
# Test concurrent worktree operations

set -e

echo "Testing concurrent worktree operations..."

# Build the tool
echo "Building git-wt..."
zig build

# Create a test repository
TEST_DIR=".e2e-test/concurrent-test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Initialize git repo
git init
echo "test" > README.md
git add .
git commit -m "initial"

# Path to our binary
GIT_WT="../../zig-out/bin/git-wt"

# Test 1: Try to create two worktrees simultaneously
echo -e "\nTest 1: Creating two worktrees simultaneously..."

# Start first worktree creation in background
$GIT_WT -n new feature-1 &
PID1=$!

# Give it a moment to acquire the lock
sleep 0.5

# Try to create second worktree (should fail with lock timeout)
echo "Trying to create second worktree while first is running..."
if $GIT_WT -n new feature-2 2>&1 | grep -q "Another git-wt operation is in progress"; then
    echo "✓ Correctly prevented concurrent operation"
else
    echo "✗ Failed to prevent concurrent operation"
    wait $PID1
    exit 1
fi

# Wait for first operation to complete
wait $PID1
echo "✓ First worktree created successfully"

# Now second operation should succeed
echo -e "\nTest 2: Creating second worktree after first completes..."
if $GIT_WT -n new feature-2; then
    echo "✓ Second worktree created successfully"
else
    echo "✗ Failed to create second worktree"
    exit 1
fi

# Test 3: Try concurrent remove operations
echo -e "\nTest 3: Testing concurrent remove operations..."

# Try to remove two worktrees simultaneously
$GIT_WT -n rm feature-1 &
PID1=$!

sleep 0.5

if $GIT_WT -n rm feature-2 2>&1 | grep -q "Another git-wt operation is in progress"; then
    echo "✓ Correctly prevented concurrent remove operation"
else
    echo "✗ Failed to prevent concurrent remove operation"
    wait $PID1
    exit 1
fi

wait $PID1
echo "✓ First worktree removed successfully"

# Clean up
cd ../..
rm -rf "$TEST_DIR"

echo -e "\n✅ All concurrent operation tests passed!"