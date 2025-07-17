const std = @import("std");
const process = std.process;
const fs = std.fs;

const git = @import("../utils/git.zig");
const colors = @import("../utils/colors.zig");

const WorktreeInfo = struct {
    path: []const u8,
    branch: []const u8,
    mod_time: i128,
    is_main: bool,
    is_current: bool,
};

const MAX_RECURSION_DEPTH = 5;

fn formatDuration(allocator: std.mem.Allocator, seconds: u64) ![]u8 {
    const minute = 60;
    const hour = minute * 60;
    const day = hour * 24;
    const week = day * 7;
    const month = day * 30;
    const year = day * 365;
    
    // Calculate each unit
    const years = seconds / year;
    const months = (seconds % year) / month;
    const weeks = (seconds % year % month) / week;
    const days = (seconds % year % month % week) / day;
    const hours = (seconds % day) / hour;
    const minutes = (seconds % hour) / minute;
    const secs = seconds % minute;
    
    // Build array of non-zero units
    var units = std.ArrayList(struct { value: u64, unit: []const u8 }).init(allocator);
    defer units.deinit();
    
    if (years > 0) try units.append(.{ .value = years, .unit = "y" });
    if (months > 0) try units.append(.{ .value = months, .unit = "mo" });
    if (weeks > 0) try units.append(.{ .value = weeks, .unit = "w" });
    if (days > 0) try units.append(.{ .value = days, .unit = "d" });
    if (hours > 0) try units.append(.{ .value = hours, .unit = "h" });
    if (minutes > 0) try units.append(.{ .value = minutes, .unit = "m" });
    if (secs > 0 or units.items.len == 0) try units.append(.{ .value = secs, .unit = "s" });
    
    // Format the two most significant units
    if (units.items.len == 1) {
        return try std.fmt.allocPrint(allocator, "{d}{s}", .{ units.items[0].value, units.items[0].unit });
    } else {
        return try std.fmt.allocPrint(allocator, "{d}{s} {d}{s}", .{
            units.items[0].value,
            units.items[0].unit,
            units.items[1].value,
            units.items[1].unit,
        });
    }
}

fn findWorktreesRecursively(
    allocator: std.mem.Allocator,
    worktrees: *std.ArrayList(WorktreeInfo),
    base_path: []const u8,
    relative_path: []const u8,
    depth: usize,
    current_worktree: ?[]const u8,
) !void {
    if (depth > MAX_RECURSION_DEPTH) return;
    
    const current_path = if (relative_path.len > 0)
        try std.fmt.allocPrint(allocator, "{s}/{s}", .{ base_path, relative_path })
    else
        base_path;
    defer if (relative_path.len > 0) allocator.free(current_path);
    
    // Check if this directory is a worktree (has .git file)
    const git_file_path = try std.fmt.allocPrint(allocator, "{s}/.git", .{current_path});
    defer allocator.free(git_file_path);
    
    if (fs.cwd().statFile(git_file_path)) |_| {
        // This is a worktree
        const stat = try fs.cwd().statFile(current_path);
        
        // Check if this is the current worktree
        const is_current = if (current_worktree) |cwt| 
            std.mem.eql(u8, current_path, cwt)
        else 
            false;
        
        // Get current branch
        var saved_cwd = try fs.cwd().openDir(".", .{});
        defer saved_cwd.close();
        try process.changeCurDir(current_path);
        const branch = try git.getCurrentBranch(allocator);
        try saved_cwd.setAsCwd();
        
        try worktrees.append(.{
            .path = try allocator.dupe(u8, current_path),
            .branch = branch,
            .mod_time = stat.mtime,
            .is_main = false,
            .is_current = is_current,
        });
        
        // Don't recurse into worktrees
        return;
    } else |_| {}
    
    // Not a worktree, recurse into subdirectories
    var dir = fs.cwd().openDir(current_path, .{ .iterate = true }) catch return;
    defer dir.close();
    
    var iter = dir.iterate();
    while (try iter.next()) |entry| {
        if (entry.kind != .directory) continue;
        
        const new_relative = if (relative_path.len > 0)
            try std.fmt.allocPrint(allocator, "{s}/{s}", .{ relative_path, entry.name })
        else
            try allocator.dupe(u8, entry.name);
        defer allocator.free(new_relative);
        
        try findWorktreesRecursively(allocator, worktrees, base_path, new_relative, depth + 1, current_worktree);
    }
}

pub fn printHelp() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("Usage: git-wt list\n\n", .{});
    try stdout.print("List all git worktrees with their status information.\n\n", .{});
    try stdout.print("Options:\n", .{});
    try stdout.print("  -h, --help       Show this help message\n", .{});
    try stdout.print("  --no-color       Disable colored output\n", .{});
    try stdout.print("  --plain          Output plain format (branch<tab>path<tab>time)\n\n", .{});
    try stdout.print("Examples:\n", .{});
    try stdout.print("  git-wt list                    # List all worktrees with details\n", .{});
    try stdout.print("  git-wt list --plain            # Machine-readable output\n", .{});
    try stdout.print("  git-wt list --no-color         # List without colors\n\n", .{});
    try stdout.print("This command will:\n", .{});
    try stdout.print("  1. Show all worktrees including the main repository\n", .{});
    try stdout.print("  2. Display branch name, path, and last modification time\n", .{});
    try stdout.print("  3. Highlight the current worktree with an indicator\n", .{});
    try stdout.print("  4. Sort by modification time (newest first)\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, no_color: bool, plain: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    // Get repository info
    const repo_info = git.getRepoInfo(allocator) catch |err| {
        try colors.printError(stderr, "Not in a git repository", .{});
        return err;
    };
    defer allocator.free(repo_info.root);
    defer if (repo_info.main_repo_root) |root| allocator.free(root);
    
    // Get current worktree path (null means we're in main)
    const current_worktree = try git.getCurrentWorktree(allocator);
    defer if (current_worktree) |wt| allocator.free(wt);
    
    const main_repo = if (repo_info.is_worktree) repo_info.main_repo_root.? else repo_info.root;
    const repo_name = fs.path.basename(main_repo);
    const parent_dir = fs.path.dirname(main_repo) orelse ".";
    const trees_dir = try std.fmt.allocPrint(allocator, "{s}/{s}-trees", .{ parent_dir, repo_name });
    defer allocator.free(trees_dir);
    
    var worktrees = std.ArrayList(WorktreeInfo).init(allocator);
    defer {
        for (worktrees.items) |wt| {
            allocator.free(wt.path);
            allocator.free(wt.branch);
        }
        worktrees.deinit();
    }
    
    // Add main repository
    const in_main = current_worktree == null;
    const main_stat = try fs.cwd().statFile(main_repo);
    const main_branch = blk: {
        var saved_cwd = try fs.cwd().openDir(".", .{});
        defer saved_cwd.close();
        try process.changeCurDir(main_repo);
        const branch = try git.getCurrentBranch(allocator);
        try saved_cwd.setAsCwd();
        break :blk branch;
    };
    
    try worktrees.append(.{
        .path = try allocator.dupe(u8, main_repo),
        .branch = main_branch,
        .mod_time = main_stat.mtime,
        .is_main = true,
        .is_current = in_main,
    });
    
    // Find worktrees in trees directory (recursively)
    if (fs.cwd().openDir(trees_dir, .{ .iterate = true })) |dir| {
        var trees_dir_handle = dir;
        defer trees_dir_handle.close();
        
        try findWorktreesRecursively(allocator, &worktrees, trees_dir, "", 0, current_worktree);
    } else |_| {}
    
    if (worktrees.items.len == 0) {
        try stdout.print("{s}No worktrees found{s}\n", .{
            colors.warning_prefix,
            colors.reset,
        });
        return;
    }
    
    // Sort by modification time (newest first)
    std.mem.sort(WorktreeInfo, worktrees.items, {}, struct {
        fn lessThan(_: void, a: WorktreeInfo, b: WorktreeInfo) bool {
            return a.mod_time > b.mod_time;
        }
    }.lessThan);
    
    // Display header
    if (!plain) {
        if (no_color) {
            try stdout.print("Git worktrees (sorted by last modified):\n\n", .{});
        } else {
            try colors.printInfo(stdout, "Git worktrees (sorted by last modified):\n", .{});
        }
    }
    
    // Display worktrees
    for (worktrees.items) |wt| {
        const display_name = if (wt.is_main) "[main]" else blk: {
            // Show relative path from trees directory for nested worktrees
            if (std.mem.indexOf(u8, wt.path, trees_dir)) |trees_idx| {
                const relative_start = trees_idx + trees_dir.len + 1; // +1 for the slash
                if (relative_start < wt.path.len) {
                    break :blk wt.path[relative_start..];
                }
            }
            break :blk fs.path.basename(wt.path);
        };
        
        const timestamp = @divFloor(wt.mod_time, std.time.ns_per_s);
        const time_ago_seconds = @as(u64, @intCast(std.time.timestamp() - timestamp));
        const duration_str = try formatDuration(allocator, time_ago_seconds);
        defer allocator.free(duration_str);
        
        if (plain) {
            // Plain format: branch<tab>path<tab>time_ago
            try stdout.print("{s}\t{s}\t{s} ago\n", .{
                wt.branch,
                wt.path,
                duration_str,
            });
        } else if (no_color) {
            // No color format
            const current_marker = if (wt.is_current) "* " else "  ";
            try stdout.print("{s}{s} @ {s}\n", .{
                current_marker,
                display_name,
                wt.branch,
            });
            try stdout.print("  Path: {s}\n", .{wt.path});
            try stdout.print("  Last modified: {s} ago\n\n", .{duration_str});
        } else {
            // Colored format
            const current_marker = if (wt.is_current) 
                try std.fmt.allocPrint(allocator, "{s}*{s} ", .{ colors.green, colors.reset })
            else 
                "  ";
            defer if (wt.is_current) allocator.free(current_marker);
            
            try stdout.print("{s}{s}{s}{s} @ {s}{s}{s}\n", .{
                current_marker,
                colors.path_color,
                display_name,
                colors.reset,
                colors.magenta,
                wt.branch,
                colors.reset,
            });
            try stdout.print("  {s}Path:{s} {s}\n", .{
                colors.yellow,
                colors.reset,
                wt.path,
            });
            try stdout.print("  {s}Last modified:{s} {s} ago\n\n", .{
                colors.yellow,
                colors.reset,
                duration_str,
            });
        }
    }
}