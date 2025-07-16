#!/bin/bash
# Simple working test script

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

FAILED_TESTS=0
PASSED_TESTS=0

pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED_TESTS++))
    return 0
}

fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED_TESTS++))
    return 0
}

info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

# Build
info "Building git-wt..."
zig build

BIN="$(pwd)/zig-out/bin/git-wt"

# Test basic commands
info "Testing basic commands..."

$BIN --version >/dev/null 2>&1 && pass "Version command works" || fail "Version command failed"
$BIN --help >/dev/null 2>&1 && pass "Help command works" || fail "Help command failed"
$BIN new --help >/dev/null 2>&1 && pass "New help works" || fail "New help failed"

# Test validation
info "Testing validation..."
! $BIN --non-interactive new 'invalid branch' >/dev/null 2>&1 && pass "Rejects invalid branch" || fail "Should reject invalid branch"

# Test worktree creation
info "Testing worktree functionality..."
TEST_DIR="$(mktemp -d)"
cd "$TEST_DIR"

git init test-repo
cd test-repo
echo "# Test" > README.md
git add README.md
git commit -m "Initial commit"

REPO_ROOT="$(pwd)"
TREES_DIR="$(dirname "$REPO_ROOT")/test-repo-trees"

# Create worktree
$BIN --non-interactive new feature-branch && pass "Worktree creation succeeded" || fail "Worktree creation failed"

# Verify creation
[ -d "$TREES_DIR/feature-branch" ] && pass "Worktree directory exists" || fail "Worktree directory missing"
[ -f "$TREES_DIR/feature-branch/.git" ] && pass "Worktree has .git file" || fail "Worktree missing .git file"
git show-ref --verify --quiet refs/heads/feature-branch && pass "Branch was created" || fail "Branch not created"

# Test go command
cd "$REPO_ROOT"
GO_OUTPUT=$($BIN --non-interactive go feature-branch 2>&1)
echo "$GO_OUTPUT" | grep -q "cd $TREES_DIR/feature-branch" && pass "Go command works" || fail "Go command failed: $GO_OUTPUT"

# Test removal
cd "$TREES_DIR/feature-branch"
$BIN --non-interactive rm && pass "Removal succeeded" || fail "Removal failed"

# Verify removal
[ ! -d "$TREES_DIR/feature-branch" ] && pass "Worktree directory removed" || fail "Worktree directory still exists"
[ "$(pwd)" = "$REPO_ROOT" ] && pass "Returned to main repo" || fail "Not in main repo: $(pwd)"

# Clean up
cd /
rm -rf "$TEST_DIR"

# Summary
echo ""
echo "================ TEST SUMMARY ================"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"
echo "=============================================="

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi