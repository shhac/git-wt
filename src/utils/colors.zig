const std = @import("std");

// ANSI escape codes
pub const reset = "\x1b[0m";
pub const bold = "\x1b[1m";
pub const red = "\x1b[31m";
pub const green = "\x1b[32m";
pub const yellow = "\x1b[33m";
pub const blue = "\x1b[34m";
pub const magenta = "\x1b[35m";
pub const cyan = "\x1b[36m";

// Common color combinations
pub const error_prefix = bold ++ red;
pub const success_prefix = bold ++ green;
pub const info_prefix = bold ++ blue;
pub const warning_prefix = bold ++ yellow;
pub const path_color = cyan;

// Formatted print helpers
pub fn printError(writer: anytype, comptime fmt: []const u8, args: anytype) !void {
    try writer.print("{s}Error:{s} ", .{ error_prefix, reset });
    try writer.print(fmt, args);
    try writer.print("\n", .{});
}

pub fn printSuccess(writer: anytype, comptime fmt: []const u8, args: anytype) !void {
    try writer.print("{s}✓ ", .{success_prefix});
    try writer.print(fmt, args);
    try writer.print("{s}\n", .{reset});
}

pub fn printInfo(writer: anytype, comptime fmt: []const u8, args: anytype) !void {
    try writer.print("{s}", .{info_prefix});
    try writer.print(fmt, args);
    try writer.print("{s}\n", .{reset});
}

pub fn printPath(writer: anytype, prefix: []const u8, path: []const u8) !void {
    try writer.print("{s} {s}{s}{s}\n", .{ prefix, path_color, path, reset });
}

/// Print a path with user-friendly display formatting
/// Uses the display path (relative) instead of absolute path for better UX
pub fn printDisplayPath(writer: anytype, prefix: []const u8, path: []const u8, allocator: std.mem.Allocator) !void {
    const fs_utils = @import("fs.zig");
    const display_path = try fs_utils.extractDisplayPath(allocator, path);
    defer allocator.free(display_path);
    try writer.print("{s} {s}{s}{s}\n", .{ prefix, path_color, display_path, reset });
}

test "color constants" {
    // Just verify the constants are defined correctly
    try std.testing.expect(reset.len > 0);
    try std.testing.expect(red.len > 0);
    try std.testing.expect(green.len > 0);
    try std.testing.expectEqualStrings("\x1b[0m", reset);
    try std.testing.expectEqualStrings("\x1b[31m", red);
}

test "color print functions" {
    // Test that the print functions work with a buffer
    var buffer = std.ArrayList(u8).init(std.testing.allocator);
    defer buffer.deinit();
    const writer = buffer.writer();
    
    // Test printError
    try printError(writer, "test error", .{});
    try std.testing.expect(std.mem.indexOf(u8, buffer.items, "Error:") != null);
    try std.testing.expect(std.mem.indexOf(u8, buffer.items, "test error") != null);
    
    // Test printSuccess
    buffer.clearRetainingCapacity();
    try printSuccess(writer, "test success", .{});
    try std.testing.expect(std.mem.indexOf(u8, buffer.items, "✓") != null);
    try std.testing.expect(std.mem.indexOf(u8, buffer.items, "test success") != null);
    
    // Test printInfo
    buffer.clearRetainingCapacity();
    try printInfo(writer, "test info", .{});
    try std.testing.expect(std.mem.indexOf(u8, buffer.items, "test info") != null);
    
    // Test printPath
    buffer.clearRetainingCapacity();
    try printPath(writer, "Path:", "/test/path");
    try std.testing.expect(std.mem.indexOf(u8, buffer.items, "Path:") != null);
    try std.testing.expect(std.mem.indexOf(u8, buffer.items, "/test/path") != null);
}