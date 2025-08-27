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