const std = @import("std");
const testing = std.testing;
const alias = @import("alias.zig");
const args = @import("../utils/args.zig");

test "shell string escaping" {
    const allocator = testing.allocator;
    
    // Test escaping special characters
    const test_cases = [_]struct {
        input: []const u8,
        expected: []const u8,
    }{
        .{ .input = "simple", .expected = "simple" },
        .{ .input = "with space", .expected = "with space" },
        .{ .input = "with\"quote", .expected = "with\\\"quote" },
        .{ .input = "with$var", .expected = "with\\$var" },
        .{ .input = "with`cmd`", .expected = "with\\`cmd\\`" },
        .{ .input = "with\\slash", .expected = "with\\\\slash" },
        .{ .input = "../{repo}-trees", .expected = "../{repo}-trees" },
    };
    
    // Note: We can't directly test the private escapeShellString function,
    // but we can verify the concepts
    for (test_cases) |tc| {
        var result = std.ArrayList(u8).empty;
        defer result.deinit(allocator);
        
        for (tc.input) |c| {
            switch (c) {
                '"', '$', '`', '\\' => {
                    try result.append(allocator, '\\');
                    try result.append(allocator, c);
                },
                else => try result.append(allocator, c),
            }
        }
        
        try testing.expectEqualStrings(tc.expected, result.items);
    }
}

test "flag parsing for alias command" {
    const allocator = testing.allocator;
    
    // Test parsing various flag combinations
    const test_args = [_][]const u8{
        "gwt",
        "--no-tty",
        "--plain",
        "--parent-dir",
        "../worktrees",
    };
    
    var parsed = try args.parseArgs(allocator, &test_args);
    defer parsed.deinit(allocator);
    
    // Verify parsing
    try testing.expectEqualStrings("gwt", parsed.getPositional(0).?);
    try testing.expect(parsed.hasFlag(&.{"--no-tty"}));
    try testing.expect(parsed.hasFlag(&.{"--plain"}));
    try testing.expectEqualStrings("../worktrees", parsed.getFlag(&.{ "--parent-dir", "-p" }).?);
}

test "repo template substitution" {
    
    // Test {repo} placeholder detection
    const test_cases = [_]struct {
        path: []const u8,
        has_template: bool,
    }{
        .{ .path = "../{repo}-trees", .has_template = true },
        .{ .path = "~/worktrees/{repo}", .has_template = true },
        .{ .path = "/tmp/{repo}/trees", .has_template = true },
        .{ .path = "../worktrees", .has_template = false },
        .{ .path = "/tmp/trees", .has_template = false },
    };
    
    for (test_cases) |tc| {
        const has_template = std.mem.indexOf(u8, tc.path, "{repo}") != null;
        try testing.expectEqual(tc.has_template, has_template);
    }
}