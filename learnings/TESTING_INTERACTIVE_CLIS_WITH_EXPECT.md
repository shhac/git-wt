# Testing Interactive CLIs with Expect

After extensive testing with git-wt's interactive features, we've developed a comprehensive approach for testing interactive CLI applications using expect. This document captures our learnings and best practices.

## Key Discoveries

### 1. Expect Actually Works Well!

Initially, we thought expect had limitations, but it turned out to be the most effective solution:

- **Provides real pseudo-TTY**: Applications detect interactive mode correctly
- **Sends real keyboard sequences**: Including arrow keys, space, ESC
- **Can capture screen state**: Using `expect_out(buffer)` at any point
- **Supports human-like timing**: Adding delays between keystrokes

### 2. Screen Capture Technique

We developed a powerful debugging technique using expect's buffer:

```tcl
proc capture_screen {msg} {
    global expect_out
    puts "\n=== $msg ==="
    puts "=== SCREEN CAPTURE START ==="
    
    if {[info exists expect_out(buffer)]} {
        puts $expect_out(buffer)
    } else {
        puts "(No buffer content available)"
    }
    
    puts "=== SCREEN CAPTURE END ==="
}
```

This allows capturing the exact terminal state at any point, including:
- On timeout (to see where it got stuck)
- After specific actions
- When tests fail

### 3. Human-like Timing

Real users don't type instantly. Adding delays makes tests more realistic:

```tcl
set human_delay 0.025 ;# 25ms between keystrokes

proc send_human {text} {
    global human_delay
    foreach char [split $text ""] {
        send -- $char
        sleep $human_delay
    }
}

proc send_key {key} {
    global human_delay
    send -- $key
    sleep $human_delay
}
```

Also add reaction time before actions:
```tcl
sleep 0.5 ;# Human reaction time before pressing key
send_key "\033\[B" ;# Down arrow
```

### 4. Handling Different UI Modes

Interactive CLIs often have fallback modes. Test both:

```tcl
expect {
    "*Navigate*Select*" {
        # Arrow-key mode detected
        send_key "\033\[B"  ;# Down arrow
    }
    "*Enter number*" {
        # Number-based fallback
        send_human "1"
        send_key "\r"
    }
    timeout {
        capture_screen "UI detection timeout"
        test_failed "No UI appeared"
    }
}
```

## Testing Patterns

### 1. Navigation with Arrow Keys

```tcl
# Down arrow
send "\033\[B"

# Up arrow  
send "\033\[A"

# Right arrow
send "\033\[C"

# Left arrow
send "\033\[D"

# Enter
send "\r"

# ESC
send "\033"

# Space
send " "
```

### 2. Multi-Select Testing

```tcl
# Navigate and select multiple items
send_key "\033\[B" ;# Down to item 1
send_key " "        ;# Space to select
send_key "\033\[B" ;# Down to item 2  
send_key " "        ;# Space to select
send_key "\r"       ;# Enter to confirm
```

### 3. Cleanup Pattern

Always cleanup after expect blocks:

```tcl
expect {
    "pattern" {
        # Handle success
    }
    timeout {
        # Handle timeout
    }
}
catch {close}
catch {wait}
```

### 4. Testing Prunable/Missing States

Create specific conditions then test handling:

```tcl
# Create worktree
system "$git_wt new test-branch --non-interactive"

# Make it prunable by removing directory
system "rm -rf ../trees/test-branch"

# Test that list shows it as prunable
spawn $git_wt list
expect {
    "*missing (prunable)*" {
        puts "✓ Shows prunable status"
    }
}
```

## Complete Test Structure

```tcl
#!/usr/bin/expect -f

set timeout 5
set human_delay 0.025
set test_failed 0

# Helper procedures
proc capture_screen {msg} { ... }
proc send_human {text} { ... }
proc send_key {key} { ... }
proc test_failed {msg} { ... }

# Setup
puts "\n=== Setting up test ==="
# Create test environment

# Test cases
puts "\n=== Test 1: Feature X ==="
spawn command
expect { ... }
catch {close}
catch {wait}

# Cleanup
puts "\n=== Cleanup ==="
# Remove test artifacts

# Results
if {$test_failed == 0} {
    puts "✅ ALL TESTS PASSED"
    exit 0
} else {
    puts "❌ SOME TESTS FAILED"
    exit 1
}
```

## Advantages Over Other Methods

### vs Piping Input
- ✅ Real TTY (interactive mode works)
- ✅ Can send special keys (arrows, ESC)
- ✅ Can react to output dynamically
- ✅ Screen capture for debugging

### vs Manual Testing
- ✅ Automated and repeatable
- ✅ Can run in CI/CD
- ✅ Consistent timing
- ✅ Better coverage

### vs Unit Tests
- ✅ Tests actual user interaction
- ✅ Catches UI/UX issues
- ✅ Tests terminal-specific features
- ✅ Validates the full experience

## Platform Considerations

### macOS
- expect usually installed or via `brew install expect`
- Terminal escape sequences work well
- Use `/usr/bin/expect` or `/opt/homebrew/bin/expect`

### Linux
- Install via package manager (`apt-get install expect`)
- Same escape sequences work
- May need to handle different terminal types

### CI/CD
- Can run headless
- Set TERM environment variable if needed
- Timeout values may need adjustment

## Debugging Tips

1. **Use capture_screen liberally**: Especially on timeout
2. **Add verbose output**: Show what keys are being sent
3. **Start with longer timeouts**: Reduce once stable
4. **Test incrementally**: Build up complex interactions
5. **Save failed output**: Capture screen on test failures

## Limitations

1. **Can't test actual shell integration**: Since expect runs in subprocess
2. **Timing sensitive**: May need platform-specific delays  
3. **Complex ANSI handling**: Some terminal features hard to test
4. **Not visual validation**: Can't verify colors/layout precisely

## Conclusion

Expect is the best tool we've found for testing interactive CLI features. The combination of:
- Real pseudo-TTY
- Keyboard simulation
- Screen capture
- Human-like timing

Makes it possible to thoroughly test interactive features that would otherwise require manual testing. The key insight is that `expect_out(buffer)` gives us visibility into exactly what the user would see, making debugging much easier.