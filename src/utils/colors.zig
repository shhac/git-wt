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