const std = @import("std");
const io = @import("io.zig");

/// Configurable file descriptor mechanism for shell integration
///
/// This module provides a clean way for git-wt commands to communicate with
/// shell wrapper functions without interfering with normal stdout/stderr output.
///
/// How it works:
/// 1. Shell aliases set GWT_FD=N and redirect fd N to capture commands
/// 2. When enabled, commands like 'go' write shell commands to fd N
/// 3. The shell wrapper evaluates captured commands (e.g., 'cd /path')
/// 4. This allows changing the shell's working directory from a subprocess
///
/// The fd number defaults to 3 but can be configured via the alias --fd flag.
///
/// Example shell alias:
/// ```bash
/// gwt() {
///     if [ "$1" = "go" ]; then
///         local cd_cmd=$(GWT_FD=3 git-wt go "$@" 3>&1 1>&2)
///         if [ -n "$cd_cmd" ]; then
///             eval "$cd_cmd"
///         fi
///     else
///         git-wt "$@"
///     fi
/// }
/// ```

/// Get the configured fd number from environment, or null if not enabled.
/// Checks GWT_FD env var which contains the fd number (e.g., "3", "5").
pub fn getFdNumber() ?std.posix.fd_t {
    const dbg = @import("../utils/debug.zig");
    if (std.process.getEnvVarOwned(std.heap.page_allocator, "GWT_FD")) |value| {
        defer std.heap.page_allocator.free(value);
        const fd_num = std.fmt.parseInt(std.posix.fd_t, value, 10) catch {
            if (dbg.isEnabled()) {
                std.debug.print("[DEBUG] fd: GWT_FD='{s}' (invalid number, ignoring)\n", .{value});
            }
            return null;
        };
        if (fd_num < 3 or fd_num > 9) {
            if (dbg.isEnabled()) {
                std.debug.print("[DEBUG] fd: GWT_FD={} (out of range 3-9, ignoring)\n", .{fd_num});
            }
            return null;
        }
        if (dbg.isEnabled()) {
            std.debug.print("[DEBUG] fd: GWT_FD={}\n", .{fd_num});
        }
        return fd_num;
    } else |_| {
        if (dbg.isEnabled()) {
            std.debug.print("[DEBUG] fd: GWT_FD not set\n", .{});
        }
        return null;
    }
}

/// Check if fd output is enabled via environment variable
pub fn isEnabled() bool {
    return getFdNumber() != null;
}

/// Writer that conditionally uses a configured fd or stdout based on environment variable
pub const CommandWriter = struct {
    fd_number: ?std.posix.fd_t,

    pub fn init() CommandWriter {
        return .{ .fd_number = getFdNumber() };
    }

    pub fn writer(self: CommandWriter) io.FileWriter {
        if (self.fd_number) |fd_num| {
            const file = std.fs.File{ .handle = fd_num };
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
    try std.testing.expect(cmd_writer.fd_number == null);

    // Test that print doesn't crash
    try cmd_writer.print("Test output\n", .{});
}

test "CommandWriter with explicit fd" {
    // Test CommandWriter with explicit fd number
    const cmd_writer = CommandWriter{ .fd_number = 5 };
    const w = cmd_writer.writer();
    _ = w; // Just verify it compiles

    // Test disabled writer
    const disabled = CommandWriter{ .fd_number = null };
    const w2 = disabled.writer();
    _ = w2;
}
