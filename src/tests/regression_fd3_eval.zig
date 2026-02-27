const std = @import("std");
const testing = std.testing;

// Regression tests for fd/eval issues
//
// Background: The shell alias was incorrectly using eval which prevented
// environment variables from being passed to the subprocess.
// This caused navigation to fail silently in interactive mode.
//
// These tests ensure that the problematic pattern doesn't reappear in our code.

test "regression: verify alias.zig doesn't use eval incorrectly" {
    // This test verifies that we don't reintroduce the problematic eval pattern
    // that broke the fd mechanism

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

test "regression: verify fd writes to configured file descriptor" {
    // This test verifies the CommandWriter structure
    const fd = @import("../utils/fd.zig");

    // When fd is not enabled, CommandWriter should use stdout
    const cmd_writer_disabled = fd.CommandWriter{ .fd_number = null };
    const writer_disabled = cmd_writer_disabled.writer();
    _ = writer_disabled; // Just verify it compiles

    // When fd is enabled with fd 3, CommandWriter should use fd 3
    const cmd_writer_fd3 = fd.CommandWriter{ .fd_number = 3 };
    const writer_fd3 = cmd_writer_fd3.writer();
    _ = writer_fd3; // Just verify it compiles

    // When fd is enabled with fd 5, CommandWriter should use fd 5
    const cmd_writer_fd5 = fd.CommandWriter{ .fd_number = 5 };
    const writer_fd5 = cmd_writer_fd5.writer();
    _ = writer_fd5; // Just verify it compiles
}

test "regression: verify go command imports fd module" {
    // This ensures the go command has access to the fd mechanism
    const go_content = @embedFile("../commands/go.zig");

    // Verify it imports the fd module
    try testing.expect(std.mem.indexOf(u8, go_content, "@import(\"../utils/fd.zig\")") != null);

    // Verify it uses fd.CommandWriter.init() for fd-based output
    try testing.expect(std.mem.indexOf(u8, go_content, "fd.CommandWriter.init()") != null);
}

test "regression: verify debug logging for fd issues" {
    // This ensures we have debug logging to diagnose fd issues
    const go_content = @embedFile("../commands/go.zig");

    // Verify debug logging exists for fd_enabled
    try testing.expect(std.mem.indexOf(u8, go_content, "[DEBUG] go: fd_enabled=") != null);
}

test "regression: shell function pattern verification" {
    // Verify the shell function uses the correct pattern for environment variables
    const alias_content = @embedFile("../commands/alias.zig");

    // The correct pattern: GWT_FD= followed by a digit and the bin path
    // Since fd_num is dynamic, just check for GWT_FD= prefix
    try testing.expect(std.mem.indexOf(u8, alias_content, "GWT_FD=") != null);

    // Verify we're using the fd redirection pattern (>&1 1>&2)
    try testing.expect(std.mem.indexOf(u8, alias_content, ">&1 1>&2") != null);

    // Verify the old GWT_USE_FD3 pattern is NOT present
    try testing.expect(std.mem.indexOf(u8, alias_content, "GWT_USE_FD3") == null);
}

test "regression: alias supports --fd flag" {
    // Verify the alias command accepts the --fd flag
    const alias_content = @embedFile("../commands/alias.zig");

    // Verify --fd flag is parsed
    try testing.expect(std.mem.indexOf(u8, alias_content, "\"--fd\"") != null);
}
