# git-wt

A Zig-based CLI tool for managing git worktrees with enhanced features like automatic setup, configuration copying, and interactive navigation.

## Features

- **Create worktrees** with automatic branch creation and setup
- **Remove worktrees** safely with branch cleanup options
- **Navigate between worktrees** interactively or directly
- **Support for branch names with slashes** (creates subdirectory structures)
- Automatic copying of configuration files (.env, .claude, etc.)
- Colored terminal output for better UX

## Installation

### Requirements
- Zig 0.14.1 or later
- Git (obviously!)

### Build from source

```bash
git clone https://github.com/yourusername/git-wt.git
cd git-wt

# Debug build (for development)
zig build

# Release build (optimized)
zig build -Doptimize=ReleaseFast

# Run tests
zig build test

# Install to ~/.local/bin
cp zig-out/bin/git-wt ~/.local/bin/

# Or install system-wide
sudo cp zig-out/bin/git-wt /usr/local/bin/
```

### Building for different platforms

```bash
# macOS (Intel)
zig build -Doptimize=ReleaseFast -Dtarget=x86_64-macos

# macOS (Apple Silicon) 
zig build -Doptimize=ReleaseFast -Dtarget=aarch64-macos

# Linux x86_64
zig build -Doptimize=ReleaseFast -Dtarget=x86_64-linux

# Linux ARM64
zig build -Doptimize=ReleaseFast -Dtarget=aarch64-linux

# Windows
zig build -Doptimize=ReleaseFast -Dtarget=x86_64-windows
```

### Creating a release

```bash
# 1. Update version in build.zig if needed
# 2. Build for all platforms
./scripts/build-release.sh  # (create this script with above commands)

# 3. Create GitHub release
gh release create v0.1.0 \
  --title "v0.1.0" \
  --notes "Initial release" \
  zig-out/bin/git-wt-*
```

### Version management

The version is automatically generated from the build system:

```bash
# Check current version
git-wt --version
git-wt -v

# Set custom version during build
zig build -Dversion="1.2.3"
```

## Setup Shell Integration

Since CLI tools can't change the parent shell's directory, you'll need to set up a shell function wrapper. For detailed setup instructions, run:

```bash
git-wt --help setup
```

Quick setup:
```bash
# Add to your shell configuration
echo 'eval "$(git-wt --alias gwt)"' >> ~/.zshrc  # for zsh
echo 'eval "$(git-wt --alias gwt)"' >> ~/.bashrc # for bash
source ~/.zshrc  # or ~/.bashrc
```

Then use `gwt` instead of `git-wt` for commands that change directories:

```bash
gwt new feature-branch    # Creates worktree AND navigates to it
gwt go main              # Actually changes to the main repository
gwt rm feature-branch    # Removes the feature-branch worktree
```

## Usage

### Create a new worktree

```bash
git-wt new feature-branch

# Branch names with slashes are supported and create subdirectories
git-wt new feature/auth-system
git-wt new bugfix/issue-123
```

This will:
1. Create a new worktree at `../repo-trees/feature-branch` (or subdirectories for slashes)
2. Create and checkout the new branch
3. Copy configuration files from the main repository
4. Optionally start Claude

### Remove a worktree

```bash
# Remove a specific worktree by branch name
git-wt rm feature-branch

# Force removal (skip uncommitted changes check)
git-wt rm feature-branch --force
git-wt rm feature-branch -f

# Non-interactive mode (skip confirmation prompts)
git-wt rm feature-branch --non-interactive
git-wt rm feature-branch -n
```

This will:
1. Validate the branch name against git naming rules
2. Find the worktree for the specified branch
3. Check for uncommitted changes (unless --force is used)
4. Remove the worktree directory
5. Optionally delete the associated branch (interactive mode only)

**Note**: The remove command supports sanitized branch names (slashes converted to hyphens) for compatibility with filesystem paths.

### Navigate to a worktree

```bash
# Interactive mode - shows all worktrees sorted by modification time
git-wt go

# Direct navigation
git-wt go main              # Go to main repository
git-wt go feature-branch    # Go to specific worktree

# Additional options
git-wt go --no-tty          # Force number-based selection (disable arrow keys)
git-wt go --show-command    # Output shell cd commands instead of navigating
git-wt go --no-color        # Disable colored output
git-wt go --plain           # Output plain paths only (one per line)
```

**Interactive Features:**
- Arrow key navigation (when TTY is available)
- Terminal resize handling (SIGWINCH support)
- Real-time display updates
- Graceful fallback to number-based selection

## Configuration Files

The following files and directories are automatically copied when creating new worktrees:
- `.claude` - Claude Code configuration
- `.env*` - All environment files (`.env`, `.env.local`, `.env.development`, `.env.test`, `.env.production`)
- `CLAUDE.local.md` - Local Claude instructions
- `.ai-cache` - AI cache directory (entire directory)

**CLAUDE Files Support:**
The tool has special support for Claude Code integration:
- Automatically detects Claude configuration files
- Preserves AI cache for faster Claude startup
- Maintains local instructions across worktrees
- Optional Claude auto-start after worktree creation (interactive mode only)

## How It Works

1. **Worktree Structure**: Creates worktrees in a parallel directory structure:
   ```
   parent-dir/
   ├── my-repo/          (main repository)
   └── my-repo-trees/    (worktrees)
       ├── feature-a/
       ├── feature-b/
       ├── bugfix-123/
       └── feature/      (subdirectories for slash branches)
           ├── auth/
           └── ui/
               └── dark-mode/
   ```

2. **Configuration Syncing**: Automatically copies important files that are typically gitignored but needed for development (env vars, editor configs, etc.)

3. **Smart Navigation**: The `go` command sorts worktrees by modification time, making it easy to jump to recently used branches.

## Testing

### Unit Tests

```bash
# Run all unit tests
zig build test
```

### Non-Interactive Mode

The tool supports a `--non-interactive` (or `-n`) flag for testing and automation:

```bash
# Create worktree without prompts
git-wt --non-interactive new feature-branch

# Remove worktree without confirmation
git-wt --non-interactive rm feature-branch

# List worktrees without interactive selection
git-wt --non-interactive go

# Navigate directly to a worktree (outputs cd command)
git-wt --non-interactive go feature-branch
```

### End-to-End Testing

Multiple test scripts are provided:

```bash
# Run non-interactive tests
./test-non-interactive.sh

# Test shell integration (requires shell alias setup)
./test-shell-integration.sh

# Run integration tests
zig build test-integration
```

The test scripts will:
- Build the binary
- Create temporary git repositories in `.e2e-test` directory
- Test all commands in various modes
- Validate actual outcomes (not just command execution)
- Clean up after themselves

**Test Directory:**
The `.e2e-test` directory is used for all test data and is gitignored to prevent test artifacts from being committed.

## Development

See [DESIGN.md](DESIGN.md) for the design principles and patterns used in this project.

```bash
# Run all tests
zig build test

# Run integration tests specifically
zig build test-integration

# Build debug version
zig build

# Run directly without installing
zig build run -- new test-branch

# Enable debug output
git-wt --debug new test-branch

# Build with custom version
zig build -Dversion="dev-1.0.0"
```

### Debug Mode

Enable debug output with the `--debug` flag to see detailed information about:
- Git operations and their output
- File system operations
- Lock acquisition and release
- Configuration file copying
- Process execution details

### Concurrent Operation Protection

The tool uses file-based locking to prevent concurrent worktree operations:
- Lock files are created in `.git/git-wt.lock`
- Automatic stale lock cleanup (detects if process died)
- 30-second timeout for lock acquisition
- Clean error messages for lock conflicts

### Project Structure

```
src/
├── main.zig                  # CLI entry point and command dispatch
├── commands/                 # Command implementations
│   ├── new.zig              # Create worktree with setup
│   ├── remove.zig           # Remove worktree with safety checks
│   └── go.zig               # Navigate between worktrees
├── utils/                   # Shared utilities
│   ├── git.zig              # Git operations and repository info
│   ├── fs.zig               # Filesystem helpers and config copying
│   ├── colors.zig           # Terminal colors and formatted output
│   ├── input.zig            # User input handling and confirmations
│   ├── process.zig          # External command execution
│   ├── validation.zig       # Branch name and path validation
│   ├── lock.zig             # File-based locking for concurrent operations
│   ├── fd.zig               # File descriptor 3 (fd3) shell integration
│   ├── interactive.zig      # Interactive UI with arrow key navigation
│   ├── time.zig             # Time formatting utilities
│   └── debug.zig            # Debug output utilities
└── integration_tests.zig    # Integration tests without git dependencies
```

## Command Reference

### Global Options

Available for all commands:

```bash
git-wt --help                    # Show general help
git-wt --version, -v             # Show version information
git-wt --debug                   # Enable debug output
git-wt --non-interactive, -n     # Disable all interactive prompts
git-wt --alias <name>            # Generate shell integration function
```

### Command-Specific Options

#### `git-wt new`
```bash
git-wt new <branch-name>
  -h, --help                     # Show command help
  -n, --non-interactive          # Skip all prompts (no Claude startup)
  -p, --parent-dir <path>        # Custom parent directory for worktree
```

#### `git-wt rm`
```bash
git-wt rm [branch-name]
  -h, --help                     # Show command help
  -n, --non-interactive          # Skip confirmation prompts
  -f, --force                    # Skip uncommitted changes check
```

#### `git-wt go`
```bash
git-wt go [branch-name]
  -h, --help                     # Show command help
  -n, --non-interactive          # List worktrees without interaction
  --no-tty                       # Force number-based selection
  --show-command                 # Output shell cd commands
  --no-color                     # Disable colored output
  --plain                        # Output plain paths only
```

### Environment Variables

- `NO_COLOR=1` - Disable colored output globally
- `DEBUG=1` - Enable debug output (alternative to `--debug`)

### Exit Codes

- `0` - Success
- `1` - General error (invalid arguments, git errors, etc.)
- `2` - Not in a git repository
- `3` - Branch already exists (new command)
- `4` - Worktree not found (go/rm commands)
- `5` - Lock timeout (concurrent operation)

## Advanced Features

### Shell Integration (fd3 Mechanism)

The tool uses a sophisticated shell integration system via file descriptor 3 (fd3):

```bash
# The shell alias function sets up fd3 for communication
eval "$(git-wt --alias gwt)"

# Now gwt commands can change the shell's directory
gwt go feature-branch    # Actually changes shell directory
gwt new my-feature      # Creates worktree AND navigates to it
```

**How it works:**
1. The alias function opens fd3 for reading
2. git-wt detects fd3 and writes cd commands to it
3. The shell function reads from fd3 and executes the commands
4. This allows the CLI tool to change the parent shell's directory

### Locking Mechanism

Protects against concurrent worktree operations:
- **Lock file location**: `.git/git-wt.lock`
- **Lock timeout**: 30 seconds
- **Stale lock detection**: Automatically cleans up locks from dead processes
- **Process ID tracking**: Stores PID and timestamp in lock files

### Branch Name Validation

Comprehensive validation following git standards:
- **Invalid characters**: No spaces, control characters, or `~^:?*[`
- **Path components**: No `.` or `..` components
- **Reserved names**: Rejects `HEAD`, `-`, `@`
- **Format rules**: No consecutive dots, proper start/end characters
- **Case sensitivity**: Detects conflicts on case-insensitive filesystems

### Path Sanitization

Handles complex branch names safely:
- **Slash conversion**: `feature/auth` → `feature-auth` for filesystem compatibility
- **Directory creation**: Supports subdirectory structures when using `--parent-dir`
- **Display consistency**: Shows relative names consistently across commands

## Troubleshooting

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
eval "$(git-wt --alias gwt)"

# Add to shell configuration permanently
echo 'eval "$(git-wt --alias gwt)"' >> ~/.zshrc
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

## Contributing

1. Fork the repository
2. Create your feature branch (`git-wt new my-feature`)
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

MIT