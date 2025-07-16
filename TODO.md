# TODO

## Features

### Support specifying worktree parent directory with command line flag
- Add `--parent-dir` or `-p` flag to `git-wt new` command
- Allow users to override the default `../repo-trees/` location
- Example: `git-wt new feature-branch --parent-dir ~/worktrees`
- Should validate the parent directory exists and is writable
- Update help documentation to explain this option

### Add --debug flag
- Add a global `--debug` flag that shows diagnostic information
- Should display the current working directory where git-wt is invoked from
- Useful for troubleshooting path-related issues
- Could also show git repository information and worktree status

## Future Enhancements

### Additional ideas for consideration
- Add `list` command to show all worktrees with their status
- Support for `--force` flag on removal to skip confirmation prompts
- Add `clean` command to remove all worktrees for deleted branches
- Support for custom worktree naming patterns via config file
- Integration with git aliases for even shorter commands