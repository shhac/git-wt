const std = @import("std");
const env = @import("env.zig");

/// Read user confirmation (y/n)
pub fn confirm(prompt: []const u8, default: bool) !bool {
    // In non-interactive mode, always return the default
    if (env.isNonInteractive()) {
        const stdout = std.io.getStdOut().writer();
        try stdout.print("{s} [auto: {s}]\n", .{ prompt, if (default) "yes" else "no" });
        return default;
    }
    
    const stdout = std.io.getStdOut().writer();
    const stdin = std.io.getStdIn().reader();
    
    const default_str = if (default) "[Y/n]" else "[y/N]";
    try stdout.print("{s} {s} ", .{ prompt, default_str });
    
    var buf: [10]u8 = undefined;
    if (try stdin.readUntilDelimiterOrEof(&buf, '\n')) |response| {
        const trimmed = std.mem.trim(u8, response, " \t\r\n");
        if (trimmed.len == 0) return default;
        return trimmed[0] == 'y' or trimmed[0] == 'Y';
    }
    return default;
}

/// Read user input string
pub fn readLine(allocator: std.mem.Allocator, prompt: []const u8) !?[]u8 {
    // In non-interactive mode, return null (empty input)
    if (env.isNonInteractive()) {
        const stdout = std.io.getStdOut().writer();
        try stdout.print("{s} [auto: skip]\n", .{prompt});
        return null;
    }
    
    const stdout = std.io.getStdOut().writer();
    const stdin = std.io.getStdIn().reader();
    
    try stdout.print("{s} ", .{prompt});
    
    const line = try stdin.readUntilDelimiterOrEofAlloc(allocator, '\n', 1024);
    if (line) |l| {
        const trimmed = std.mem.trim(u8, l, " \t\r\n");
        if (trimmed.len == 0) {
            allocator.free(l);
            return null;
        }
        if (trimmed.ptr != l.ptr) {
            const result = try allocator.dupe(u8, trimmed);
            allocator.free(l);
            return result;
        }
        return l;
    }
    return null;
}