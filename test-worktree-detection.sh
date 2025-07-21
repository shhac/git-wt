#!/bin/bash
# End-to-end test for worktree detection
# This test ensures that is_current detection works correctly and doesn't regress

set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Testing git-wt worktree detection${NC}"
echo "========================================"

# Build the binary
echo -e "\n${YELLOW}Building git-wt...${NC}"
zig build || { echo -e "${RED}Build failed${NC}"; exit 1; }

# Get absolute paths
MAIN_DIR=$(pwd)
GWT="${MAIN_DIR}/zig-out/bin/git-wt"

# Clean up any previous test worktrees
echo -e "\n${YELLOW}Cleaning up any existing test worktrees...${NC}"
git worktree remove -f ../git-wt-trees/test-detection-1 2>/dev/null || true
git worktree remove -f ../git-wt-trees/test-detection-2 2>/dev/null || true
git branch -D test-detection-1 2>/dev/null || true
git branch -D test-detection-2 2>/dev/null || true

# Create test worktrees
echo -e "\n${YELLOW}Creating test worktrees...${NC}"
$GWT new test-detection-1 -n
$GWT new test-detection-2 -n

# Test 1: List from main should show main as current
echo -e "\n${YELLOW}Test 1: List from main repository${NC}"
cd "$MAIN_DIR"
OUTPUT=$($GWT list --plain 2>&1)
echo "$OUTPUT"

# Check that main is marked as current (only one line should have main in it)
MAIN_LINES=$(echo "$OUTPUT" | grep -c "main" || true)
if [ "$MAIN_LINES" -eq 1 ]; then
    echo -e "${GREEN}✓ Main repository correctly identified${NC}"
else
    echo -e "${RED}✗ Main repository detection failed${NC}"
    exit 1
fi

# Test 2: Go from main should NOT show main
echo -e "\n${YELLOW}Test 2: Go command from main repository${NC}"
GO_OUTPUT=$($GWT go -n --plain 2>&1)
echo "$GO_OUTPUT"

# Check that main is NOT in the go output
if echo "$GO_OUTPUT" | grep -q "main"; then
    echo -e "${RED}✗ Go command incorrectly showing main repository${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Go command correctly excludes current (main) repository${NC}"
fi

# Test 3: List from worktree should show that worktree as current
echo -e "\n${YELLOW}Test 3: List from worktree${NC}"
cd ../git-wt-trees/test-detection-1
OUTPUT=$($GWT list --plain 2>&1)
echo "$OUTPUT"

# Verify test-detection-1 appears exactly once
DETECTION1_COUNT=$(echo "$OUTPUT" | grep -c "test-detection-1" || true)
if [ "$DETECTION1_COUNT" -eq 1 ]; then
    echo -e "${GREEN}✓ Current worktree correctly identified${NC}"
else
    echo -e "${RED}✗ Current worktree detection failed (found $DETECTION1_COUNT times)${NC}"
    exit 1
fi

# Test 4: Go from worktree should show other worktrees but NOT current
echo -e "\n${YELLOW}Test 4: Go command from worktree${NC}"
GO_OUTPUT=$($GWT go -n 2>&1)
echo "$GO_OUTPUT"

# Check that current worktree is NOT shown
if echo "$GO_OUTPUT" | grep -q "test-detection-1"; then
    echo -e "${RED}✗ Go command incorrectly showing current worktree${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Go command correctly excludes current worktree${NC}"
fi

# Check that other worktrees ARE shown
if echo "$GO_OUTPUT" | grep -q "test-detection-2" && echo "$GO_OUTPUT" | grep -q "\[main\]"; then
    echo -e "${GREEN}✓ Go command shows other worktrees${NC}"
else
    echo -e "${RED}✗ Go command missing other worktrees${NC}"
    exit 1
fi

# Test 5: Navigate to subdirectory and verify detection still works
echo -e "\n${YELLOW}Test 5: Detection from subdirectory${NC}"
mkdir -p src/utils
cd src/utils
OUTPUT=$($GWT list --plain 2>&1)

# Should still detect test-detection-1 as current
DETECTION1_COUNT=$(echo "$OUTPUT" | grep -c "test-detection-1" || true)
if [ "$DETECTION1_COUNT" -eq 1 ]; then
    echo -e "${GREEN}✓ Current worktree correctly identified from subdirectory${NC}"
else
    echo -e "${RED}✗ Subdirectory detection failed${NC}"
    exit 1
fi

# Test 6: Verify false positive protection
echo -e "\n${YELLOW}Test 6: False positive protection${NC}"
cd "$MAIN_DIR"

# The main repo path should NOT match worktrees with similar prefixes
# This was the original bug - /path/to/repo would match /path/to/repo-trees/*
LIST_OUTPUT=$($GWT list --no-color 2>&1)

# Count how many times we see the current marker (*)
CURRENT_COUNT=$(echo "$LIST_OUTPUT" | grep -c "^\s*\*" || true)
if [ "$CURRENT_COUNT" -eq 1 ]; then
    echo -e "${GREEN}✓ Only one worktree marked as current${NC}"
else
    echo -e "${RED}✗ Multiple worktrees marked as current (found $CURRENT_COUNT)${NC}"
    echo "Full output:"
    echo "$LIST_OUTPUT"
    exit 1
fi

# Clean up
echo -e "\n${YELLOW}Cleaning up test worktrees...${NC}"
cd "$MAIN_DIR"
git worktree remove -f ../git-wt-trees/test-detection-1
git worktree remove -f ../git-wt-trees/test-detection-2

echo -e "\n${GREEN}All tests passed! ✓${NC}"
echo -e "${GREEN}Worktree detection is working correctly${NC}"