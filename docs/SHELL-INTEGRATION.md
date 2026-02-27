# Shell Integration

Since CLI tools can't change the parent shell's directory, you need to set up a shell function wrapper for commands that navigate between directories.

## Quick Setup

```bash
# Generate and add to your shell configuration
echo 'eval "$(git-wt alias gwt)"' >> ~/.zshrc  # for zsh
echo 'eval "$(git-wt alias gwt)"' >> ~/.bashrc # for bash
source ~/.zshrc  # or ~/.bashrc
```

## Advanced Options

```bash
# Always force number-based selection
git-wt alias gwt --no-tty

# Always use plain output
git-wt alias gwt --plain

# Custom parent directory
git-wt alias gwt --parent-dir "../my-worktrees"

# Dynamic parent with repo name
git-wt alias gwt --parent-dir "../{repo}-trees"

# Use a different file descriptor (default is 3)
git-wt alias gwt --fd 5
```

## Usage with Shell Integration

Then use `gwt` instead of `git-wt` for commands that change directories:

```bash
gwt new feature-branch    # Creates worktree AND navigates to it
gwt go main              # Actually changes to the main repository
gwt rm feature-branch    # Removes the feature-branch worktree
```

## How It Works (File Descriptor Mechanism)

The tool uses a shell integration system via a configurable file descriptor (fd3 by default):

1. The alias function sets `GWT_FD=N` and opens that fd for reading
2. git-wt detects `GWT_FD` and writes `cd` commands to the specified fd
3. The shell function reads from the fd and `eval`s the commands
4. This allows the CLI tool to change the parent shell's directory

If fd3 conflicts with another tool in your environment, use `--fd <N>` (3-9) when generating the alias:
```bash
eval "$(git-wt alias gwt --fd 5)"
```

## Without Shell Alias (Bare Mode)

When running `git-wt` directly (without the shell alias), it cannot change your
shell's working directory. Instead, it outputs the worktree path:

**Interactive terminal (stdout is a TTY):**
The tool shows a copy-paste hint on stderr:
```
→ cd '/path/to/worktree'
```

**Piped or scripted (stdout is not a TTY):**
The tool outputs the raw path on stdout, suitable for command substitution:
```bash
cd "$(git-wt go feature-branch)"
cd "$(git-wt go main)"
cd "$(git-wt new my-feature)"
```

All informational output (progress messages, prompts) goes to stderr in this mode,
keeping stdout clean for the path.

## Troubleshooting Shell Integration

### Alias not working
```bash
# Check if alias exists
type gwt

# Recreate alias
eval "$(git-wt alias gwt)"

# Add to shell configuration permanently
echo 'eval "$(git-wt alias gwt)"' >> ~/.zshrc
```

### Commands not changing directories
Make sure you're using the aliased command (`gwt`) rather than the direct binary (`git-wt`) for navigation commands.

For detailed setup instructions, run:
```bash
git-wt alias --help
```