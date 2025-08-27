# How to Test TTY Inputs

Testing interactive TTY applications can be challenging because many testing methods don't properly simulate a real terminal. This document covers various approaches we used while debugging git-wt.

## The Challenge

When testing `gwt go` interactively, we need to:
1. Provide a real TTY so the application detects interactive mode
2. Send keyboard input (numbers, arrow keys, Enter)
3. Capture output from multiple file descriptors (stdout, stderr, fd3)
4. Verify that shell functions work correctly

## Testing Methods

### 1. Piping Input (Limited)

```bash
echo "1" | gwt go
```

**Pros:**
- Simple and scriptable
- Works for number-based input

**Cons:**
- Doesn't provide a real TTY
- May trigger different code paths
- Runs in a subshell (directory changes don't persist)

### 2. Here-Strings and Here-Documents

```bash
# Here-string
gwt go <<< "1"

# Here-document
gwt go << EOF
1
EOF
```

**Pros:**
- Slightly better than pipes
- Still scriptable

**Cons:**
- Still doesn't provide a real TTY
- Limited to simple input

### 3. Expect Scripts (RECOMMENDED)

**Update**: After extensive testing, expect proved to be the most effective solution. See [TESTING_INTERACTIVE_CLIS_WITH_EXPECT.md](./TESTING_INTERACTIVE_CLIS_WITH_EXPECT.md) for comprehensive guide.

```expect
#!/usr/bin/expect -f
spawn gwt go
expect "Select" {
    send "\r"
}
expect eof
```

**Pros:**
- Provides a real pseudo-TTY
- Can send complex sequences (arrow keys, ESC, space)
- Can react to output dynamically
- Screen capture capability with `expect_out(buffer)`
- Human-like timing simulation possible

**Cons:**
- Requires expect to be installed
- Initial learning curve
- Timing may need platform-specific adjustment

### 4. Script Command

```bash
# macOS
script -q /dev/null command

# Linux  
script -q -c "command" /dev/null
```

**Pros:**
- Provides a real TTY
- Records all terminal output
- Built into most systems

**Cons:**
- Platform differences in syntax
- Harder to provide input programmatically

### 5. Manual Testing Instructions

Sometimes the best approach is to document manual test steps:

```bash
# Run this manually:
gwt go
# When prompted, press ↓ then Enter
# Verify: pwd shows you're in a different directory
```

## Testing File Descriptors

When testing fd3 output (used for shell integration):

```bash
# Capture fd3 output
command 3>&1 1>&2 | cat

# Save fd3 to a file
command 3>output.txt

# In a shell function
cd_cmd=$(command 3>&1 1>&2)
```

## Debugging Shell Functions

Add debug output to shell functions:

```bash
gwt() {
    # ... function code ...
    >&2 echo "[DEBUG] variable=$variable"
    # ... more code ...
}
```

## Best Practices

1. **Test multiple scenarios:**
   - Direct execution
   - Through shell alias
   - With/without TTY
   - Different input methods

2. **Use debug flags:**
   - Add `--debug` to your application
   - Set debug environment variables
   - Add temporary debug output

3. **Isolate issues:**
   - Test the binary directly vs through alias
   - Test with/without environment variables
   - Test in minimal environments

4. **Document limitations:**
   - Some tests can't be fully automated
   - Platform differences exist
   - Shell behavior varies

## Example: Debugging gwt go

Our investigation used several approaches:

```bash
# 1. Test direct command
GWT_USE_FD3=1 ./git-wt go 3>&1 1>&2

# 2. Test through alias with debug
gwt() {
    # ... 
    >&2 echo "[DEBUG gwt] cd_cmd='$cd_cmd'"
    # ...
}

# 3. Use expect for arrow keys
expect -c 'spawn gwt go; expect "Navigate"; send "\r"; expect eof'

# 4. Manual verification
./simple-test.sh  # Run interactively
```

The key insight was that debugging at multiple levels (application, shell function, and terminal interaction) was necessary to understand the full picture.