const std = @import("std");

test {
    _ = @import("git.zig");
    _ = @import("fs.zig");
    _ = @import("colors.zig");
    _ = @import("env.zig");
    _ = @import("validation.zig");
    _ = @import("process.zig");
    _ = @import("debug.zig");
    // Note: input.zig is harder to unit test due to I/O
    
    // Commands
    _ = @import("../commands/go.zig");
}