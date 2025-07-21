#!/bin/bash
set -e

echo "Testing specific copy bug scenario..."

# Clean up from any previous tests
rm -rf .e2e-test/bug-test
mkdir -p .e2e-test/bug-test

# Test case: Running git-wt from a different directory
cd .e2e-test/bug-test
git init main-repo
cd main-repo
echo "# Test" > README.md
git add . && git commit -m "init"

# Create files to copy
echo "SHOULD_COPY=yes" > .env
mkdir -p .claude
echo '{"test": true}' > .claude/config.json

echo
echo "Files in main repo:"
ls -la

# Now run git-wt from a parent directory (simulating user running from outside repo)
cd ..
echo
echo "Running git-wt from parent directory..."
pwd
# Must be in the repo to run git-wt
cd main-repo
../../../zig-out/bin/git-wt -n new test-from-parent

# Check if files were copied
echo
echo "Checking if files were copied to worktree..."
echo "Looking in: $(pwd)"
ls -la ../main-repo-trees/test-from-parent/ || echo "Directory doesn't exist"

if [ -f ../main-repo-trees/test-from-parent/.env ]; then
    echo "✓ .env was copied"
else
    echo "✗ .env was NOT copied"
fi

if [ -d ../main-repo-trees/test-from-parent/.claude ]; then
    echo "✓ .claude was copied"
else
    echo "✗ .claude was NOT copied"
fi