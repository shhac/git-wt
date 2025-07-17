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

### 6. Zero Dependencies
- Uses only Zig standard library
- Single self-contained binary
- No external runtime requirements

## Code Patterns

- **Command Pattern**: Each command implements `execute()` and `printHelp()`
- **Resource Management**: Consistent use of `defer` for cleanup
- **Error Context**: Errors bubble up with meaningful messages
- **Composability**: Small functions that combine into larger operations

## Philosophy

"Simple, correct, and helpful" - Do the right thing by default while giving users control when needed.