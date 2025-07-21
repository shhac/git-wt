const std = @import("std");

/// Run a command and optionally return output
pub fn run(allocator: std.mem.Allocator, argv: []const []const u8) !std.process.Child.Term {
    const result = try std.process.Child.run(.{
        .allocator = allocator,
        .argv = argv,
    });
    defer allocator.free(result.stdout);
    defer allocator.free(result.stderr);
    
    return result.term;
}

/// Run a command and return true if successful
pub fn runSilent(allocator: std.mem.Allocator, argv: []const []const u8) !bool {
    const term = try run(allocator, argv);
    return term.Exited == 0;
}

/// Run a command and print output if successful
pub fn runWithOutput(allocator: std.mem.Allocator, argv: []const []const u8) !bool {
    const result = try std.process.Child.run(.{
        .allocator = allocator,
        .argv = argv,
    });
    defer allocator.free(result.stdout);
    defer allocator.free(result.stderr);
    
    if (result.term.Exited == 0) {
        const stdout = std.io.getStdOut().writer();
        try stdout.writeAll(result.stdout);
        return true;
    }
    return false;
}

test "run command" {
    const allocator = std.testing.allocator;
    
    // Use 'echo' which is more universally available than 'true'
    const term = run(allocator, &.{ "echo", "test" }) catch |err| {
        // Skip test if command not found
        if (err == error.FileNotFound) return;
        return err;
    };
    try std.testing.expectEqual(@as(u8, 0), term.Exited);
}

test "runSilent" {
    const allocator = std.testing.allocator;
    
    // Use 'echo' for success test
    const success = runSilent(allocator, &.{ "echo", "test" }) catch |err| {
        if (err == error.FileNotFound) return;
        return err;
    };
    try std.testing.expect(success);
}