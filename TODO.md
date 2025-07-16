# TODO

## Features

### Improve `go` command behavior
- Current location/worktree should not be shown in the list
- Default behavior if no index is provided should be to go to the most recent worktree (index 1)
- Worktrees should be ordered by most recently modified first (currently implemented but verify)
- Show the root repo as [main] (not [main repository]) unless we're already there
- Fix: Interactive mode doesn't actually change pwd in shell (CLI runs in subprocess)

### Add branch argument to rm command
- Allow `git-wt rm <branch-name>` to remove a specific worktree
- Currently rm only works on the current worktree
- Should navigate to the worktree first, then remove it
- Example: `git-wt rm feature-branch` removes the feature-branch worktree
- Should work with branch names containing slashes

### Support invoking commands from subdirectories
- `go` and `rm` should work when invoked from subdirectories within a worktree
- Currently may fail if not run from worktree root
- Need to find git root directory first before operations

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