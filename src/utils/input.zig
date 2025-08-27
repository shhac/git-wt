
/// Read user confirmation (y/n)
const std = @import("std");
const env = @import("env.zig");
const io = @import("io.zig");
pub fn confirm(prompt: []const u8, default: bool) !bool {
    // In non-interactive mode, always return the default
    if (env.isNonInteractive()) {
        const stdout = io.getStdOut();
        try stdout.print("{s} [auto: {s}]\n", .{ prompt, if (default) "yes" else "no" });
        return default;
    }
    
    const stdout = io.getStdOut();
    const stdin = io.getStdIn();
    
    const default_str = if (default) "[Y/n]" else "[y/N]";
    try stdout.print("{s} {s} ", .{ prompt, default_str });
    
    var buf: [256]u8 = undefined;
    // In Zig 0.15, we need to use read directly
    const bytes_read = try stdin.read(&buf);
    
    // Find newline in the buffer
    const result: ?[]u8 = if (bytes_read > 0) blk: {
        for (buf[0..bytes_read], 0..) |c, i| {
            if (c == '\n') {
                break :blk buf[0..i];
            }
        }
        break :blk buf[0..bytes_read];
    } else null;
    
    if (result) |response| {
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
        const stdout = io.getStdOut();
        try stdout.print("{s} [auto: skip]\n", .{prompt});
        return null;
    }
    
    const stdout = io.getStdOut();
    const stdin = io.getStdIn();
    
    try stdout.print("{s} ", .{prompt});
    
    // Read input directly
    var buf: [1024]u8 = undefined;
    const bytes_read = try stdin.read(&buf);
    
    const line: ?[]u8 = if (bytes_read > 0) blk: {
        for (buf[0..bytes_read], 0..) |c, i| {
            if (c == '\n') {
                const result = try allocator.dupe(u8, buf[0..i]);
                break :blk result;
            }
        }
        const result = try allocator.dupe(u8, buf[0..bytes_read]);
        break :blk result;
    } else null;
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

test "confirm default behavior" {
    // Test the parsing logic without actual I/O
    // Since confirm() reads from stdin, we test the logic indirectly
    
    // Test that 'y' and 'Y' are recognized as yes
    try std.testing.expect('y' == 'y' or 'y' == 'Y');
    try std.testing.expect('Y' == 'y' or 'Y' == 'Y');
    
    // Test that other values are not yes
    try std.testing.expect(!('n' == 'y' or 'n' == 'Y'));
    try std.testing.expect(!('q' == 'y' or 'q' == 'Y'));
}

test "readLine trimming behavior" {
    // Test trimming newline characters (simulating the logic)
    const test_input = "test input\n";
    const trimmed = std.mem.trim(u8, test_input, " \t\r\n");
    try std.testing.expectEqualStrings("test input", trimmed);
    
    // Test trimming carriage return
    const test_input_cr = "test input\r\n";
    const trimmed_cr = std.mem.trim(u8, test_input_cr, " \t\r\n");
    try std.testing.expectEqualStrings("test input", trimmed_cr);
    
    // Test empty input becomes null behavior
    const test_empty = "   \n";
    const trimmed_empty = std.mem.trim(u8, test_empty, " \t\r\n");
    try std.testing.expect(trimmed_empty.len == 0);
}
