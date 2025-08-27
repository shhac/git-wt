#!/bin/bash

# Run all interactive tests for git-wt
# This script runs all expect-based interactive tests

set -e  # Exit on first error

echo "================================"
echo "Running git-wt Interactive Tests"
echo "================================"
echo ""

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Make sure we have a built binary
echo "Building git-wt..."
cd "$PROJECT_ROOT"
zig build
cd "$SCRIPT_DIR"

echo ""
echo "Running tests..."
echo ""

# Track test results
FAILED_TESTS=()
PASSED_TESTS=()

# Run navigation tests
echo ">>> Running navigation tests..."
if ./test-navigation.exp > /dev/null 2>&1; then
    PASSED_TESTS+=("Navigation")
    echo "✅ Navigation tests passed"
else
    FAILED_TESTS+=("Navigation")
    echo "❌ Navigation tests failed"
fi

echo ""

# Run removal tests
echo ">>> Running removal tests..."
if ./test-removal.exp > /dev/null 2>&1; then
    PASSED_TESTS+=("Removal")
    echo "✅ Removal tests passed"
else
    FAILED_TESTS+=("Removal")
    echo "❌ Removal tests failed"
fi

echo ""

# Run prunable worktree tests
echo ">>> Running prunable worktree tests..."
if ./test-prunable.exp > /dev/null 2>&1; then
    PASSED_TESTS+=("Prunable")
    echo "✅ Prunable worktree tests passed"
else
    FAILED_TESTS+=("Prunable")
    echo "❌ Prunable worktree tests failed"
fi

echo ""
echo "================================"
echo "Test Summary"
echo "================================"
echo ""

if [ ${#PASSED_TESTS[@]} -gt 0 ]; then
    echo "✅ Passed (${#PASSED_TESTS[@]}):"
    for test in "${PASSED_TESTS[@]}"; do
        echo "   - $test"
    done
fi

if [ ${#FAILED_TESTS[@]} -gt 0 ]; then
    echo ""
    echo "❌ Failed (${#FAILED_TESTS[@]}):"
    for test in "${FAILED_TESTS[@]}"; do
        echo "   - $test"
    done
fi

echo ""

# Exit with appropriate code
if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
    echo "🎉 All interactive tests passed!"
    exit 0
else
    echo "💥 Some tests failed. Run individual test files for details."
    exit 1
fi