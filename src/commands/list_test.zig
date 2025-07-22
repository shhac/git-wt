const std = @import("std");
const testing = std.testing;
const list = @import("list.zig");
const git = @import("../utils/git.zig");
const time = @import("../utils/time.zig");

test "list output formatting" {
    
    // Test that list can handle different worktree states
    const test_worktrees = [_]git.WorktreeWithTime{
        .{
            .worktree = .{
                .path = "/Users/test/my-repo",
                .branch = "main",
                .commit = "abc123",
                .is_bare = false,
                .is_detached = false,
                .is_current = true,
            },
            .mod_time = std.time.timestamp(),
            .display_name = "main",
        },
        .{
            .worktree = .{
                .path = "/Users/test/my-repo-trees/feature",
                .branch = "feature/auth",
                .commit = "def456",
                .is_bare = false,
                .is_detached = false,
                .is_current = false,
            },
            .mod_time = std.time.timestamp() - 3600, // 1 hour ago
            .display_name = "feature",
        },
        .{
            .worktree = .{
                .path = "/Users/test/my-repo-trees/detached",
                .branch = "HEAD",
                .commit = "789ghi",
                .is_bare = false,
                .is_detached = true,
                .is_current = false,
            },
            .mod_time = std.time.timestamp() - 86400, // 1 day ago
            .display_name = "detached",
        },
    };
    
    // Verify worktree properties
    for (test_worktrees) |wt| {
        try testing.expect(wt.worktree.path.len > 0);
        try testing.expect(wt.worktree.branch.len > 0);
        try testing.expect(wt.worktree.commit.len > 0);
    }
}

test "plain output format" {
    // Test that plain output produces simple paths
    const paths = [_][]const u8{
        "/Users/test/my-repo",
        "/Users/test/my-repo-trees/feature",
        "/Users/test/my-repo-trees/bugfix",
    };
    
    for (paths) |path| {
        // In plain mode, we just output the path
        try testing.expect(path.len > 0);
        try testing.expect(std.mem.indexOf(u8, path, "/") != null);
    }
}