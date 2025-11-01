# git-wt

A fast, reliable CLI tool for managing git worktrees with enhanced features like automatic setup, configuration copying, and interactive navigation.

## Features

- **Create worktrees** with automatic branch creation and configuration copying
- **Remove worktrees** safely with multi-select support and branch cleanup
- **Navigate between worktrees** interactively with smart sorting
- **List all worktrees** with details or machine-readable format
- **Clean deleted branches** automatically remove worktrees for branches deleted from remote
- **Configuration file support** with user-level and project-level config files
- **Generate shell aliases** for seamless directory navigation
- **Support for complex branch names** with slashes and special characters
- Built with [Zig](https://ziglang.org/) for performance and zero runtime dependencies

## Quick Start

### Installation

**Option 1: Download Pre-built Binary (Recommended)**

Download the latest release for your platform from [Releases](https://github.com/shhac/git-wt/releases):

```bash
# macOS (Universal - works on both Intel and Apple Silicon)
curl -L https://github.com/shhac/git-wt/releases/latest/download/git-wt-macos-universal.tar.gz | tar xz
chmod +x git-wt
mv git-wt ~/.local/bin/

# Linux x86_64
curl -L https://github.com/shhac/git-wt/releases/latest/download/git-wt-linux-x86_64.tar.gz | tar xz
chmod +x git-wt
mv git-wt ~/.local/bin/
```

**Option 2: Build from Source**

```bash
git clone https://github.com/shhac/git-wt.git
cd git-wt
zig build -Doptimize=ReleaseFast
cp zig-out/bin/git-wt ~/.local/bin/
```

**Requirements**:
- Runtime: Git only
- Build from source: Zig 0.15.1+, Git

**Windows users**: Use WSL2 with the Linux build

### Shell Integration Setup

```bash
# Add to your shell configuration (.zshrc, .bashrc, etc.)
echo 'eval "$(git-wt alias gwt)"' >> ~/.zshrc
source ~/.zshrc
```

Now use `gwt` for commands that change directories.

### Basic Usage

```bash
# Create a new worktree
gwt new feature-branch

# Navigate between worktrees (interactive)
gwt go

# Remove worktrees (supports multi-select)
gwt rm

# List all worktrees  
git-wt list
```

## Commands

| Command | Description | Example |
|---------|-------------|---------|
| `new <branch>` | Create new worktree with branch | `gwt new feature/auth` |
| `rm [branch...]` | Remove worktree(s), interactive if no args | `gwt rm old-feature` |
| `go [branch]` | Navigate to worktree, interactive if no args | `gwt go main` |
| `list` | List all worktrees with details | `git-wt list --plain` |
| `clean` | Remove worktrees for deleted branches | `git-wt clean --dry-run` |
| `alias <name>` | Generate shell function for directory navigation | `git-wt alias gwt` |

### Interactive Features

- **Arrow key navigation** (↑/↓) with automatic fallback to number selection
- **Multi-select removal**: Use Space to toggle `[*]`/`[ ]`, Enter to confirm
- **Smart sorting**: Worktrees sorted by modification time (most recent first)
- **Terminal resize handling** and graceful interrupt handling

## Documentation

- **[Installation Guide](docs/INSTALLATION.md)** - Detailed build and install instructions
- **[Usage Guide](docs/USAGE.md)** - Complete command reference and examples
- **[Configuration](docs/CONFIGURATION.md)** - Config file setup and options
- **[Shell Integration](docs/SHELL-INTEGRATION.md)** - Setup and troubleshooting shell aliases
- **[Testing](docs/TESTING.md)** - Running tests and development workflows
- **[Advanced Features](docs/ADVANCED.md)** - Configuration copying, locking, performance
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Design Principles](DESIGN.md)** - Architecture and development guidelines

## How It Works

Creates worktrees in parallel directory structure:
```
my-repo/              # Main repository  
my-repo-trees/        # Worktrees directory
├── feature-a/
├── feature-b/
└── feature/
    ├── auth/         # Supports branch names with slashes
    └── ui/
```

- **Configuration syncing**: Automatically copies `.env*`, `.claude`, `CLAUDE.local.md`, `.ai-cache`
- **Safety by default**: Confirmation prompts, uncommitted changes detection, concurrent operation locking
- **Shell integration**: Uses file descriptor 3 (fd3) to enable directory changes from CLI

## Contributing

1. Fork the repository
2. Create your feature branch (`git-wt new my-feature`)
3. Commit your changes  
4. Push to the branch
5. Create a Pull Request

Please ensure all tests pass (`zig build test`) and follow existing code patterns.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Zig](https://ziglang.org/) for performance and reliability
- Inspired by the need for better git worktree management workflows