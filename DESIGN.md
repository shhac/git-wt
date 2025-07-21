# Design Principles

This project follows clear design principles to ensure maintainability, reliability, and usability.

## Core Principles

### 1. Separation of Concerns
- Each command (`new`, `rm`, `go`) lives in its own module
- Utilities are extracted into focused, reusable modules
- Main.zig only handles argument parsing and dispatch

### 2. Explicit Over Implicit
- All errors are handled explicitly with Zig's error unions
- Memory allocation is always explicit with `defer` cleanup
- No hidden side effects or magic behavior

### 3. User Experience First
- Colored output for readability
- Interactive prompts with sensible defaults (Y/n)
- Clear progress indicators and status messages
- Shell integration for seamless directory navigation

### 4. Safety by Default
- Confirmation prompts for destructive operations
- Validation before execution (branch names, repo state)
- Comprehensive error messages with context

### 5. Testability
- Pure functions wherever possible
- Non-interactive mode for automation
- Comprehensive unit and e2e tests

### 6. Zero Runtime Dependencies (except git)
- Single self-contained binary with no external runtime requirements beyond git
- **Git CLI is an intentional dependency**: This tool is specifically designed to enhance git workflows, so requiring git is both reasonable and beneficial:
  - Avoids reimplementing complex git operations
  - Ensures compatibility with user's git version and configuration
  - Leverages git's battle-tested functionality
  - Reduces maintenance burden significantly
- Uses Zig standard library as the foundation
- Build-time code inclusion is acceptable if:
  - The license permits it (MIT, BSD, Apache, etc.)
  - The code is compiled into the final binary
  - No dynamic linking or external files are required
- The goal is a portable executable that "just works" anywhere git is installed

## Code Patterns

- **Command Pattern**: Each command implements `execute()` and `printHelp()`
- **Resource Management**: Consistent use of `defer` for cleanup
- **Error Context**: Errors bubble up with meaningful messages
- **Composability**: Small functions that combine into larger operations

## Philosophy

"Simple, correct, and helpful" - Do the right thing by default while giving users control when needed.