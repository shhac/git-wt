const std = @import("std");

/// Wrapper struct for File to add print functionality for Zig 0.15+
pub const FileWriter = struct {
    file: std.fs.File,

    pub fn print(self: FileWriter, comptime fmt: []const u8, args: anytype) !void {
        var buffer: [4096]u8 = undefined;
        const message = try std.fmt.bufPrint(&buffer, fmt, args);
        try self.file.writeAll(message);
    }

    pub fn writeAll(self: FileWriter, bytes: []const u8) !void {
        try self.file.writeAll(bytes);
    }

    /// Flush buffered output to ensure immediate rendering
    /// This is critical for interactive UI updates to prevent flicker and delays
    pub fn flush(self: FileWriter) void {
        // For TTY devices, sync ensures kernel buffers are flushed to the terminal
        // This makes ANSI escape sequences apply immediately and atomically
        // Errors are intentionally ignored as flush is best-effort
        self.file.sync() catch {};
    }
};

/// Get stdout with print functionality
pub fn getStdOut() FileWriter {
    return FileWriter{ .file = std.fs.File.stdout() };
}

/// Get stderr with print functionality  
pub fn getStdErr() FileWriter {
    return FileWriter{ .file = std.fs.File.stderr() };
}

/// Get stdin
pub fn getStdIn() std.fs.File {
    return std.fs.File.stdin();
}

test "FileWriter print functionality" {
    // Test that FileWriter.print works correctly
    const allocator = std.testing.allocator;
    
    // Create a temporary file
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();
    
    const file = try tmp_dir.dir.createFile("test_output.txt", .{ .read = true });
    defer file.close();
    
    const writer = FileWriter{ .file = file };
    
    // Test print with formatting
    try writer.print("Hello {s}, number: {d}\n", .{ "World", 42 });
    
    // Read back and verify
    try file.seekTo(0);
    const content = try file.readToEndAlloc(allocator, 1024);
    defer allocator.free(content);
    
    try std.testing.expectEqualStrings("Hello World, number: 42\n", content);
}

test "FileWriter writeAll functionality" {
    // Test that FileWriter.writeAll works correctly
    const allocator = std.testing.allocator;
    
    // Create a temporary file
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();
    
    const file = try tmp_dir.dir.createFile("test_write.txt", .{ .read = true });
    defer file.close();
    
    const writer = FileWriter{ .file = file };
    
    // Test writeAll
    try writer.writeAll("Direct write test\n");
    
    // Read back and verify
    try file.seekTo(0);
    const content = try file.readToEndAlloc(allocator, 1024);
    defer allocator.free(content);
    
    try std.testing.expectEqualStrings("Direct write test\n", content);
}