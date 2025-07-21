# TODO

## Features

### Improve `go` command behavior
- Default behavior if no index is provided should be to go to the most recent worktree (index 1)
- Worktrees should be ordered by most recently modified first (currently implemented but verify)



### Support specifying worktree parent directory with command line flag
- Add `--parent-dir` or `-p` flag to `git-wt new` command
- Allow users to override the default `../repo-trees/` location
- Example: `git-wt new feature-branch --parent-dir ~/worktrees`
- Should validate the parent directory exists and is writable
- Update help documentation to explain this option


## Future Enhancements

### Additional ideas for consideration
- Add `clean` command to remove all worktrees for deleted branches
- Support for custom worktree naming patterns via config file
- Integration with git aliases for even shorter commands