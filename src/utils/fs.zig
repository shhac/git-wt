const std = @import("std");
const fs = std.fs;

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
};

/// Construct worktree path based on repository structure
pub fn constructWorktreePath(allocator: std.mem.Allocator, repo_root: []const u8, branch_name: []const u8) ![]u8 {
    const repo_name = fs.path.basename(repo_root);
    const parent_dir = fs.path.dirname(repo_root) orelse ".";
    
    return std.fmt.allocPrint(allocator, "{s}/{s}-trees/{s}", .{
        parent_dir,
        repo_name,
        branch_name,
    });
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
    
    var walker = try src_dir.walk(std.heap.page_allocator);
    defer walker.deinit();
    
    while (try walker.next()) |entry| {
        const src_path = try fs.path.join(std.heap.page_allocator, &.{ src, entry.path });
        defer std.heap.page_allocator.free(src_path);
        
        const dst_path = try fs.path.join(std.heap.page_allocator, &.{ dst, entry.path });
        defer std.heap.page_allocator.free(dst_path);
        
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

test "constructWorktreePath" {
    const allocator = std.testing.allocator;
    
    const path = try constructWorktreePath(allocator, "/home/user/projects/myrepo", "feature-branch");
    defer allocator.free(path);
    
    try std.testing.expectEqualStrings("/home/user/projects/myrepo-trees/feature-branch", path);
    
    // Test with root directory
    const path2 = try constructWorktreePath(allocator, "/myrepo", "test");
    defer allocator.free(path2);
    try std.testing.expectEqualStrings("//myrepo-trees/test", path2);
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