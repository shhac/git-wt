const fd = @import("fd.zig");
const debug = @import("debug.zig");

/// Operating mode for git-wt commands.
///
/// - `wrapper`: Running under the shell alias with GWT_FD set.
///   Navigation commands write `cd` commands to the configured fd
///   for the shell wrapper to eval.
///
/// - `bare`: Running as a standalone binary without GWT_FD.
///   Navigation commands output worktree paths to stdout,
///   suitable for scripting and piping.
pub const Mode = enum {
    wrapper,
    bare,

    /// Returns true when running under the shell alias wrapper.
    pub fn isWrapper(self: Mode) bool {
        return self == .wrapper;
    }

    /// Returns true when running as a standalone binary.
    pub fn isBare(self: Mode) bool {
        return self == .bare;
    }
};

/// Detect the current operating mode from the environment.
/// Call once at startup and pass the result to commands.
pub fn detect() Mode {
    const m: Mode = if (fd.isEnabled()) .wrapper else .bare;
    if (debug.isEnabled()) {
        debug.print("Mode: {s}", .{@tagName(m)});
    }
    return m;
}

test "detect returns bare when GWT_FD not set" {
    // In test environment, GWT_FD is not set
    const m = detect();
    try @import("std").testing.expectEqual(Mode.bare, m);
}

test "Mode convenience methods" {
    const std = @import("std");
    try std.testing.expect(Mode.wrapper.isWrapper());
    try std.testing.expect(!Mode.wrapper.isBare());
    try std.testing.expect(Mode.bare.isBare());
    try std.testing.expect(!Mode.bare.isWrapper());
}
