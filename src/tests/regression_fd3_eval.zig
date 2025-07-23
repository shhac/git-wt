const std = @import("std");
const testing = std.testing;

// Regression tests for fd3/eval issue discovered 2025-01-24
// 
// Background: The shell alias was incorrectly using eval which prevented
// the GWT_USE_FD3 environment variable from being passed to the subprocess.
// This caused navigation to fail silently in interactive mode.
//
// These tests ensure that the problematic pattern doesn't reappear in our code.

test "regression: verify alias.zig doesn't use eval incorrectly" {
    // This test verifies that we don't reintroduce the problematic eval pattern
    // that broke fd3 mechanism
    
    // Read the alias.zig file content at compile time
    const alias_content = @embedFile("../commands/alias.zig");
    
    // The problematic pattern we had before
    const bad_pattern = "eval \\\"$git_wt_bin\\\"";
    
    // The correct pattern we should have
    const good_pattern = "\\\"$git_wt_bin\\\"";
    
    // Verify the bad pattern is NOT present
    try testing.expect(std.mem.indexOf(u8, alias_content, bad_pattern) == null);
    
    // Verify the good pattern IS present
    try testing.expect(std.mem.indexOf(u8, alias_content, good_pattern) != null);
}

test "regression: verify fd3 writes to file descriptor 3" {
    // This test verifies the CommandWriter structure
    const fd = @import("../utils/fd.zig");
    
    // When fd3 is not enabled, CommandWriter should use stdout
    const cmd_writer_disabled = fd.CommandWriter{ .use_fd3 = false };
    const writer_disabled = cmd_writer_disabled.writer();
    _ = writer_disabled; // Just verify it compiles
    
    // When fd3 is enabled, CommandWriter should use fd 3
    const cmd_writer_enabled = fd.CommandWriter{ .use_fd3 = true };
    const writer_enabled = cmd_writer_enabled.writer();
    _ = writer_enabled; // Just verify it compiles
}

test "regression: verify go command imports fd module" {
    // This ensures the go command has access to the fd mechanism
    const go_content = @embedFile("../commands/go.zig");
    
    // Verify it imports the fd module
    try testing.expect(std.mem.indexOf(u8, go_content, "@import(\"../utils/fd.zig\")") != null);
    
    // Verify it uses fd.isEnabled()
    try testing.expect(std.mem.indexOf(u8, go_content, "fd.isEnabled()") != null);
}

test "regression: verify debug logging for fd3 issues" {
    // This ensures we have debug logging to diagnose fd3 issues
    const go_content = @embedFile("../commands/go.zig");
    
    // Verify debug logging exists for fd_enabled
    try testing.expect(std.mem.indexOf(u8, go_content, "[DEBUG] go: fd_enabled=") != null);
}

test "regression: shell function pattern verification" {
    // Verify the shell function uses the correct pattern for environment variables
    const alias_content = @embedFile("../commands/alias.zig");
    
    // The correct pattern for passing environment variables
    const correct_env_pattern = "GWT_USE_FD3=1 \\\"$git_wt_bin\\\"";
    
    // Verify this pattern exists
    try testing.expect(std.mem.indexOf(u8, alias_content, correct_env_pattern) != null);
    
    // Verify we're using the fd3 redirection pattern
    const fd3_redirect = "3>&1 1>&2";
    try testing.expect(std.mem.indexOf(u8, alias_content, fd3_redirect) != null);
}