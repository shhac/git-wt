const std = @import("std");

/// Check if we're running in non-interactive mode
pub fn isNonInteractive() bool {
    if (std.process.getEnvVarOwned(std.heap.page_allocator, "GIT_WT_NON_INTERACTIVE")) |value| {
        defer std.heap.page_allocator.free(value);
        return std.mem.eql(u8, value, "1");
    } else |_| {
        return false;
    }
}

/// Check if NO_COLOR is set (for respecting color preferences)
pub fn isNoColor() bool {
    if (std.process.getEnvVarOwned(std.heap.page_allocator, "NO_COLOR")) |value| {
        std.heap.page_allocator.free(value);
        return true;
    } else |_| {
        return false;
    }
}

test "isNonInteractive" {
    // Note: Can't easily test environment variables in Zig tests
    // as setenv/unsetenv are not available. Just test current state.
    _ = isNonInteractive();
}

test "isNoColor" {
    // Note: Can't easily test environment variables in Zig tests
    // as setenv/unsetenv are not available. Just test current state.
    _ = isNoColor();
}