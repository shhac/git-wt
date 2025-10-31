#!/usr/bin/env bash
# Test script for ANSI code nesting and color bleeding
# This script helps reproduce color bleeding between menu items

set -e

echo "=== Color Bleeding Rendering Test ==="
echo ""
echo "This test will help identify if colors persist beyond their intended scope."
echo ""
echo "Expected: Only current item has bold text and highlighted brackets"
echo "Bug: Previous items retain bold or colored brackets after navigation"
echo ""

cd "$(dirname "$0")/.."

# Build the binary
echo "Building git-wt..."
zig build 2>&1 | head -5

echo ""
echo "Test 1: Rapid navigation through menu items"
echo "--------------------------------------------"
echo "Action: Open menu, rapidly press Down arrow 10+ times"
echo "Watch for: Bold text or green brackets persisting on wrong items"
echo ""
read -p "Press Enter to run test 1..."

./zig-out/bin/git-wt go

echo ""
echo "Test 2: Navigate to bottom, then back to top"
echo "---------------------------------------------"
echo "Action: Open menu, press Down to bottom, then Up to top"
echo "Watch for: Formatting artifacts on items you passed"
echo ""
read -p "Press Enter to run test 2..."

./zig-out/bin/git-wt go

echo ""
echo "Test 3: Multi-select with space bar"
echo "------------------------------------"
echo "Action: git-wt rm (multi-select), toggle items with Space"
echo "Watch for: Selection indicators appearing on wrong items"
echo ""
echo "Note: This will open remove dialog - press ESC to cancel"
echo ""
read -p "Press Enter to run test 3..."

./zig-out/bin/git-wt rm

echo ""
echo "=== Test Complete ==="
echo ""
echo "Signs of color bleeding bug:"
echo "- Bold text on multiple items simultaneously"
echo "- Green brackets on non-current items"
echo "- Asterisks (*) appearing/disappearing incorrectly"
echo "- Dim attribute persisting to next items"
echo ""
echo "Next steps:"
echo "1. Review .ai-cache/plan-fix-ui-rendering.md"
echo "2. Implement Fix #3: Simplify ANSI nesting"
echo "3. Implement Fix #6: Replace inline ANSI codes"
echo "4. Re-run this test to verify fix"
