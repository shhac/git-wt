# TODO

## Features

### Support slashes in branch names
- Allow branch names like `feature/auth-system` or `bugfix/issue-123`
- Currently slashes in branch names would create subdirectories in the worktree path
- Need to handle path construction to flatten or escape slashes appropriately
- Consider using underscore or dash replacement for the directory name while preserving the actual branch name

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