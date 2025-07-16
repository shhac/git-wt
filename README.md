# git-wt

A Zig-based CLI tool for managing git worktrees with enhanced features like automatic setup, configuration copying, and interactive navigation.

## Features

- **Create worktrees** with automatic branch creation and setup
- **Remove worktrees** safely with branch cleanup options
- **Navigate between worktrees** interactively or directly
- **Support for branch names with slashes** (creates subdirectory structures)
- Automatic copying of configuration files (.env, .claude, etc.)
- Node.js project support (nvm, yarn detection)
- Colored terminal output for better UX

## Installation

### Requirements
- Zig 0.14.1 or later
- Git (obviously!)
- Optional: nvm and yarn for Node.js project support

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
# 1. Update version in src/main.zig
# 2. Build for all platforms
./scripts/build-release.sh  # (create this script with above commands)

# 3. Create GitHub release
gh release create v0.1.0 \
  --title "v0.1.0" \
  --notes "Initial release" \
  zig-out/bin/git-wt-*
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
gwt rm                   # Works the same as git-wt rm
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
4. Run `nvm use` if .nvmrc exists
5. Run `yarn install` if package.json with yarn is detected
6. Optionally start Claude

### Remove current worktree

```bash
git-wt rm
```

This will:
1. Confirm you're in a worktree (not main repository)
2. Navigate back to the main repository
3. Remove the worktree
4. Optionally delete the associated branch

### Navigate to a worktree

```bash
# Interactive mode - shows all worktrees sorted by modification time
git-wt go

# Direct navigation
git-wt go main              # Go to main repository
git-wt go feature-branch    # Go to specific worktree
```

## Configuration Files

The following files are automatically copied when creating new worktrees:
- `.claude` - Claude Code configuration
- `.env`, `.env.local`, `.env.development`, `.env.test`, `.env.production`
- `CLAUDE.local.md` - Local Claude instructions
- `.ai-cache` - AI cache directory

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
git-wt --non-interactive rm

# List worktrees without interactive selection
git-wt --non-interactive go

# Navigate directly to a worktree (outputs cd command)
git-wt --non-interactive go feature-branch
```

### End-to-End Testing

A simple test script is provided:

```bash
# Run non-interactive tests
./test-non-interactive.sh
```

The test script will:
- Build the binary
- Create a temporary git repository
- Test all commands in non-interactive mode
- Clean up after itself

## Development

```bash
# Run tests
zig build test

# Build debug version
zig build

# Run directly without installing
zig build run -- new test-branch

# Enable verbose output (if implemented)
DEBUG=1 git-wt new test-branch
```

### Project Structure

```
src/
├── main.zig           # CLI entry point and command dispatch
├── commands/          # Command implementations
│   ├── new.zig       
│   ├── remove.zig    
│   └── go.zig        
└── utils/            # Shared utilities
    ├── git.zig       # Git operations
    ├── fs.zig        # Filesystem helpers
    ├── colors.zig    # Terminal colors
    ├── input.zig     # User input handling
    └── process.zig   # Process execution
```

## Troubleshooting

### "Not in a git repository" error
Make sure you're running the command from within a git repository.

### Colors not showing
The tool uses ANSI escape codes. Make sure your terminal supports them. You can disable colors by setting `NO_COLOR=1`.

### nvm/yarn commands not found
These are optional dependencies. The tool will skip them if not installed.

## Contributing

1. Fork the repository
2. Create your feature branch (`git-wt new my-feature`)
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

MIT