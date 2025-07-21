const std = @import("std");
const fs = std.fs;

/// Characters that need to be escaped for filesystem paths
const UNSAFE_CHARS = [_]struct { char: u8, replacement: []const u8 }{
    .{ .char = ':', .replacement = "%3A" },
    .{ .char = '?', .replacement = "%3F" },
    .{ .char = '*', .replacement = "%2A" },
    .{ .char = '|', .replacement = "%7C" },
    .{ .char = '<', .replacement = "%3C" },
    .{ .char = '>', .replacement = "%3E" },
    .{ .char = '"', .replacement = "%22" },
};

/// Sanitize branch name for filesystem path (URL encode unsafe characters)
pub fn sanitizeBranchPath(allocator: std.mem.Allocator, branch: []const u8) ![]u8 {
    var result = std.ArrayList(u8).init(allocator);
    defer result.deinit();
    
    for (branch) |char| {
        var found = false;
        for (UNSAFE_CHARS) |unsafe| {
            if (char == unsafe.char) {
                try result.appendSlice(unsafe.replacement);
                found = true;
                break;
            }
        }
        if (!found) {
            try result.append(char);
        }
    }
    
    return result.toOwnedSlice();
}

/// Reverse sanitization for display (decode URL encoded characters)
pub fn unsanitizeBranchPath(allocator: std.mem.Allocator, path: []const u8) ![]u8 {
    var result = std.ArrayList(u8).init(allocator);
    defer result.deinit();
    
    var i: usize = 0;
    while (i < path.len) {
        if (path[i] == '%' and i + 2 < path.len) {
            // Check if this matches any of our encoded characters
            var found = false;
            for (UNSAFE_CHARS) |unsafe| {
                if (std.mem.startsWith(u8, path[i..], unsafe.replacement)) {
                    try result.append(unsafe.char);
                    i += unsafe.replacement.len;
                    found = true;
                    break;
                }
            }
            if (!found) {
                try result.append(path[i]);
                i += 1;
            }
        } else {
            try result.append(path[i]);
            i += 1;
        }
    }
    
    return result.toOwnedSlice();
}

/// Configuration files to copy when creating a new worktree
pub const CONFIG_FILES = [_][]const u8{
    ".claude",
    ".env",
    ".env.local",
    ".env.development",
    ".env.test",
    ".env.production",
    "CLAUDE.local.md",
    ".ai-cache",
    "node_modules",
};

/// Construct worktree path based on repository structure
pub fn constructWorktreePath(
    allocator: std.mem.Allocator, 
    repo_root: []const u8,
    repo_name: []const u8,
    branch_name: []const u8,
    parent_dir: ?[]const u8
) ![]u8 {
    // Sanitize the branch name for filesystem usage
    const sanitized_branch = try sanitizeBranchPath(allocator, branch_name);
    defer allocator.free(sanitized_branch);
    
    const worktree_path = if (parent_dir) |custom_parent| blk: {
        // Use custom parent directory (already validated and absolute)
        break :blk try fs.path.join(allocator, &.{ custom_parent, sanitized_branch });
    } else blk: {
        // Default behavior: ../repo-trees/branch-name
        const repo_parent = fs.path.dirname(repo_root) orelse ".";
        const trees_dir = try std.fmt.allocPrint(allocator, "{s}-trees", .{repo_name});
        defer allocator.free(trees_dir);
        
        break :blk try fs.path.join(allocator, &.{ repo_parent, trees_dir, sanitized_branch });
    };
    
    // Ensure parent directories exist if branch contains slashes
    if (std.mem.indexOf(u8, sanitized_branch, "/") != null) {
        const worktree_parent = fs.path.dirname(worktree_path) orelse return worktree_path;
        fs.cwd().makePath(worktree_parent) catch |err| switch (err) {
            error.PathAlreadyExists => {},
            else => return err,
        };
    }
    
    return worktree_path;
}

/// Copy configuration files from source to destination
pub fn copyConfigFiles(allocator: std.mem.Allocator, src_root: []const u8, dst_root: []const u8) !void {
    for (CONFIG_FILES) |config_item| {
        const src_path = try fs.path.join(allocator, &.{ src_root, config_item });
        defer allocator.free(src_path);
        
        const dst_path = try fs.path.join(allocator, &.{ dst_root, config_item });
        defer allocator.free(dst_path);
        
        // Check if source exists
        const src_stat = fs.cwd().statFile(src_path) catch |err| switch (err) {
            error.FileNotFound => continue,
            else => return err,
        };
        
        if (src_stat.kind == .directory) {
            // Copy directory recursively
            try copyDir(src_path, dst_path);
        } else {
            // Copy file
            try fs.cwd().copyFile(src_path, fs.cwd(), dst_path, .{});
        }
    }
}

/// Helper to ensure directory exists
fn ensureDir(path: []const u8) !void {
    fs.cwd().makeDir(path) catch |err| switch (err) {
        error.PathAlreadyExists => {},
        else => return err,
    };
}

/// Copy directory recursively
fn copyDir(src: []const u8, dst: []const u8) !void {
    try ensureDir(dst);
    
    var src_dir = try fs.cwd().openDir(src, .{ .iterate = true });
    defer src_dir.close();
    
    // Use a general purpose allocator for the walker
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();
    
    var walker = try src_dir.walk(allocator);
    defer walker.deinit();
    
    while (try walker.next()) |entry| {
        const src_path = try fs.path.join(allocator, &.{ src, entry.path });
        defer allocator.free(src_path);
        
        const dst_path = try fs.path.join(allocator, &.{ dst, entry.path });
        defer allocator.free(dst_path);
        
        switch (entry.kind) {
            .directory => try ensureDir(dst_path),
            .file => {
                // Ensure parent directory exists
                if (fs.path.dirname(dst_path)) |parent| {
                    try fs.cwd().makePath(parent);
                }
                try fs.cwd().copyFile(src_path, fs.cwd(), dst_path, .{});
            },
            else => {}, // Skip other types
        }
    }
}

/// Check if a file exists in the given path
fn fileExists(base_path: []const u8, file_name: []const u8) !bool {
    const full_path = try fs.path.join(std.heap.page_allocator, &.{ base_path, file_name });
    defer std.heap.page_allocator.free(full_path);
    
    const stat = fs.cwd().statFile(full_path) catch |err| switch (err) {
        error.FileNotFound => return false,
        else => return err,
    };
    
    return stat.kind == .file;
}

/// Check if a file or directory exists at the given path (simple version)
pub fn pathExists(path: []const u8) bool {
    const stat = fs.cwd().statFile(path) catch return false;
    _ = stat;
    return true;
}

/// Check if we have node.js project files
pub fn hasNodeProject(path: []const u8) !bool {
    return try fileExists(path, "package.json");
}

/// Check if we have .nvmrc file
pub fn hasNvmrc(path: []const u8) !bool {
    return try fileExists(path, ".nvmrc");
}

/// Check if package.json uses yarn
pub fn usesYarn(allocator: std.mem.Allocator, path: []const u8) !bool {
    const package_json = try fs.path.join(allocator, &.{ path, "package.json" });
    defer allocator.free(package_json);
    
    const file = fs.cwd().openFile(package_json, .{}) catch |err| switch (err) {
        error.FileNotFound => return false,
        else => return err,
    };
    defer file.close();
    
    const content = try file.readToEndAlloc(allocator, 1024 * 1024); // 1MB max
    defer allocator.free(content);
    
    // Simple check for packageManager field containing yarn
    return std.mem.indexOf(u8, content, "\"packageManager\"") != null and
           std.mem.indexOf(u8, content, "\"yarn") != null;
}

/// Extract a user-friendly display path from an absolute worktree path
/// Returns the branch name (basename) for display purposes, making output consistent
/// For the main repository, detects it and returns "[main]"
pub fn extractDisplayPath(allocator: std.mem.Allocator, worktree_path: []const u8) ![]u8 {
    // Check if this appears to be a main repository (not in a -trees directory)
    if (std.mem.indexOf(u8, worktree_path, "-trees") == null) {
        return try allocator.dupe(u8, "[main]");
    }
    
    // For worktrees, return the basename (branch name)
    const basename = fs.path.basename(worktree_path);
    return try allocator.dupe(u8, basename);
}

test "constructWorktreePath" {
    const allocator = std.testing.allocator;
    
    // Create a temporary directory for testing
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();
    
    const tmp_path = try tmp_dir.dir.realpathAlloc(allocator, ".");
    defer allocator.free(tmp_path);
    
    // Create test repo directory
    const test_repo = try fs.path.join(allocator, &.{ tmp_path, "myrepo" });
    defer allocator.free(test_repo);
    try tmp_dir.dir.makeDir("myrepo");
    
    // Test with default parent
    const path1 = try constructWorktreePath(allocator, test_repo, "myrepo", "feature-branch", null);
    defer allocator.free(path1);
    const expected1 = try fs.path.join(allocator, &.{ tmp_path, "myrepo-trees", "feature-branch" });
    defer allocator.free(expected1);
    try std.testing.expectEqualStrings(expected1, path1);
    
    // Test with custom parent
    const custom_parent = try fs.path.join(allocator, &.{ tmp_path, "custom" });
    defer allocator.free(custom_parent);
    try tmp_dir.dir.makeDir("custom");
    
    const path2 = try constructWorktreePath(allocator, test_repo, "myrepo", "feature-branch", custom_parent);
    defer allocator.free(path2);
    const expected2 = try fs.path.join(allocator, &.{ custom_parent, "feature-branch" });
    defer allocator.free(expected2);
    try std.testing.expectEqualStrings(expected2, path2);
    
    // Test with branch containing slashes - this will create the parent directories
    const path3 = try constructWorktreePath(allocator, test_repo, "myrepo", "feature/new-ui", custom_parent);
    defer allocator.free(path3);
    const expected3 = try fs.path.join(allocator, &.{ custom_parent, "feature/new-ui" });
    defer allocator.free(expected3);
    try std.testing.expectEqualStrings(expected3, path3);
    
    // Verify the parent directory was created
    var feature_dir = try tmp_dir.dir.openDir("custom/feature", .{});
    feature_dir.close();
}

test "config files list" {
    try std.testing.expect(CONFIG_FILES.len > 0);
    try std.testing.expectEqualStrings(".claude", CONFIG_FILES[0]);
    
    // Ensure all important files are in the list
    var has_env = false;
    var has_claude_local = false;
    for (CONFIG_FILES) |file| {
        if (std.mem.eql(u8, file, ".env")) has_env = true;
        if (std.mem.eql(u8, file, "CLAUDE.local.md")) has_claude_local = true;
    }
    try std.testing.expect(has_env);
    try std.testing.expect(has_claude_local);
}

test "sanitizeBranchPath and unsanitizeBranchPath" {
    const allocator = std.testing.allocator;
    
    // Test normal branch name (no changes)
    const normal = try sanitizeBranchPath(allocator, "feature-branch");
    defer allocator.free(normal);
    try std.testing.expectEqualStrings("feature-branch", normal);
    
    // Test branch with slashes (preserved)
    const with_slash = try sanitizeBranchPath(allocator, "feature/auth");
    defer allocator.free(with_slash);
    try std.testing.expectEqualStrings("feature/auth", with_slash);
    
    // Test branch with colon (encoded)
    const with_colon = try sanitizeBranchPath(allocator, "feature:experimental");
    defer allocator.free(with_colon);
    try std.testing.expectEqualStrings("feature%3Aexperimental", with_colon);
    
    // Test branch with multiple special chars
    const complex = try sanitizeBranchPath(allocator, "feat:test?v2");
    defer allocator.free(complex);
    try std.testing.expectEqualStrings("feat%3Atest%3Fv2", complex);
    
    // Test reverse sanitization
    const reversed = try unsanitizeBranchPath(allocator, "feat%3Atest%3Fv2");
    defer allocator.free(reversed);
    try std.testing.expectEqualStrings("feat:test?v2", reversed);
    
    // Test reverse with normal path
    const normal_reverse = try unsanitizeBranchPath(allocator, "feature/auth");
    defer allocator.free(normal_reverse);
    try std.testing.expectEqualStrings("feature/auth", normal_reverse);
}

test "fileExists and ensureDir" {
    const allocator = std.testing.allocator;
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();
    
    // Test fileExists with non-existent file
    const tmp_path = try tmp_dir.dir.realpathAlloc(allocator, ".");
    defer allocator.free(tmp_path);
    
    try std.testing.expect(!try fileExists(tmp_path, "test.txt"));
    
    // Create file and test again
    const file = try tmp_dir.dir.createFile("test.txt", .{});
    file.close();
    
    try std.testing.expect(try fileExists(tmp_path, "test.txt"));
    
    // Test ensureDir
    const test_dir = try std.fmt.allocPrint(allocator, "{s}/testdir", .{tmp_path});
    defer allocator.free(test_dir);
    
    try ensureDir(test_dir);
    
    // Check directory was created
    var dir = try fs.cwd().openDir(test_dir, .{});
    dir.close();
}