const std = @import("std");
const testing = std.testing;
const fs = std.fs;

// Import utility modules for integration testing
const fs_utils = @import("utils/fs.zig");
const validation = @import("utils/validation.zig");
const time = @import("utils/time.zig");
const colors = @import("utils/colors.zig");
const git = @import("utils/git.zig");

// Integration Tests (Bug #19)
// These test the interaction between different modules

test "integration: branch name validation and path construction" {
    const allocator = testing.allocator;
    
    // Test: Validate a branch name then construct its path
    const branch_name = "feature/auth-system";
    
    // This should pass validation
    try validation.validateBranchName(branch_name);
    
    // Create a temporary directory for testing
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();
    
    const tmp_path = try tmp_dir.dir.realpathAlloc(allocator, ".");
    defer allocator.free(tmp_path);
    
    const test_repo = try fs.path.join(allocator, &.{ tmp_path, "test-repo" });
    defer allocator.free(test_repo);
    try tmp_dir.dir.makeDir("test-repo");
    
    // Test: Construct worktree path from validated branch name
    const worktree_path = try fs_utils.constructWorktreePath(allocator, test_repo, "test-repo", branch_name, null);
    defer allocator.free(worktree_path);
    
    // Verify: Path contains the expected components
    try testing.expect(std.mem.indexOf(u8, worktree_path, "feature/auth-system") != null);
    try testing.expect(std.mem.indexOf(u8, worktree_path, "test-repo-trees") != null);
}

test "integration: path sanitization and display extraction" {
    const allocator = testing.allocator;
    
    // Test: Branch with special characters
    const original_branch = "feature:test?system";
    
    // Sanitize for filesystem use
    const sanitized = try fs_utils.sanitizeBranchPath(allocator, original_branch);
    defer allocator.free(sanitized);
    
    // Create mock worktree path with sanitized branch
    const mock_path = try std.fmt.allocPrint(allocator, "/tmp/repo-trees/{s}", .{sanitized});
    defer allocator.free(mock_path);
    
    // Extract display path
    const display_path = try fs_utils.extractDisplayPath(allocator, mock_path);
    defer allocator.free(display_path);
    
    // Should show the sanitized version since that's what's in the path
    try testing.expectEqualStrings(sanitized, display_path);
    
    // Test unsanitization round-trip
    const unsanitized = try fs_utils.unsanitizeBranchPath(allocator, sanitized);
    defer allocator.free(unsanitized);
    try testing.expectEqualStrings(original_branch, unsanitized);
}

test "integration: time formatting with color output" {
    const allocator = testing.allocator;
    
    // Test various time durations
    const test_cases = [_]struct { seconds: u64, expected_contains: []const u8 }{
        .{ .seconds = 0, .expected_contains = "just now" },
        .{ .seconds = 45, .expected_contains = "45s" },
        .{ .seconds = 3600, .expected_contains = "1h" },
        .{ .seconds = 86400, .expected_contains = "1d" },
    };
    
    for (test_cases) |tc| {
        const duration = try time.formatDuration(allocator, tc.seconds);
        defer allocator.free(duration);
        
        try testing.expect(std.mem.indexOf(u8, duration, tc.expected_contains) != null);
        
        // Test: Integration with color output (should not crash)
        var buffer = std.ArrayList(u8).empty;
        defer buffer.deinit(allocator);
        const writer = buffer.writer();
        
        try colors.printInfo(writer, "Last modified: {s} ago", .{duration});
        try testing.expect(buffer.items.len > 0);
    }
}

test "integration: parent directory validation workflow" {
    const allocator = testing.allocator;
    
    // Create temporary directories for testing
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();
    
    const tmp_path = try tmp_dir.dir.realpathAlloc(allocator, ".");
    defer allocator.free(tmp_path);
    
    // Create a valid parent directory
    const parent_dir = try fs.path.join(allocator, &.{ tmp_path, "valid-parent" });
    defer allocator.free(parent_dir);
    try tmp_dir.dir.makeDir("valid-parent");
    
    // Create a mock repo (outside of parent directory)
    const repo_dir = try fs.path.join(allocator, &.{ tmp_path, "repo" });
    defer allocator.free(repo_dir);
    try tmp_dir.dir.makeDir("repo");
    
    const repo_info = git.RepoInfo{
        .root = repo_dir,
        .name = "repo",
        .is_worktree = false,
        .main_repo_root = null,
    };
    
    // Test: Validate parent directory
    const validated_path = try validation.validateParentDir(allocator, parent_dir, repo_info);
    defer allocator.free(validated_path);
    
    try testing.expect(fs.path.isAbsolute(validated_path));
    
    // Test: Use validated path in worktree path construction
    const worktree_path = try fs_utils.constructWorktreePath(allocator, repo_dir, "repo", "test-branch", validated_path);
    defer allocator.free(worktree_path);
    
    try testing.expect(std.mem.indexOf(u8, worktree_path, "test-branch") != null);
}

test "integration: error message formatting consistency" {
    // Test that validation errors provide consistent formatting
    const validation_msg = validation.getValidationErrorMessage(validation.ValidationError.EmptyBranchName);
    try testing.expect(validation_msg.len > 0);
    
    const parent_dir_msg = validation.getParentDirErrorMessage(validation.ParentDirError.ParentDirNotFound);
    try testing.expect(parent_dir_msg.len > 0);
    
    // Both should be user-friendly messages without technical jargon
    try testing.expect(std.mem.indexOf(u8, validation_msg, "empty") != null);
    try testing.expect(std.mem.indexOf(u8, parent_dir_msg, "does not exist") != null);
}

// Edge Case Tests (Bug #20) - More comprehensive edge cases

test "edge_case: extremely long branch names" {
    const allocator = testing.allocator;
    
    // Test various lengths around common filesystem limits
    const test_lengths = [_]usize{ 50, 100, 200, 255, 500 };
    
    for (test_lengths) |length| {
        // Create a branch name of specific length
        const branch_name = try allocator.alloc(u8, length);
        defer allocator.free(branch_name);
        
        // Fill with valid characters
        for (branch_name, 0..) |*char, i| {
            char.* = @as(u8, @intCast('a' + (i % 26)));
        }
        
        // Test sanitization (should not crash)
        const sanitized = try fs_utils.sanitizeBranchPath(allocator, branch_name);
        defer allocator.free(sanitized);
        
        // Should produce some result
        try testing.expect(sanitized.len > 0);
        
        // Test unsanitization round-trip
        const unsanitized = try fs_utils.unsanitizeBranchPath(allocator, sanitized);
        defer allocator.free(unsanitized);
        try testing.expectEqualStrings(branch_name, unsanitized);
    }
}

test "edge_case: unicode and special characters" {
    const allocator = testing.allocator;
    
    const test_cases = [_][]const u8{
        "café-branch", // accented characters
        "브랜치-이름",   // Korean characters
        "🚀-rocket",   // emoji
        "branch_with_underscores",
        "branch-with-dashes",
        "MixedCaseRanch",
        "123numeric456",
    };
    
    for (test_cases) |branch_name| {
        // Test validation (may pass or fail, but shouldn't crash)
        const validation_result = validation.validateBranchName(branch_name);
        
        if (validation_result) {
            // If validation passes, test further processing
            const sanitized = try fs_utils.sanitizeBranchPath(allocator, branch_name);
            defer allocator.free(sanitized);
            
            const unsanitized = try fs_utils.unsanitizeBranchPath(allocator, sanitized);
            defer allocator.free(unsanitized);
            
            try testing.expectEqualStrings(branch_name, unsanitized);
        } else |err| {
            // Validation failed, which is acceptable for some edge cases
            try testing.expect(err != error.OutOfMemory); // Should not be a crash
        }
    }
}

test "edge_case: boundary value time formatting" {
    const allocator = testing.allocator;
    
    const boundary_values = [_]u64{
        0,                    // absolute minimum
        1,                    // just above minimum
        59,                   // just below minute
        60,                   // exactly one minute
        61,                   // just above minute
        3599,                 // just below hour
        3600,                 // exactly one hour
        86399,                // just below day
        86400,                // exactly one day
        2592000,              // approximately one month
        31536000,             // approximately one year
        315360000,            // approximately one decade
        std.math.maxInt(u32), // large value
    };
    
    for (boundary_values) |seconds| {
        const formatted = try time.formatDuration(allocator, seconds);
        defer allocator.free(formatted);
        
        // Should always produce non-empty result
        try testing.expect(formatted.len > 0);
        
        // Should not contain any obviously incorrect formatting
        try testing.expect(!std.mem.eql(u8, formatted, ""));
        try testing.expect(!std.mem.startsWith(u8, formatted, "0"));
    }
}

test "edge_case: path construction with edge cases" {
    const allocator = testing.allocator;
    
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();
    
    const tmp_path = try tmp_dir.dir.realpathAlloc(allocator, ".");
    defer allocator.free(tmp_path);
    
    const test_repo = try fs.path.join(allocator, &.{ tmp_path, "test-repo" });
    defer allocator.free(test_repo);
    try tmp_dir.dir.makeDir("test-repo");
    
    const edge_case_branches = [_][]const u8{
        "a",                    // single character
        "ab",                   // two characters
        "a/b",                  // single level nesting
        "a/b/c/d/e",           // deep nesting
        "feature--double-dash", // double dash
        "123456789",           // all numeric
        "UPPERCASE",           // all uppercase
        "lowercase",           // all lowercase
    };
    
    for (edge_case_branches) |branch| {
        const path_result = fs_utils.constructWorktreePath(allocator, test_repo, "test-repo", branch, null) catch |err| {
            // Some edge cases might fail, which is acceptable
            try testing.expect(err != error.OutOfMemory);
            continue;
        };
        defer allocator.free(path_result);
        
        // If successful, should contain expected components
        try testing.expect(std.mem.indexOf(u8, path_result, branch) != null);
        try testing.expect(fs.path.isAbsolute(path_result));
    }
}

test "edge_case: display path extraction robustness" {
    const allocator = testing.allocator;
    
    const edge_paths = [_][]const u8{
        "/",                           // root
        "",                            // empty
        "/path/to/repo-trees/",       // trailing slash
        "/path/to/repo-trees//branch", // double slash
        "/path\\to\\repo-trees\\branch", // mixed separators
        "relative/path/repo-trees/branch", // relative path
        "/very/long/path/" ++ "component/" ** 50 ++ "repo-trees/branch", // very long
    };
    
    for (edge_paths) |path| {
        const result = fs_utils.extractDisplayPath(allocator, path) catch |err| {
            // Some edge cases might fail gracefully
            try testing.expect(err != error.OutOfMemory);
            continue;
        };
        defer allocator.free(result);
        
        // Should always produce some non-empty result
        try testing.expect(result.len > 0);
    }
}