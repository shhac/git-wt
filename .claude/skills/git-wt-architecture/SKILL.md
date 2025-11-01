---
name: git-wt-architecture
description: This skill should be used when navigating the git-wt codebase, understanding the project structure, locating specific functionality, or learning how components interact. Invoked when asking "where is", "find code for", "how does X work", or understanding architecture.
allowed-tools: Read, Grep, Glob
---

# git-wt Architecture Guide

Navigate and understand the git-wt codebase structure, patterns, and component organization.

## Instructions

### 1. Understand Project Structure

```
git-wt/
├── src/
│   ├── main.zig              # Entry point, command dispatch
│   ├── commands/             # Command implementations
│   │   ├── new.zig          # Create worktree
│   │   ├── remove.zig       # Remove worktree(s)
│   │   ├── go.zig           # Navigate between worktrees
│   │   ├── list.zig         # List all worktrees
│   │   ├── alias.zig        # Generate shell wrapper
│   │   ├── clean.zig        # Clean deleted branches
│   │   └── *_test.zig       # Unit tests for each command
│   ├── utils/               # Utility modules
│   │   ├── args.zig         # CLI argument parsing
│   │   ├── colors.zig       # ANSI colors
│   │   ├── config.zig       # Configuration file support
│   │   ├── debug.zig        # Debug logging
│   │   ├── env.zig          # Environment variables
│   │   ├── fd.zig           # File descriptor 3 (fd3) for shell integration
│   │   ├── fs.zig           # File operations
│   │   ├── git.zig          # Git command wrapper
│   │   ├── input.zig        # User input utilities
│   │   ├── interactive.zig  # Terminal control (arrow keys, etc.)
│   │   ├── io.zig           # I/O wrappers
│   │   ├── lock.zig         # File locking
│   │   ├── process.zig      # External command execution
│   │   ├── time.zig         # Time formatting
│   │   └── validation.zig   # Input validation
│   ├── integration_tests.zig # Integration tests
│   └── tests/               # Additional test files
├── docs/                    # User documentation
├── test-interactive/        # Expect-based interactive tests
├── scripts/                 # Build and test scripts
├── .github/workflows/       # CI/CD workflows
└── build.zig               # Zig build configuration
```

### 2. Locate Specific Functionality

**To find where feature X is implemented:**

```bash
# Search for function/feature
grep -r "functionName" src/

# Find file by name pattern
find src/ -name "*keyword*"

# Search for specific pattern
grep -r "pattern" src/commands/
```

**Common locations:**

- **Command logic:** `src/commands/<command>.zig`
- **Git operations:** `src/utils/git.zig`
- **File operations:** `src/utils/fs.zig`
- **User interaction:** `src/utils/interactive.zig`
- **Configuration:** `src/utils/config.zig`
- **Validation:** `src/utils/validation.zig`

### 3. Understanding Key Patterns

#### Command Table Pattern

**Location:** `src/main.zig:30-37`

Commands are registered in a table for clean dispatch:

```zig
const commands = [_]Command{
    .{ .name = "new", .execute = executeNew, .help = cmd_new.printHelp },
    .{ .name = "rm", .execute = executeRemove, .help = cmd_remove.printHelp },
    // ...
};
```

#### GitResult Pattern

**Location:** `src/utils/git.zig:36-50`

Explicit success/failure handling for git commands:

```zig
pub const GitResult = union(enum) {
    success: []u8,
    failure: struct {
        exit_code: u8,
        stderr: []u8,
    },
};
```

#### Configuration Precedence

**Location:** `src/utils/config.zig` and `src/main.zig:149-165`

Config loading and merging:
1. Load user config (`~/.config/git-wt/config`)
2. Load project config (`.git-wt.toml`)
3. Merge configs (project overrides user)
4. Apply CLI flags (flags override config)

#### Resource Management

Consistent use of `defer` for cleanup throughout codebase:

```zig
var buffer = std.ArrayList(u8).empty;
defer buffer.deinit(allocator);
```

### 4. Component Interactions

#### Command Flow

```
main.zig (entry)
    ↓
Parse global flags
    ↓
Load configuration (config.zig)
    ↓
Dispatch to command (commands/*.zig)
    ↓
Use utilities (utils/*.zig)
    ↓
Execute git operations (git.zig)
    ↓
Return to shell (fd.zig for directory changes)
```

#### Configuration Flow

```
User config (~/.config/git-wt/config)
    ↓
Project config (.git-wt.toml)
    ↓
Merge (config.zig)
    ↓
Apply CLI flags
    ↓
Pass to commands
```

#### Interactive Selection Flow

```
Interactive command (go, rm)
    ↓
Check TTY availability (interactive.zig)
    ↓
Arrow keys available? → Arrow navigation
    ↓
No TTY? → Number selection
    ↓
User selects
    ↓
Execute action
```

### 5. Finding Code Examples

**To understand how to:**

- **Parse CLI args:** Read `src/utils/args.zig`
- **Execute git commands:** Read `src/utils/git.zig:exec()`
- **Handle user input:** Read `src/utils/input.zig`
- **Navigate interactively:** Read `src/utils/interactive.zig`
- **Validate branch names:** Read `src/utils/validation.zig`
- **Load config files:** Read `src/utils/config.zig`
- **Lock operations:** Read `src/utils/lock.zig`

### 6. Understanding Tests

**Test organization:**

- **Unit tests:** Co-located with modules (`*_test.zig`)
- **Integration tests:** `src/integration_tests.zig`
- **Interactive tests:** `test-interactive/*.exp`
- **Test registry:** `src/*/test_all.zig`

**Find tests for feature X:**

```bash
# Find test file
find src/ -name "*test.zig" | xargs grep -l "feature"

# Or use glob
grep -r "test.*feature" src/
```

## Design Principles

**From DESIGN.md:**

1. **Separation of Concerns**
   - Each command in own module
   - Utilities are focused and reusable
   - Main.zig only handles dispatch

2. **Explicit Over Implicit**
   - All errors handled explicitly
   - Memory allocation always explicit with `defer`
   - No hidden side effects

3. **User Experience First**
   - Colored output
   - Interactive prompts
   - Clear progress indicators
   - Shell integration

4. **Safety by Default**
   - Confirmation prompts
   - Validation before execution
   - Comprehensive error messages

5. **Testability**
   - Pure functions
   - Non-interactive mode
   - Comprehensive tests

6. **Zero Runtime Dependencies**
   - Only git required
   - Uses Zig standard library
   - Single self-contained binary

## Key Code Locations

### Entry Points

- **Main entry:** `src/main.zig:128` (main function)
- **Command dispatch:** `src/main.zig:287` (command loop)
- **Config loading:** `src/main.zig:154`

### Core Features

- **Worktree creation:** `src/commands/new.zig`
- **Worktree removal:** `src/commands/remove.zig`
- **Navigation:** `src/commands/go.zig`
- **Listing:** `src/commands/list.zig`
- **Shell alias:** `src/commands/alias.zig`
- **Cleanup:** `src/commands/clean.zig`

### Utilities

- **Git wrapper:** `src/utils/git.zig:52` (exec function)
- **Config parser:** `src/utils/config.zig:108` (parseConfigContent)
- **Interactive UI:** `src/utils/interactive.zig:360` (selectFromListUnified)
- **File locking:** `src/utils/lock.zig`
- **Branch validation:** `src/utils/validation.zig`

### Shell Integration

- **fd3 mechanism:** `src/utils/fd.zig`
- **Alias generation:** `src/commands/alias.zig`
- **Directory navigation:** Uses fd3 to communicate with parent shell

## Zig-Specific Patterns

### Memory Management

```zig
// Arena allocator for CLI operations
var arena = std.heap.ArenaAllocator.init(gpa.allocator());
defer arena.deinit();

// Always free from exec calls
const output = try git.exec(allocator, &.{"status"});
defer allocator.free(output);
```

### Error Handling

```zig
// Explicit error unions
pub fn execute() !void {
    // ...
}

// Error bubbling with context
return GitError.CommandFailed;
```

### Const Correctness

```zig
// openDir returns const Dir
const dir = try std.fs.cwd().openDir(".", .{});
// Must use const when assigning
```

## Common Code Patterns

### Reading Configuration

```zig
// From src/main.zig
var cfg = config.loadConfig(allocator) catch config.Config{};
defer cfg.deinit(allocator);
```

### Executing Git Commands

```zig
// From src/utils/git.zig
const result = try execWithResult(allocator, &.{"worktree", "list"});
defer result.deinit(allocator);
```

### Interactive Selection

```zig
// From src/commands/go.zig
const selection = try interactive.selectFromList(
    allocator,
    "Select worktree:",
    options,
    no_tty
);
```

### Validation

```zig
// From src/commands/new.zig
try validation.validateBranchName(branch_name);
```

## References

- **DESIGN.md** - Design principles and patterns
- **docs/ADVANCED.md** - Advanced technical details
- **learnings/** - Technical deep-dives on specific topics
