#!/bin/bash

echo "=== Checking User's Alias Setup ==="
echo

echo "1. Current alias definition:"
type gwt 2>/dev/null || echo "ERROR: gwt alias not found!"

echo
echo "2. Generate fresh alias to compare:"
echo
/Users/paul/projects-personal/git-wt/zig-out/bin/git-wt alias gwt

echo
echo "=== Key Check ==="
echo "The user's alias should contain this line:"
echo "    cd_cmd=\$(GWT_USE_FD3=1 eval \"\$git_wt_bin\" go --no-tty \$flags 3>&1 1>&2)"
echo
echo "If the user's alias is missing '--no-tty', that's the problem!"
echo
echo "=== Solution ==="
echo "Run this to update the alias:"
echo "eval \"\$(git-wt alias gwt)\""