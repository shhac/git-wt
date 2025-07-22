#!/bin/bash

echo "=== Debugging shell alias with --no-tty ==="

# Setup
./zig-out/bin/git-wt new test-alias -n 2>/dev/null || true
eval "$(./zig-out/bin/git-wt alias gwt)"

# Test the raw command with fd3
echo
echo "1. Raw command test:"
cd_cmd=$(echo "1" | GWT_USE_FD3=1 ./zig-out/bin/git-wt go --no-tty 3>&1 1>&2)
echo "   Exit code: $?"
echo "   Captured: '$cd_cmd'"

# Test through the alias function
echo
echo "2. Through alias function:"
# Create a modified version of gwt that shows what it's doing
gwt_debug() {
    local git_wt_bin="./zig-out/bin/git-wt"
    local flags=""
    
    if [ "$1" = "go" ]; then
        shift
        echo "[DEBUG] Running: GWT_USE_FD3=1 $git_wt_bin go $@ $flags"
        local cd_cmd=$(echo "1" | GWT_USE_FD3=1 eval "$git_wt_bin" go "$@" $flags 3>&1 1>&2)
        echo "[DEBUG] Exit code: $?"
        echo "[DEBUG] Captured: '$cd_cmd'"
        echo "[DEBUG] Grep test: $(echo "$cd_cmd" | grep -q '^cd ' && echo "MATCHES" || echo "NO MATCH")"
        if [ $? -eq 0 ] && [ -n "$cd_cmd" ] && echo "$cd_cmd" | grep -q '^cd '; then
            echo "[DEBUG] Would run: eval '$cd_cmd'"
            eval "$cd_cmd"
        else
            echo "[DEBUG] Not running cd command"
        fi
    fi
}

gwt_debug go --no-tty

# Cleanup
./zig-out/bin/git-wt rm test-alias -n 2>/dev/null || true
git branch -D test-alias 2>/dev/null || true