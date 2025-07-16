#!/bin/bash
# Simple non-interactive test script

set -euo pipefail

echo "Building git-wt..."
zig build

BIN="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/zig-out/bin/git-wt"

echo ""
echo "Testing help:"
$BIN --help

echo ""
echo "Testing version:"
$BIN --version

echo ""
echo "Testing in test directory..."
TEST_DIR="$(mktemp -d)"
cd "$TEST_DIR"

# Create test repo
git init test-repo
cd test-repo
echo "# Test" > README.md
git add README.md
git commit -m "Initial commit"

echo ""
echo "Creating worktree with --non-interactive:"
$BIN --non-interactive new feature-branch

echo ""
echo "Listing worktrees:"
git worktree list

echo ""
echo "Testing go command:"
$BIN --non-interactive go feature-branch

echo ""
echo "Navigating to worktree and removing:"
cd ../test-repo-trees/feature-branch
$BIN --non-interactive rm

echo ""
echo "Cleaning up..."
cd /
rm -rf "$TEST_DIR"

echo "Tests completed!"