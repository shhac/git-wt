# Troubleshooting

## Common Issues

### "Not in a git repository" error
Make sure you're running the command from within a git repository.

### "Another git-wt operation is in progress" error
Another instance is running, or a stale lock exists:
```bash
# Wait for the operation to complete, or
# Remove stale lock manually (only if process is definitely dead)
rm .git/git-wt.lock
```

### Colors not showing
The tool uses ANSI escape codes. Options to fix:
- Ensure your terminal supports ANSI colors
- Use `--no-color` flag to disable colors
- Set `NO_COLOR=1` environment variable

### Interactive mode not working
Requires TTY for both input and output:
- Use `--no-tty` flag to force number-based selection
- Check that stdin/stdout are connected to a terminal
- In scripts, use `--non-interactive` flag

### Shell integration not working
Ensure the alias is properly set up:
```bash
# Check if alias exists
type gwt

# Recreate alias
eval "$(git-wt alias gwt)"

# Add to shell configuration permanently
echo 'eval "$(git-wt alias gwt)"' >> ~/.zshrc
```

### Case-insensitive filesystem conflicts
On macOS/Windows, branch names differing only in case will conflict:
```bash
# This will fail if 'Feature' directory already exists
git-wt new feature
```
Use different branch names to avoid conflicts.

### "Repository is not in a clean state" error
Complete any ongoing git operations:
```bash
# Check repository status
git status

# Complete or abort ongoing operations
git merge --abort     # or --continue
git rebase --abort    # or --continue
git cherry-pick --abort  # or --continue
```

## Advanced Troubleshooting

### Lock File Issues
If operations are getting stuck with lock errors:

1. **Check for running processes**:
   ```bash
   ps aux | grep git-wt
   ```

2. **Check lock file contents**:
   ```bash
   cat .git/git-wt.lock
   ```
   
3. **Force removal** (only if no processes are running):
   ```bash
   rm .git/git-wt.lock
   ```

### Debug Mode
Enable detailed logging:
```bash
git-wt --debug <command>
```

This shows:
- Git command execution and output
- File operations and permissions
- Lock acquisition attempts
- Process spawning details

### Environment Variables

- `NO_COLOR=1` - Disable colored output globally
- `DEBUG=1` - Enable debug output (alternative to `--debug`)

## Exit Codes

- `0` - Success
- `1` - General error (invalid arguments, git errors, etc.)
- `2` - Not in a git repository
- `3` - Branch already exists (new command)
- `4` - Worktree not found (go/rm commands)
- `5` - Lock timeout (concurrent operation)