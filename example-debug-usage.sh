#!/bin/bash
# Example of how to use the debug feature with the shell alias

echo "=== Correct usage of debug feature ==="
echo

echo "1. First, create the alias WITH the --debug flag:"
echo '   eval "$(git-wt alias gwt --debug)"'
echo

echo "2. Then use the alias normally - debug output will appear:"
echo "   gwt go                # Interactive mode with debug"
echo "   gwt go main           # Direct navigation with debug"
echo "   gwt new feature       # New worktree with debug"
echo

echo "The debug output from the shell alias shows:"
echo "   - What command is being run"
echo "   - The exit code"
echo "   - The captured cd command"
echo

echo "Note: 'gwt go --debug' passes --debug to the go command itself,"
echo "      not to the shell wrapper!"