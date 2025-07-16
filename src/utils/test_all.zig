const std = @import("std");

test {
    _ = @import("git.zig");
    _ = @import("fs.zig");
    _ = @import("colors.zig");
    _ = @import("env.zig");
    // Note: input.zig and process.zig are harder to unit test due to I/O
}