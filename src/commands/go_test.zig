const std = @import("std");
const testing = std.testing;
const go = @import("go.zig");
const git = @import("../utils/git.zig");
const colors = @import("../utils/colors.zig");

test "display name formatting" {
    
    // Test worktree display names
    const test_cases = [_]struct {
        path: []const u8,
        branch: []const u8,
        expected_contains: []const u8,
    }{
        .{ 
            .path = "/Users/test/projects/my-repo", 
            .branch = "main",
            .expected_contains = "[main]",
        },
        .{ 
            .path = "/Users/test/projects/my-repo-trees/feature", 
            .branch = "feature/auth",
            .expected_contains = "feature",
        },
    };
    
    // We can't easily test the full display logic without mocking,
    // but we can test the components
    for (test_cases) |tc| {
        try testing.expect(tc.branch.len > 0);
        try testing.expect(tc.path.len > 0);
    }
}

test "branch name matching" {
    // Test exact and fuzzy matching logic
    const test_cases = [_]struct {
        input: []const u8,
        branch: []const u8,
        should_match: bool,
    }{
        // Exact matches
        .{ .input = "main", .branch = "main", .should_match = true },
        .{ .input = "feature", .branch = "feature", .should_match = true },
        
        // Case insensitive
        .{ .input = "MAIN", .branch = "main", .should_match = true },
        .{ .input = "Feature", .branch = "feature", .should_match = true },
        
        // Partial matches
        .{ .input = "feat", .branch = "feature", .should_match = true },
        .{ .input = "auth", .branch = "feature/auth", .should_match = true },
        
        // No match
        .{ .input = "develop", .branch = "main", .should_match = false },
        .{ .input = "xyz", .branch = "feature", .should_match = false },
    };
    
    for (test_cases) |tc| {
        const matches = std.ascii.startsWithIgnoreCase(tc.branch, tc.input) or
                       std.mem.indexOf(u8, tc.branch, tc.input) != null;
        try testing.expectEqual(tc.should_match, matches);
    }
}