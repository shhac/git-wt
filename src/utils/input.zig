const std = @import("std");

/// Read user confirmation (y/n)
pub fn confirm(prompt: []const u8, default: bool) !bool {
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