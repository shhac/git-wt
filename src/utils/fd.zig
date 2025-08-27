const std = @import("std");
const io = @import("io.zig");

/// File descriptor 3 (fd3) mechanism for shell integration
/// 
/// This module provides a clean way for git-wt commands to communicate with
/// shell wrapper functions without interfering with normal stdout/stderr output.
/// 
/// How it works:
/// 1. Shell aliases set GWT_USE_FD3=1 and redirect fd3 to capture commands
/// 2. When enabled, commands like 'go' write shell commands to fd3
/// 3. The shell wrapper evaluates captured commands (e.g., 'cd /path')
/// 4. This allows changing the shell's working directory from a subprocess
/// 
/// Example shell alias:
/// ```bash
/// gwt() {
///     if [ "$1" = "go" ]; then
///         local cd_cmd=$(GWT_USE_FD3=1 git-wt go "$@" 3>&1 1>&2)
///         if [ -n "$cd_cmd" ]; then
///             eval "$cd_cmd"
///         fi
///     else
///         git-wt "$@"
///     fi
/// }
/// ```

/// Check if fd3 output is enabled via environment variable
pub fn isEnabled() bool {
    if (std.process.getEnvVarOwned(std.heap.page_allocator, "GWT_USE_FD3")) |value| {
        defer std.heap.page_allocator.free(value);
        const enabled = std.mem.eql(u8, value, "1");
        if (@import("../utils/debug.zig").isEnabled()) {
            std.debug.print("[DEBUG] fd3: GWT_USE_FD3='{s}', enabled={}\n", .{ value, enabled });
        }
        return enabled;
    } else |_| {
        if (@import("../utils/debug.zig").isEnabled()) {
            std.debug.print("[DEBUG] fd3: GWT_USE_FD3 not set\n", .{});
        }
        return false;
    }
}

/// Writer that conditionally uses fd 3 or stdout based on environment variable
pub const CommandWriter = struct {
    use_fd3: bool,
    
    pub fn init() CommandWriter {
        return .{ .use_fd3 = isEnabled() };
    }
    
    pub fn writer(self: CommandWriter) io.FileWriter {
        if (self.use_fd3) {
            // File descriptor 3 should be provided by the shell wrapper
            // If not available, this will fail when trying to write
            const file = std.fs.File{ .handle = 3 };
            return io.FileWriter{ .file = file };
        }
        return io.getStdOut();
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