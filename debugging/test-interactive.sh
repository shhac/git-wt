#!/bin/bash

echo "=== Testing git-wt interactive commands ==="
echo ""

# Test 1: Check that go without args shows interactive UI
echo "Test 1: git-wt go (interactive mode)"
echo "q" | timeout 2 ./zig-out/bin/git-wt go 2>&1 | head -20
echo ""

# Test 2: Check that rm without args shows multi-select
echo "Test 2: git-wt rm (multi-select mode)"  
echo -e "\033[B \n" | timeout 2 ./zig-out/bin/git-wt rm 2>&1 | head -20
echo ""

# Test 3: Verify list command works
echo "Test 3: git-wt list"
./zig-out/bin/git-wt list
echo ""

# Test 4: Test new command (should not prompt for Claude anymore)
echo "Test 4: git-wt new test-branch (in temp repo)"
cd /tmp
rm -rf test-git-wt-repo 2>/dev/null
mkdir test-git-wt-repo
cd test-git-wt-repo
git init -q
git config user.email "test@test.com"
git config user.name "Test User"
echo "test" > README.md
git add .
git commit -q -m "initial"

# Run new command and check it doesn't ask about Claude
echo "Testing new command..."
/Users/paul/projects-personal/git-wt/zig-out/bin/git-wt new test-feature 2>&1 | grep -i claude && echo "ERROR: Still prompting for Claude!" || echo "SUCCESS: No Claude prompt found"

# Cleanup
cd /Users/paul/projects-personal/git-wt
rm -rf /tmp/test-git-wt-repo

echo ""
echo "=== Interactive tests complete ==="