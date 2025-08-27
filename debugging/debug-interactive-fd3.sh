#!/bin/bash

echo "=== Debug Interactive FD3 Issue ==="
echo

# Test the exact scenario the user is experiencing
echo "Testing interactive mode fd3 detection..."
echo

echo "1. Test fd3 directly (should work):"
echo "   GWT_USE_FD3=1 git-wt go --no-tty"
echo "   Result: (using --no-tty to get number selection)"
cd /Users/paul/projects/web
echo "1" | GWT_USE_FD3=1 /Users/paul/projects-personal/git-wt/zig-out/bin/git-wt go --no-tty 3>&1 1>&2

echo
echo "2. Test what happens in interactive mode:"
echo "   This simulates what happens when you use arrow keys..."
echo

# Create a simulated interactive test
echo "   Testing fd3.isEnabled() in interactive context:"
echo "   GWT_USE_FD3=1 git-wt go --debug (with arrow key mode)"

# This should show us if fd3 is detected in interactive mode
GWT_USE_FD3=1 /Users/paul/projects-personal/git-wt/zig-out/bin/git-wt go --debug 2>&1 | head -10

echo
echo "3. The issue might be that interactive mode runs in a different context"
echo "   where fd3 detection fails even though GWT_USE_FD3=1 is set."
echo
echo "4. Let's check if the issue is with the tty detection logic:"
echo "   Interactive mode only activates when both stdin and stdout are TTY"

echo
echo "Current TTY status:"
if [ -t 0 ]; then echo "  stdin is TTY"; else echo "  stdin is NOT TTY"; fi
if [ -t 1 ]; then echo "  stdout is TTY"; else echo "  stdout is NOT TTY"; fi
if [ -t 2 ]; then echo "  stderr is TTY"; else echo "  stderr is NOT TTY"; fi
if [ -t 3 ]; then echo "  fd3 is TTY"; else echo "  fd3 is NOT TTY (this is normal)"; fi

echo
echo "When you run 'gwt go', the shell alias should set up fd3 properly."
echo "The issue might be that interactive mode doesn't inherit the fd3 setup."