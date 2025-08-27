# TODO

## Features

### Add `clean` command
- Remove all worktrees for deleted branches
- List worktrees that would be removed before confirmation
- Support `--dry-run` to show what would be cleaned without doing it
- Support `--force` to skip confirmation

## Future Enhancements

### Additional ideas for consideration
- Support for custom worktree naming patterns via config file
- Integration with git aliases for even shorter commands
- Add `--json` output format for scripting
- Support for worktree templates (predefined setups for different project types)
- Add `move` command to relocate a worktree
- Support for `.gitworktree` config file for project-specific settings