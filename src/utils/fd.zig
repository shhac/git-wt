const std = @import("std");

/// Check if fd3 output is enabled via environment variable
pub fn isEnabled() bool {
    if (std.process.getEnvVarOwned(std.heap.page_allocator, "GWT_USE_FD3")) |value| {
        defer std.heap.page_allocator.free(value);
        return std.mem.eql(u8, value, "1");
    } else |_| {
        return false;
    }
}

/// Writer that conditionally uses fd 3 or stdout based on environment variable
pub const CommandWriter = struct {
    use_fd3: bool,
    
    pub fn init() CommandWriter {
        return .{ .use_fd3 = isEnabled() };
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

test "CommandWriter" {
    // Test CommandWriter defaults to stdout when env var not set
    const cmd_writer = CommandWriter.init();
    try std.testing.expect(!cmd_writer.use_fd3);
    
    // Test that print doesn't crash
    try cmd_writer.print("Test output\n", .{});
}