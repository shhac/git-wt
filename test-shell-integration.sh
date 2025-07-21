#!/bin/bash
# Shell integration tests that require the shell alias to be set up

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
}

fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED_TESTS++))
}

info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

cleanup() {
    # Change back to original directory
    cd "$ORIGINAL_DIR" 2>/dev/null || true
    
    # Clean up any test repositories
    if [[ -d ".e2e-test" ]]; then
        find .e2e-test -name "shell_integration_*" -type d -exec rm -rf {} + 2>/dev/null || true
    fi
}

# Set trap for cleanup
trap cleanup EXIT

# Save original directory
ORIGINAL_DIR="$(pwd)"

info "Building git-wt for shell integration tests..."
if ! zig build; then
    fail "Failed to build git-wt"
    exit 1
fi

# Get the binary path
GWT_BIN="$(pwd)/zig-out/bin/git-wt"

info "Setting up shell alias for testing..."
eval "$("$GWT_BIN" --alias gwt)"

info "Creating test repository..."
TEST_DIR=".e2e-test/shell_integration_$(date +%s)"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Initialize git repo
git init
git config user.email "test@example.com"
git config user.name "Test User"
echo "# Test Repo" > README.md
git add README.md
git commit -m "Initial commit"

info "Testing shell alias basic functionality..."

# Test 1: Shell alias should exist
if ! command -v gwt &> /dev/null; then
    fail "Shell alias 'gwt' was not created"
else
    pass "Shell alias 'gwt' exists"
fi

# Test 2: Create worktree with shell alias
info "Creating worktree with shell alias..."
if gwt new test-feature 2>/dev/null; then
    pass "Created worktree using shell alias"
    
    # Verify we're in the correct directory after creation
    CURRENT_DIR="$(basename "$(pwd)")"
    if [[ "$CURRENT_DIR" == "test-feature" ]]; then
        pass "Shell alias correctly changed directory to new worktree"
    else
        fail "Shell alias did not change directory (in: $CURRENT_DIR, expected: test-feature)"
    fi
    
    # Test 3: Navigate back to main using shell alias
    if gwt go main 2>/dev/null; then
        CURRENT_DIR="$(basename "$(pwd)")"
        if [[ "$CURRENT_DIR" != "test-feature" ]]; then
            pass "Shell alias navigation to main repository works"
        else
            fail "Shell alias did not navigate away from worktree"
        fi
    else
        fail "Shell alias navigation to main failed"
    fi
    
    # Test 4: Navigate back to worktree
    if gwt go test-feature 2>/dev/null; then
        CURRENT_DIR="$(basename "$(pwd)")"
        if [[ "$CURRENT_DIR" == "test-feature" ]]; then
            pass "Shell alias navigation to specific worktree works"
        else
            fail "Shell alias did not navigate to specified worktree"
        fi
    else
        fail "Shell alias navigation to worktree failed"
    fi
    
    # Test 5: Remove worktree (non-interactive)
    gwt go main 2>/dev/null # Go back to main first
    if echo "y" | gwt rm test-feature 2>/dev/null; then
        pass "Shell alias worktree removal works"
    else
        fail "Shell alias worktree removal failed"
    fi
    
else
    fail "Failed to create worktree using shell alias"
fi

# Test 6: Help commands should work
if gwt --help >/dev/null 2>&1; then
    pass "Shell alias help command works"
else
    fail "Shell alias help command failed"
fi

# Test 7: Version command should work
if gwt --version >/dev/null 2>&1; then
    pass "Shell alias version command works"
else
    fail "Shell alias version command failed"
fi

# Summary
echo
info "Test Summary:"
echo -e "  ${GREEN}Passed:${NC} $PASSED_TESTS"
echo -e "  ${RED}Failed:${NC} $FAILED_TESTS"

if [[ $FAILED_TESTS -eq 0 ]]; then
    echo -e "${GREEN}All shell integration tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi