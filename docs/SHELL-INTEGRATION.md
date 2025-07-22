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
```

## Usage with Shell Integration

Then use `gwt` instead of `git-wt` for commands that change directories:

```bash
gwt new feature-branch    # Creates worktree AND navigates to it
gwt go main              # Actually changes to the main repository
gwt rm feature-branch    # Removes the feature-branch worktree
```

## How It Works (fd3 Mechanism)

The tool uses a sophisticated shell integration system via file descriptor 3 (fd3):

1. The alias function opens fd3 for reading
2. git-wt detects fd3 and writes cd commands to it
3. The shell function reads from fd3 and executes the commands
4. This allows the CLI tool to change the parent shell's directory

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
git-wt --help setup
```