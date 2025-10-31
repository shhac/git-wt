#!/usr/bin/env bash
# Master test script for all rendering issues
# Run this to systematically test all identified rendering problems

set -e

cd "$(dirname "$0")/.."

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║           Git-wt Rendering Issues Test Suite                  ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""
echo "This suite tests the 3 critical rendering issues identified:"
echo ""
echo "  1. Exit Cleanup    - Ghost menu items after selection"
echo "  2. Color Bleeding  - Bold/color persisting beyond scope"
echo "  3. Output Flicker  - Progressive rendering and delays"
echo ""
echo "For detailed analysis, see:"
echo "  - .ai-cache/plan-fix-ui-rendering.md"
echo "  - .ai-cache/deep-dive-rendering-investigation.md"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo ""

PS3="Select test to run (or 0 to exit): "
options=(
    "Exit Cleanup Test"
    "Color Bleeding Test"
    "Output Flicker Test"
    "Run All Tests"
    "View Fix Plan"
    "Exit"
)

while true; do
    echo ""
    select opt in "${options[@]}"; do
        case $REPLY in
            1)
                echo ""
                ./debugging/test-rendering-exit-cleanup.sh
                break
                ;;
            2)
                echo ""
                ./debugging/test-rendering-color-bleed.sh
                break
                ;;
            3)
                echo ""
                ./debugging/test-rendering-flicker.sh
                break
                ;;
            4)
                echo ""
                echo "Running all tests sequentially..."
                echo ""
                ./debugging/test-rendering-exit-cleanup.sh
                echo ""
                echo "═══════════════════════════════════════════════════════════════"
                echo ""
                ./debugging/test-rendering-color-bleed.sh
                echo ""
                echo "═══════════════════════════════════════════════════════════════"
                echo ""
                ./debugging/test-rendering-flicker.sh
                echo ""
                echo "═══════════════════════════════════════════════════════════════"
                echo ""
                echo "All tests complete!"
                break
                ;;
            5)
                echo ""
                if [ -f ".ai-cache/plan-fix-ui-rendering.md" ]; then
                    cat .ai-cache/plan-fix-ui-rendering.md | less
                else
                    echo "Fix plan not found at .ai-cache/plan-fix-ui-rendering.md"
                fi
                break
                ;;
            6)
                echo ""
                echo "Exiting test suite."
                exit 0
                ;;
            *)
                echo "Invalid option. Please try again."
                break
                ;;
        esac
    done
done
