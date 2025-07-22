const std = @import("std");
const testing = std.testing;
const new = @import("new.zig");
const git = @import("../utils/git.zig");
const fs = @import("../utils/fs.zig");
const validation = @import("../utils/validation.zig");

test "new command validation" {
    // Test invalid branch names
    const invalid_names = [_][]const u8{
        "feature branch", // spaces
        "feature~branch", // tilde
        "feature^branch", // caret
        "feature:branch", // colon
        "feature?branch", // question mark
        "feature*branch", // asterisk
        "feature[branch", // bracket
        "-branch", // starts with dash
        "HEAD", // reserved name
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
    
    // Test valid branch names
    const valid_names = [_][]const u8{
        "feature",
        "feature-123",
        "feature_123",
        "feature/auth",
        "bugfix/issue-123",
        "release/v1.0.0",
        "user/john/feature",
    };
    
    for (valid_names) |name| {
        validation.validateBranchName(name) catch |err| {
            std.debug.print("Unexpected error for valid name '{s}': {}\n", .{ name, err });
            return err;
        };
    }
}

test "worktree path generation" {
    const allocator = testing.allocator;
    
    // Test branch name sanitization  
    const test_cases = [_]struct {
        branch: []const u8,
        expected: []const u8,
    }{
        .{ .branch = "feature", .expected = "feature" },
        .{ .branch = "feature:auth", .expected = "feature%3Aauth" },
        .{ .branch = "feature?auth", .expected = "feature%3Fauth" },
        .{ .branch = "feature*auth", .expected = "feature%2Aauth" },
    };
    
    for (test_cases) |tc| {
        const sanitized = try fs.sanitizeBranchPath(allocator, tc.branch);
        defer allocator.free(sanitized);
        try testing.expectEqualStrings(tc.expected, sanitized);
    }
}

test "path existence check" {
    const allocator = testing.allocator;
    
    // Create a temporary directory for testing
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();
    
    const tmp_path = try tmp_dir.dir.realpathAlloc(allocator, ".");
    defer allocator.free(tmp_path);
    
    // Test path existence
    try testing.expect(fs.pathExists(tmp_path));
    
    // Test non-existent path
    const fake_path = try std.fmt.allocPrint(allocator, "{s}/non-existent", .{tmp_path});
    defer allocator.free(fake_path);
    try testing.expect(!fs.pathExists(fake_path));
}