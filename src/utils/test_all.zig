const std = @import("std");

test {
    _ = @import("git.zig");
    _ = @import("fs.zig");
    _ = @import("colors.zig");
    _ = @import("env.zig");
    _ = @import("validation.zig");
    _ = @import("process.zig");
    _ = @import("debug.zig");
    _ = @import("fd.zig");
    _ = @import("time.zig");
    // Note: input.zig and interactive.zig are harder to unit test due to I/O
    // Note: Commands import interactive modules which can interfere with tests
}