---
name: git-wt-debug
description: This skill should be used when debugging git-wt issues, diagnosing problems, troubleshooting failures, or investigating unexpected behavior. Invoked when debugging, troubleshooting, investigating errors, or diagnosing issues.
allowed-tools: Read, Bash, Grep, Glob
---

# git-wt Debugging Guide

Diagnose and troubleshoot git-wt issues using built-in debugging features and tools.

## Instructions

### 1. Enable Debug Mode

Use the `--debug` flag for diagnostic information:

```bash
# See fd3 mechanism details
./zig-out/bin/git-wt --debug go

# See git command execution
./zig-out/bin/git-wt --debug new feature-branch

# Combine with other flags
./zig-out/bin/git-wt --debug --show-command go
```

**Debug output includes:**
- Environment variables
- Current working directory
- Command-line arguments
- Git version
- Repository information
- Command execution details

### 2. Common Issues and Solutions

#### Arrow Keys Not Working

**Symptoms:** Arrow keys don't navigate, shows characters instead

**Diagnosis:**
- Check if `--no-tty` flag is set (forces number selection)
- Verify terminal supports raw mode input
- Try explicit TTY mode by running directly (not via pipe)

**Solutions:**
```bash
# Use number selection instead
git-wt --no-tty go

# Check terminal type
echo $TERM  # Should be xterm-256color or similar

# Test without TTY constraints
./zig-out/bin/git-wt go
```

#### Colors Not Showing

**Symptoms:** Output appears plain without ANSI colors

**Diagnosis:**
- Check `TERM` environment variable
- Check if `NO_COLOR` is set
- Verify terminal supports colors

**Solutions:**
```bash
# Check terminal capabilities
echo $TERM
env | grep COLOR

# Use debug mode to see capability detection
git-wt --debug list

# Force plain output if colors are problematic
git-wt --no-color list
```

#### fd3 Mechanism Not Working

**Symptoms:** `gwt go` doesn't change directories

**Diagnosis:**
- Ensure using the shell alias: `eval "$(git-wt --alias gwt)"`
- Check `GWT_USE_FD3` is set when running via alias
- Verify shell function is properly defined

**Solutions:**
```bash
# Test user's fd3 setup
./debugging/test-user-fd3.sh

# Check if alias is defined
type gwt

# Re-create alias
eval "$(git-wt --alias gwt)"

# Debug fd3 mechanism
./debugging/debug-interactive-fd3.sh
```

#### Performance Issues

**Symptoms:** Commands are slow, especially in large repos

**Diagnosis:**
- Check repository size (very large repos may be slow)
- Use `--debug` to see which operations take time
- Profile git command execution

**Solutions:**
```bash
# Use direct branch name instead of interactive selection
git-wt go branch-name

# Check repo size
git count-objects -vH

# Use plain output to reduce rendering overhead
git-wt --plain list
```

### 3. Debugging Scripts

The `debugging/` directory contains helpful scripts:

**Interactive debugging:**
```bash
# Debug fd3 mechanism with your actual shell setup
./debugging/test-user-fd3.sh

# Debug interactive selection
./debugging/debug-interactive-fd3.sh

# Check user's alias configuration
./debugging/check-user-alias.sh
```

**Scratch testing:**
```bash
# Edit this script for quick experimentation
./debugging/test-claude-script.sh
```

### 4. Check Test Failures

When tests fail:

```bash
# Run tests with verbose output
zig test src/main.zig --summary all

# Run specific module test
zig test src/utils/validation.zig

# Check for memory leaks
zig build -Doptimize=Debug
# Run with debug allocator
```

### 5. Investigate Git Issues

When git operations fail:

```bash
# Check git version
git --version

# Verify git repository
git rev-parse --git-dir

# Check worktree list
git worktree list

# Check repository state
git status
```

### 6. Analyze Configuration Issues

When config files cause problems:

```bash
# Check user config
cat ~/.config/git-wt/config

# Check project config
cat .git-wt.toml

# Validate TOML syntax
# (git-wt will fall back to defaults on parse errors)

# Debug config loading
git-wt --debug list
```

## Debugging Tools

### Built-in Flags

- `--debug` - Show diagnostic information
- `--show-command` - Show the cd command that would be executed
- `--no-color` - Disable colors for easier parsing
- `--plain` - Plain output format
- `--no-tty` - Force number-based selection

### Environment Variables

Check these when debugging:

```bash
# Terminal type
echo $TERM

# Language/UTF-8 support
echo $LANG
echo $LC_CTYPE

# git-wt specific
echo $GWT_USE_FD3
echo $NON_INTERACTIVE
echo $NO_COLOR
```

### Log Files

git-wt doesn't create log files by default, but you can redirect output:

```bash
# Capture all output
git-wt new feature 2>&1 | tee debug.log

# Capture only errors
git-wt new feature 2> errors.log
```

## Common Error Messages

### "Not in a git repository"

**Cause:** Current directory is not inside a git repository

**Solution:**
```bash
# Navigate to git repository first
cd /path/to/your/repo
git-wt new feature
```

### "Branch already exists"

**Cause:** Trying to create worktree for existing branch

**Solution:**
```bash
# List existing branches
git branch

# Use different branch name
git-wt new feature-v2
```

### "Worktree already exists"

**Cause:** Worktree directory already exists at target path

**Solution:**
```bash
# List existing worktrees
git worktree list

# Remove old worktree if no longer needed
git-wt rm old-branch

# Or use different parent directory
git-wt new feature --parent-dir /other/path
```

### "Lock file exists"

**Cause:** Another git-wt operation is in progress (or crashed)

**Solution:**
```bash
# Check if another git-wt is running
ps aux | grep git-wt

# If nothing running, remove stale lock
rm .git/git-wt.lock

# Or wait for other operation to complete
```

## Advanced Debugging

### Memory Issues

```bash
# Build with debug info
zig build -Doptimize=Debug

# Check for leaks with valgrind (Linux)
valgrind --leak-check=full ./zig-out/bin/git-wt new test
```

### Compilation Issues

```bash
# Check Zig version
zig version  # Should be 0.15.1+

# Clean build
rm -rf zig-cache zig-out
zig build

# Verbose build
zig build --verbose
```

### CI/CD Issues

When GitHub Actions fail:

```bash
# Check workflow logs
gh run list
gh run view <run-id>

# Download workflow logs
gh run view <run-id> --log

# Re-run failed jobs
# (via GitHub UI)
```

## Troubleshooting Checklist

When debugging issues:

- [ ] Run with `--debug` flag
- [ ] Check git repository status
- [ ] Verify Zig version (0.15.1+)
- [ ] Check environment variables
- [ ] Test with `--non-interactive` mode
- [ ] Try with `--no-color` and `--plain`
- [ ] Check for stale lock files
- [ ] Verify terminal compatibility
- [ ] Review recent code changes
- [ ] Check CHANGELOG.md for known issues
- [ ] Look in BUGS.md for similar problems

## Getting Help

If debugging doesn't resolve the issue:

1. **Check documentation:**
   - docs/TROUBLESHOOTING.md
   - BUGS.md
   - GitHub Issues

2. **Gather diagnostic info:**
   ```bash
   git-wt --debug --version
   git --version
   zig version
   echo $TERM
   uname -a
   ```

3. **Create minimal reproduction:**
   - Simplest steps to reproduce
   - Expected vs actual behavior
   - Full debug output

4. **Report issue:**
   - GitHub: https://github.com/shhac/git-wt/issues
   - Include diagnostic information
   - Attach debug logs if relevant
