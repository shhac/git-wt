

const std = @import("std");
const process = std.process;
const fs = std.fs;

const git = @import("../utils/git.zig");
const colors = @import("../utils/colors.zig");
const io = @import("../utils/io.zig");

/// Escape a string for JSON output
fn escapeJson(allocator: std.mem.Allocator, str: []const u8) ![]u8 {
    var result = std.ArrayList(u8).empty;
    defer result.deinit(allocator);

    for (str) |c| {
        switch (c) {
            '"' => try result.appendSlice(allocator, "\\\""),
            '\\' => try result.appendSlice(allocator, "\\\\"),
            '\n' => try result.appendSlice(allocator, "\\n"),
            '\r' => try result.appendSlice(allocator, "\\r"),
            '\t' => try result.appendSlice(allocator, "\\t"),
            else => try result.append(allocator, c),
        }
    }

    return try result.toOwnedSlice(allocator);
}

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
    var units = std.ArrayList(struct { value: u64, unit: []const u8 }).empty;
    defer units.deinit(allocator);
    
    if (years > 0) try units.append(allocator, .{ .value = years, .unit = "y" });
    if (months > 0) try units.append(allocator, .{ .value = months, .unit = "mo" });
    if (weeks > 0) try units.append(allocator, .{ .value = weeks, .unit = "w" });
    if (days > 0) try units.append(allocator, .{ .value = days, .unit = "d" });
    if (hours > 0) try units.append(allocator, .{ .value = hours, .unit = "h" });
    if (minutes > 0) try units.append(allocator, .{ .value = minutes, .unit = "m" });
    if (secs > 0 or units.items.len == 0) try units.append(allocator, .{ .value = secs, .unit = "s" });
    
    // Format the two most significant units
    if (units.items.len == 1) {
        return try std.fmt.allocPrint(allocator, "{d}{s}", .{ units.items[0].value, units.items[0].unit });
    } else if (units.items.len >= 2) {
        return try std.fmt.allocPrint(allocator, "{d}{s} {d}{s}", .{
            units.items[0].value,
            units.items[0].unit,
            units.items[1].value,
            units.items[1].unit,
        });
    } else {
        // Fallback for unexpected empty case
        return try allocator.dupe(u8, "0s");
    }
}


pub fn printHelp() !void {
    const stdout = io.getStdOut();
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

pub fn execute(allocator: std.mem.Allocator, no_color: bool, plain: bool, json: bool) !void {
    const stdout = io.getStdOut();
    const stderr = io.getStdErr();

    // Get all worktrees using git worktree list
    const worktrees = try git.listWorktreesSmart(allocator, false); // false = not for interactive
    defer git.freeWorktrees(allocator, worktrees);

    if (worktrees.len == 0) {
        if (json) {
            try stdout.print("[]\n", .{});
        } else {
            try stderr.print("{s}Warning:{s} No worktrees found\n", .{ colors.warning_prefix, colors.reset });
        }
        return;
    }
    
    // Create a list with modification times for sorting
    const WorktreeWithTime = struct {
        worktree: git.Worktree,
        mod_time: i128,
        display_name: []const u8,
    };
    
    var worktrees_with_time = try allocator.alloc(WorktreeWithTime, worktrees.len);
    var allocated_count: usize = 0;
    defer {
        // Only free display_names that were actually allocated
        for (worktrees_with_time[0..allocated_count]) |wt| {
            allocator.free(wt.display_name);
        }
        allocator.free(worktrees_with_time);
    }
    
    // Get modification times and prepare display names
    for (worktrees, 0..) |wt, i| {
        // Try to get modification time, use 0 if path doesn't exist
        const mod_time = blk: {
            const stat = fs.cwd().statFile(wt.path) catch |err| switch (err) {
                error.FileNotFound => {
                    // Worktree directory doesn't exist (prunable)
                    break :blk 0;
                },
                else => return err,
            };
            break :blk stat.mtime;
        };
        
        // Determine display name
        const display_name = if (i == 0) // First worktree is always main
            try allocator.dupe(u8, "[main]")
        else blk2: {
            // Extract relative path from worktree path
            const basename = fs.path.basename(wt.path);
            break :blk2 try allocator.dupe(u8, basename);
        };
        
        worktrees_with_time[i] = .{
            .worktree = wt,
            .mod_time = mod_time,
            .display_name = display_name,
        };
        allocated_count = i + 1;
    }
    
    // Sort by modification time (newest first)
    std.mem.sort(WorktreeWithTime, worktrees_with_time, {}, struct {
        fn lessThan(_: void, a: WorktreeWithTime, b: WorktreeWithTime) bool {
            return a.mod_time > b.mod_time;
        }
    }.lessThan);
    
    // JSON output
    if (json) {
        try stdout.print("[\n", .{});
        for (worktrees_with_time, 0..) |wt_info, i| {
            const wt = wt_info.worktree;

            // Handle time display for missing worktrees
            const duration_str = if (wt_info.mod_time == 0)
                try allocator.dupe(u8, "unknown")
            else blk: {
                const timestamp = @divFloor(wt_info.mod_time, std.time.ns_per_s);
                const time_ago_seconds = @as(u64, @intCast(std.time.timestamp() - timestamp));
                break :blk try formatDuration(allocator, time_ago_seconds);
            };
            defer allocator.free(duration_str);

            // Escape JSON strings
            const escaped_branch = try escapeJson(allocator, wt.branch);
            defer allocator.free(escaped_branch);
            const escaped_path = try escapeJson(allocator, wt.path);
            defer allocator.free(escaped_path);
            const escaped_display = try escapeJson(allocator, wt_info.display_name);
            defer allocator.free(escaped_display);
            const escaped_duration = try escapeJson(allocator, duration_str);
            defer allocator.free(escaped_duration);

            try stdout.print("  {{\n", .{});
            try stdout.print("    \"branch\": \"{s}\",\n", .{escaped_branch});
            try stdout.print("    \"path\": \"{s}\",\n", .{escaped_path});
            try stdout.print("    \"display_name\": \"{s}\",\n", .{escaped_display});
            try stdout.print("    \"is_current\": {},\n", .{wt.is_current});
            try stdout.print("    \"last_modified\": \"{s}\"\n", .{escaped_duration});
            if (i < worktrees_with_time.len - 1) {
                try stdout.print("  }},\n", .{});
            } else {
                try stdout.print("  }}\n", .{});
            }
        }
        try stdout.print("]\n", .{});
        return;
    }

    // Display header
    if (!plain) {
        if (no_color) {
            try stdout.print("Git worktrees (sorted by last modified):\n\n", .{});
        } else {
            try colors.printInfo(stdout, "Git worktrees (sorted by last modified):\n", .{});
        }
    }

    // Display worktrees
    for (worktrees_with_time) |wt_info| {
        const wt = wt_info.worktree;
        
        // Handle time display for missing worktrees
        const duration_str = if (wt_info.mod_time == 0)
            try allocator.dupe(u8, "unknown")
        else blk: {
            const timestamp = @divFloor(wt_info.mod_time, std.time.ns_per_s);
            const time_ago_seconds = @as(u64, @intCast(std.time.timestamp() - timestamp));
            break :blk try formatDuration(allocator, time_ago_seconds);
        };
        defer allocator.free(duration_str);
        
        if (plain) {
            // Plain format: branch<tab>path<tab>time_ago
            try stdout.print("{s}\t{s}\t{s}\n", .{
                wt.branch,
                wt.path,
                duration_str,
            });
        } else {
            // Pretty format with colors
            const current_marker = if (wt.is_current) " *" else "  ";
            
            if (no_color) {
                try stdout.print("{s} {s} @ {s}\n", .{
                    current_marker,
                    wt_info.display_name,
                    wt.branch,
                });
                try stdout.print("    Last modified: {s} ago\n", .{duration_str});
            } else {
                try stdout.print("{s} {s}{s}{s} @ {s}{s}{s}", .{
                    current_marker,
                    colors.path_color,
                    wt_info.display_name,
                    colors.reset,
                    colors.magenta,
                    wt.branch,
                    colors.reset,
                });
                
                if (wt.is_current) {
                    try stdout.print(" {s}(current){s}", .{ colors.green, colors.reset });
                }
                try stdout.print("\n", .{});
                
                try stdout.print("    {s}Last modified:{s} {s} ago\n", .{
                    colors.yellow,
                    colors.reset,
                    duration_str,
                });
            }
            
            if (wt.is_detached) {
                try stdout.print("    {s}Status:{s} detached HEAD\n", .{
                    colors.yellow,
                    colors.reset,
                });
            }
            
            // Show if worktree is missing (prunable)
            if (wt_info.mod_time == 0) {
                if (no_color) {
                    try stdout.print("    Status: missing (prunable)\n", .{});
                } else {
                    try stdout.print("    {s}Status:{s} {s}missing (prunable){s}\n", .{
                        colors.yellow,
                        colors.reset,
                        colors.red,
                        colors.reset,
                    });
                }
            }
        }
    }
}
