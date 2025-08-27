#!/bin/bash
set -e

echo "Building git-wt..."
zig build

echo -e "\n=== Testing 'git-wt new' without arguments (should show error) ==="
./zig-out/bin/git-wt new || true

echo -e "\n=== Testing through alias function ==="
# Set up the alias function
eval "$(./zig-out/bin/git-wt --alias gwt)"

echo -e "\n=== Testing 'gwt new' without arguments (should show error) ==="
gwt new || true

echo -e "\nDone!"