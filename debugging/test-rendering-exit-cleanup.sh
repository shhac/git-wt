#!/usr/bin/env bash
# Test script for exit cleanup rendering issues
# This script helps reproduce ghost menu items after selection

set -e

echo "=== Exit Cleanup Rendering Test ==="
echo ""
echo "This test will help identify if ghost menu items remain after selection."
echo "You should observe the terminal AFTER selecting an item."
echo ""
echo "Expected: Clean terminal, no ghost lines"
echo "Bug: Remnants of menu items remain visible"
echo ""

cd "$(dirname "$0")/.."

# Build the binary
echo "Building git-wt..."
zig build 2>&1 | head -5

echo ""
echo "Test 1: Interactive selection with Enter"
echo "----------------------------------------"
echo "Action: Open menu, navigate with arrows, press Enter"
echo "Watch for: Ghost menu items after exit"
echo ""
read -p "Press Enter to run test 1..."

# Run interactive go command
./zig-out/bin/git-wt go

echo ""
echo "Did you see ghost menu items? (They would appear as faint text above)"
echo ""

echo ""
echo "Test 2: Interactive selection with ESC"
echo "---------------------------------------"
echo "Action: Open menu, navigate with arrows, press ESC"
echo "Watch for: Ghost menu items after cancel"
echo ""
read -p "Press Enter to run test 2..."

./zig-out/bin/git-wt go

echo ""
echo "Test 3: Interactive selection with 'q'"
echo "---------------------------------------"
echo "Action: Open menu, navigate with arrows, press 'q'"
echo "Watch for: Ghost menu items after quit"
echo ""
read -p "Press Enter to run test 3..."

./zig-out/bin/git-wt go

echo ""
echo "=== Test Complete ==="
echo ""
echo "If you observed ghost menu items in any test, the exit cleanup bug is present."
echo ""
echo "Next steps:"
echo "1. Review .ai-cache/plan-fix-ui-rendering.md"
echo "2. Implement Fix #2: Use \\x1b[0J for cleanup"
echo "3. Re-run this test to verify fix"
