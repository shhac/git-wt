const std = @import("std");
const process = std.process;
const fs = std.fs;

const git = @import("../utils/git.zig");
const fs_utils = @import("../utils/fs.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");

const WorktreeInfo = struct {
    path: []const u8,
    branch: []const u8,
    mod_time: i128,
    is_main: bool,
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
        
        try findWorktreesRecursively(allocator, worktrees, base_path, new_relative, depth + 1);
    }
}

pub fn printHelp() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("Usage: git-wt go [branch-name]\n\n", .{});
    try stdout.print("Navigate to a git worktree or the main repository.\n\n", .{});
    try stdout.print("Arguments:\n", .{});
    try stdout.print("  [branch-name]    Name of the branch/worktree to navigate to (optional)\n\n", .{});
    try stdout.print("Options:\n", .{});
    try stdout.print("  -h, --help       Show this help message\n", .{});
    try stdout.print("  -n, --non-interactive  List worktrees without interaction\n", .{});
    try stdout.print("  --show-command   Output shell cd commands instead of navigating\n", .{});
    try stdout.print("  --no-color       Disable colored output\n", .{});
    try stdout.print("  --plain          Output plain paths only (one per line)\n\n", .{});
    try stdout.print("Examples:\n", .{});
    try stdout.print("  git-wt go                      # Interactive selection of worktrees\n", .{});
    try stdout.print("  git-wt go main                 # Navigate to main repository\n", .{});
    try stdout.print("  git-wt go feature-branch       # Navigate to feature-branch worktree\n", .{});
    try stdout.print("  git-wt go --non-interactive    # List worktrees with timestamps\n", .{});
    try stdout.print("  git-wt go -n --no-color        # List without colors\n", .{});
    try stdout.print("  git-wt go -n --plain           # List paths only\n\n", .{});
    try stdout.print("This command will:\n", .{});
    try stdout.print("  1. List all available worktrees (sorted by modification time)\n", .{});
    try stdout.print("  2. Allow interactive selection if no branch specified\n", .{});
    try stdout.print("  3. Navigate to the selected worktree\n", .{});
    try stdout.print("  4. Change the current working directory\n\n", .{});
    try stdout.print("Note: Use 'main' as the branch name to navigate to the main repository.\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, branch_name: ?[]const u8, non_interactive: bool, no_color: bool, plain: bool, show_command: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    // Get repository info
    const repo_info = git.getRepoInfo(allocator) catch |err| {
        try colors.printError(stderr, "Not in a git repository", .{});
        return err;
    };
    defer allocator.free(repo_info.root);
    defer if (repo_info.main_repo_root) |root| allocator.free(root);
    
    const main_repo = if (repo_info.is_worktree) repo_info.main_repo_root.? else repo_info.root;
    const repo_name = fs.path.basename(main_repo);
    const parent_dir = fs.path.dirname(main_repo) orelse ".";
    const trees_dir = try std.fmt.allocPrint(allocator, "{s}/{s}-trees", .{ parent_dir, repo_name });
    defer allocator.free(trees_dir);
    
    if (branch_name) |branch| {
        // Direct navigation
        if (std.mem.eql(u8, branch, "main")) {
            if (show_command) {
                try stdout.print("cd {s}\n", .{main_repo});
            } else {
                try colors.printPath(stdout, "📁 Navigating to main repository:", main_repo);
                try process.changeCurDir(main_repo);
            }
            return;
        }
        
        // Navigate to specific worktree
        const worktree_path = try std.fmt.allocPrint(allocator, "{s}/{s}", .{ trees_dir, branch });
        defer allocator.free(worktree_path);
        
        // Check if directory exists
        fs.cwd().access(worktree_path, .{}) catch {
            try stderr.print("{s}Error:{s} Worktree for branch '{s}' not found at:\n", .{ colors.error_prefix, colors.reset, branch });
            try stderr.print("       {s}{s}{s}\n", .{ colors.path_color, worktree_path, colors.reset });
            return error.WorktreeNotFound;
        };
        
        if (show_command) {
            try stdout.print("cd {s}\n", .{worktree_path});
        } else {
            try colors.printPath(stdout, "📁 Navigating to worktree:", worktree_path);
            try process.changeCurDir(worktree_path);
        }
    } else {
        // Interactive selection (or just list if non-interactive)
        var worktrees = std.ArrayList(WorktreeInfo).init(allocator);
        defer {
            for (worktrees.items) |wt| {
                allocator.free(wt.path);
                allocator.free(wt.branch);
            }
            worktrees.deinit();
        }
        
        // Add main repository if we're not already in it
        if (!std.mem.eql(u8, repo_info.root, main_repo)) {
            const stat = try fs.cwd().statFile(main_repo);
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
                .mod_time = stat.mtime,
                .is_main = true,
            });
        }
        
        // Find worktrees in trees directory (recursively)
        if (fs.cwd().openDir(trees_dir, .{ .iterate = true })) |dir| {
            var trees_dir_handle = dir;
            defer trees_dir_handle.close();
            
            try findWorktreesRecursively(allocator, &worktrees, trees_dir, "", 0);
        } else |_| {}
        
        if (worktrees.items.len == 0) {
            try stdout.print("{s}No worktrees found in:{s} {s}{s}{s}\n", .{
                colors.warning_prefix,
                colors.reset,
                colors.path_color,
                trees_dir,
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
        
        // Display worktrees
        if (!plain and !show_command) {
            if (non_interactive and no_color) {
                try stdout.print("Available worktrees:\n", .{});
            } else {
                try colors.printInfo(stdout, "Available worktrees:\n", .{});
            }
        }
        
        for (worktrees.items, 1..) |wt, idx| {
            const display_name = if (wt.is_main) "[main repository]" else blk: {
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
            
            if (show_command) {
                if (plain) {
                    // Plain mode - just paths
                    try stdout.print("{s}\n", .{wt.path});
                } else {
                    // Show command mode - output cd commands
                    try stdout.print("cd {s}  # {s} @ {s} - {s} ago\n", .{
                        wt.path,
                        display_name,
                        wt.branch,
                        duration_str,
                    });
                }
            } else {
                // Both interactive and non-interactive show the same list
                if (non_interactive and no_color) {
                    try stdout.print("  {d}) {s} @ {s} - {s} ago\n", .{
                        idx,
                        display_name,
                        wt.branch,
                        duration_str,
                    });
                } else {
                    try stdout.print("  {s}{d}{s}) {s}{s}{s} @ {s}{s}{s}\n", .{
                        colors.green,
                        idx,
                        colors.reset,
                        colors.path_color,
                        display_name,
                        colors.reset,
                        colors.magenta,
                        wt.branch,
                        colors.reset,
                    });
                    
                    // Format timestamp
                    try stdout.print("     {s}Last modified:{s} {s} ago\n", .{
                        colors.yellow,
                        colors.reset,
                        duration_str,
                    });
                }
            }
        }
        
        // In non-interactive mode without a specific selection, just list and exit
        if (non_interactive) {
            return;
        }
        
        // In show_command mode, output the first (most recent) worktree automatically
        if (show_command) {
            if (worktrees.items.len > 0) {
                const selected = worktrees.items[0];
                try stdout.print("cd {s}\n", .{selected.path});
            }
            return;
        }
        
        const prompt = try std.fmt.allocPrint(allocator, "\n{s}Enter number to navigate to (or 'q' to quit):{s}", .{ colors.yellow, colors.reset });
        defer allocator.free(prompt);
        
        if (try input.readLine(allocator, prompt)) |response| {
            defer allocator.free(response);
            
            if (response.len > 0 and (response[0] == 'q' or response[0] == 'Q')) {
                try colors.printInfo(stdout, "Cancelled", .{});
                return;
            }
            
            const selection = if (response.len == 0) 1 else std.fmt.parseInt(usize, response, 10) catch {
                try colors.printError(stderr, "Invalid selection", .{});
                return error.InvalidSelection;
            };
            
            if (selection < 1 or selection > worktrees.items.len) {
                try colors.printError(stderr, "Invalid selection", .{});
                return error.InvalidSelection;
            }
            
            const selected = worktrees.items[selection - 1];
            try colors.printPath(stdout, "📁 Navigating to worktree:", selected.path);
            try process.changeCurDir(selected.path);
        }
    }
}