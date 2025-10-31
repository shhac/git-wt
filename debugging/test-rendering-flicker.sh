#!/usr/bin/env bash
# Test script for output buffering and flicker issues
# This script helps reproduce flickering and progressive rendering

set -e

echo "=== Output Buffering / Flicker Test ==="
echo ""
echo "This test will help identify if menu redraws cause flicker or delays."
echo ""
echo "Expected: Smooth, instant menu updates"
echo "Bug: Menu items appear line-by-line, cursor visible during redraw, flicker"
echo ""

cd "$(dirname "$0")/.."

# Build the binary
echo "Building git-wt..."
zig build 2>&1 | head -5

echo ""
echo "Test 1: Rapid arrow key navigation"
echo "-----------------------------------"
echo "Action: Open menu, rapidly alternate Up/Down arrows"
echo "Watch for: Flicker, progressive line-by-line rendering"
echo ""
read -p "Press Enter to run test 1..."

./zig-out/bin/git-wt go

echo ""
echo "Test 2: Window resize during interaction"
echo "-----------------------------------------"
echo "Action: Open menu, resize terminal window"
echo "Watch for: Full screen clear, loss of context above menu"
echo ""
read -p "Press Enter to run test 2 (resize window while menu is open)..."

./zig-out/bin/git-wt go

echo ""
echo "Test 3: Slow connection simulation (optional)"
echo "----------------------------------------------"
echo "This test requires 'trickle' tool to simulate slow connection."
echo ""

if command -v trickle &> /dev/null; then
    echo "Trickle found! Testing with simulated 10KB/s connection..."
    echo "Action: Open menu, navigate with arrows"
    echo "Watch for: Visible delays, progressive rendering"
    echo ""
    read -p "Press Enter to run slow connection test..."

    trickle -s -d 10 -u 10 ./zig-out/bin/git-wt go
else
    echo "Trickle not installed. Skipping slow connection test."
    echo "To install: brew install trickle (macOS) or apt install trickle (Linux)"
fi

echo ""
echo "Test 4: Monitor actual ANSI codes sent"
echo "---------------------------------------"
echo "This will show the raw escape sequences sent to terminal."
echo "Watch for: Commands sent in multiple batches instead of atomically"
echo ""
read -p "Press Enter to monitor ANSI codes (use Ctrl+C after selecting)..."

echo ""
echo "Opening git-wt with output capture..."
script -q /dev/null ./zig-out/bin/git-wt go | cat -v | head -100

echo ""
echo "=== Test Complete ==="
echo ""
echo "Signs of buffering/flicker bug:"
echo "- Menu items appear progressively instead of all at once"
echo "- Visible cursor during redraw"
echo "- Flash or flicker when navigating"
echo "- Window resize clears entire screen"
echo "- ANSI codes appear in multiple batches"
echo ""
echo "Next steps:"
echo "1. Review .ai-cache/plan-fix-ui-rendering.md"
echo "2. Implement Fix #1: Add output flushing"
echo "3. Implement Fix #9: Improve window resize handling"
echo "4. Re-run this test to verify fix"
