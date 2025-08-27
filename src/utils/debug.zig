
const std = @import("std");
const io = @import("io.zig");
var debug_enabled: bool = false;

pub fn setEnabled(enabled: bool) void {
    debug_enabled = enabled;
}

pub fn isEnabled() bool {
    return debug_enabled;
}

pub fn print(comptime fmt: []const u8, args: anytype) void {
    if (!debug_enabled) return;
    
    const stderr = io.getStdErr();
    stderr.print("[DEBUG] ", .{}) catch return;
    stderr.print(fmt, args) catch return;
    stderr.print("\n", .{}) catch return;
}

pub fn printSection(title: []const u8) void {
    if (!debug_enabled) return;
    
    const stderr = io.getStdErr();
    stderr.print("\n[DEBUG] === {s} ===\n", .{title}) catch return;
}

test "debug module" {
    // Test that debug is disabled by default
    try std.testing.expect(!isEnabled());
    
    // Test enabling debug
    setEnabled(true);
    try std.testing.expect(isEnabled());
    
    // Test disabling debug
    setEnabled(false);
    try std.testing.expect(!isEnabled());
}
