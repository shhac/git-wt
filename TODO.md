# TODO

## Features

### Support slashes in branch names
- Allow branch names like `feature/auth-system` or `bugfix/issue-123` to create subdirectory structures
- Subdirectories in worktree paths are fine and actually desirable for organization
- Need to update the `go` command to discover worktrees in subdirectories recursively
- Need to handle other characters that are valid in git branch names but illegal in file paths:
  - Colon `:` (Windows)
  - Question mark `?` (Windows)
  - Asterisk `*` (Windows/Unix)
  - Pipe `|` (Windows)
  - Less than/greater than `<>` (Windows)
  - Double quote `"` (Windows)
- Consider escaping strategy: URL encoding, underscore replacement, or other approaches
- Ensure branch name is preserved correctly in git while filesystem path is safe

### Support specifying worktree parent directory with command line flag
- Add `--parent-dir` or `-p` flag to `git-wt new` command
- Allow users to override the default `../repo-trees/` location
- Example: `git-wt new feature-branch --parent-dir ~/worktrees`
- Should validate the parent directory exists and is writable
- Update help documentation to explain this option

## Future Enhancements

### Additional ideas for consideration
- Add `list` command to show all worktrees with their status
- Support for `--force` flag on removal to skip confirmation prompts
- Add `clean` command to remove all worktrees for deleted branches
- Support for custom worktree naming patterns via config file
- Integration with git aliases for even shorter commands