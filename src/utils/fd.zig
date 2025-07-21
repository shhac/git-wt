const std = @import("std");

/// Write to a specific file descriptor
pub fn writeToFd(fd_num: std.fs.File.Handle, data: []const u8) !void {
    const file = std.fs.File{ .handle = fd_num };
    const writer = file.writer();
    try writer.writeAll(data);
}

/// Check if a file descriptor is available for writing
pub fn isFdAvailable(fd_num: std.fs.File.Handle) bool {
    // For now, only consider standard fds as available
    // fd 3 requires special shell setup that we can't reliably detect
    return fd_num == 1 or fd_num == 2;
}

/// Get a writer for command output (fd 3 if available, stdout otherwise)
pub fn getCommandWriter() std.fs.File.Writer {
    const command_fd: std.fs.File.Handle = 3;
    
    // Check if fd 3 is available
    if (isFdAvailable(command_fd)) {
        const file = std.fs.File{ .handle = command_fd };
        return file.writer();
    }
    
    // Fall back to stdout
    return std.io.getStdOut().writer();
}

/// Writer that conditionally uses fd 3 or stdout
pub const CommandWriter = struct {
    use_fd3: bool,
    
    pub fn init() CommandWriter {
        // Check if GWT_USE_FD3 environment variable is set
        const use_fd3 = if (std.process.getEnvVarOwned(std.heap.page_allocator, "GWT_USE_FD3")) |value| blk: {
            defer std.heap.page_allocator.free(value);
            break :blk std.mem.eql(u8, value, "1");
        } else |_| false;
        
        return .{ .use_fd3 = use_fd3 };
    }
    
    pub fn writer(self: CommandWriter) std.fs.File.Writer {
        if (self.use_fd3) {
            const file = std.fs.File{ .handle = 3 };
            return file.writer();
        }
        return std.io.getStdOut().writer();
    }
    
    pub fn print(self: CommandWriter, comptime fmt: []const u8, args: anytype) !void {
        try self.writer().print(fmt, args);
    }
};

test "fd utilities" {
    // Test writing to stdout (fd 1)
    try writeToFd(1, "Hello from fd 1\n");
    
    // Test checking fd availability
    try std.testing.expect(isFdAvailable(1)); // stdout should be available
    try std.testing.expect(isFdAvailable(2)); // stderr should be available
    
    // fd 3 might not be available in test environment
    _ = isFdAvailable(3);
    
    // Test CommandWriter
    const cmd_writer = CommandWriter.init();
    try cmd_writer.print("Test output\n", .{});
}