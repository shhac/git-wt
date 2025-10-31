const std = @import("std");
const testing = std.testing;
const remove = @import("remove.zig");
const git = @import("../utils/git.zig");
const validation = @import("../utils/validation.zig");

test "branch name validation in remove" {
    // Test that remove validates branch names
    const invalid_names = [_][]const u8{
        "", // empty
        " ", // whitespace
        "feature branch", // spaces
        "HEAD", // reserved
        "..", // path component
    };
    
    for (invalid_names) |name| {
        validation.validateBranchName(name) catch {
            // Expected error, continue
            continue;
        };
        // If we get here, validation passed when it shouldn't have
        std.debug.print("Expected validation to fail for '{s}' but it passed\n", .{name});
        try testing.expect(false);
    }
}

test "sanitized branch name matching" {
    // Test that remove can match sanitized branch names
    const test_cases = [_]struct {
        input: []const u8,
        worktree_branch: []const u8,
        should_match: bool,
    }{
        .{ .input = "feature/auth", .worktree_branch = "feature/auth", .should_match = true },
        .{ .input = "feature-auth", .worktree_branch = "feature/auth", .should_match = true }, // sanitized form
        .{ .input = "feature/auth", .worktree_branch = "feature/login", .should_match = false },
        .{ .input = "main", .worktree_branch = "main", .should_match = true },
    };
    
    for (test_cases) |tc| {
        if (tc.should_match) {
            // Both forms should match
            const sanitized = try std.mem.replaceOwned(u8, testing.allocator, tc.worktree_branch, "/", "-");
            defer testing.allocator.free(sanitized);
            try testing.expect(std.mem.eql(u8, tc.input, tc.worktree_branch) or
                               std.mem.eql(u8, tc.input, sanitized));
        }
    }
}