const std = @import("std");
const process = std.process;
const fs = std.fs;

const git = @import("../utils/git.zig");
const fs_utils = @import("../utils/fs.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");
const fd = @import("../utils/fd.zig");
const interactive = @import("../utils/interactive.zig");

pub fn formatDuration(allocator: std.mem.Allocator, seconds: u64) ![]u8 {
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


pub fn printHelp() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("Usage: git-wt go [branch-name]\n\n", .{});
    try stdout.print("Navigate to a git worktree or the main repository.\n\n", .{});
    try stdout.print("Arguments:\n", .{});
    try stdout.print("  [branch-name]    Name of the branch/worktree to navigate to (optional)\n\n", .{});
    try stdout.print("Options:\n", .{});
    try stdout.print("  -h, --help       Show this help message\n", .{});
    try stdout.print("  -n, --non-interactive  List worktrees without interaction\n", .{});
    try stdout.print("  --no-tty         Force number-based selection (disable arrow keys)\n", .{});
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
    
    // Get all worktrees using git worktree list
    const worktrees = try git.listWorktrees(allocator);
    defer git.freeWorktrees(allocator, worktrees);
    
    if (worktrees.len == 0) {
        try stderr.print("{s}Error:{s} No worktrees found\n", .{ colors.error_prefix, colors.reset });
        return error.NoWorktrees;
    }
    
    if (branch_name) |branch| {
        // Direct navigation to specific branch
        for (worktrees) |wt| {
            // Check if branch matches (handle both "main" for the main worktree and regular branch names)
            const matches = if (std.mem.eql(u8, branch, "main") and std.mem.indexOf(u8, wt.path, "-trees") == null) 
                true
            else 
                std.mem.eql(u8, wt.branch, branch) or std.mem.endsWith(u8, wt.path, branch);
                
            if (matches) {
                if (show_command) {
                    // Use fd 3 if available for cleaner shell integration
                    const cmd_writer = fd.CommandWriter.init();
                    try cmd_writer.print("cd {s}\n", .{wt.path});
                } else {
                    try colors.printPath(stdout, "📁 Navigating to worktree:", wt.path);
                    try process.changeCurDir(wt.path);
                }
                return;
            }
        }
        
        // Not found
        try stderr.print("{s}Error:{s} Worktree for branch '{s}' not found\n", .{ 
            colors.error_prefix, colors.reset, branch 
        });
        return error.WorktreeNotFound;
    } else {
        // Interactive selection (or just list if non-interactive)
        
        // Get worktrees with modification times, sorted by newest first
        const worktrees_with_time = try git.listWorktreesWithTime(allocator, true);
        defer git.freeWorktreesWithTime(allocator, worktrees_with_time);
        
        if (worktrees_with_time.len == 0) {
            try stdout.print("{s}No other worktrees found{s}\n", .{
                colors.warning_prefix,
                colors.reset,
            });
            return;
        }
        
        // Check if we'll use interactive mode
        const will_use_interactive = !non_interactive and interactive.isStdinTty() and interactive.isStdoutTty() and (!show_command or fd.isEnabled());
        
        // Display worktrees (skip if we're going to show interactive UI)
        if (!plain and !will_use_interactive) {
            const header_writer = if (show_command) stderr else stdout;
            if (non_interactive and no_color) {
                try header_writer.print("Available worktrees:\n", .{});
            } else {
                try colors.printInfo(header_writer, "Available worktrees:\n", .{});
            }
        }
        
        // Only display the list if we're not going to show interactive UI
        if (!will_use_interactive) {
            for (worktrees_with_time, 1..) |wt_info, idx| {
                const wt = wt_info.worktree;
                const timestamp = @divFloor(wt_info.mod_time, std.time.ns_per_s);
                const time_ago_seconds = @as(u64, @intCast(std.time.timestamp() - timestamp));
                const duration_str = try formatDuration(allocator, time_ago_seconds);
                defer allocator.free(duration_str);
                
                if (show_command and !non_interactive) {
                // In interactive show_command mode, show the numbered list to stderr
                try stderr.print("  {s}{d}{s}) {s}{s}{s} @ {s}{s}{s}\n", .{
                    colors.green,
                    idx,
                    colors.reset,
                    colors.path_color,
                    wt_info.display_name,
                    colors.reset,
                    colors.magenta,
                    wt.branch,
                    colors.reset,
                });
                
                // Format timestamp
                try stderr.print("     {s}Last modified:{s} {s} ago\n", .{
                    colors.yellow,
                    colors.reset,
                    duration_str,
                });
            } else if (show_command) {
                if (plain) {
                    // Plain mode - just paths
                    try stdout.print("{s}\n", .{wt.path});
                } else {
                    // Show command mode - output cd commands
                    const cmd_writer = fd.CommandWriter.init();
                    try cmd_writer.print("cd {s}  # {s} @ {s} - {s} ago\n", .{
                        wt.path,
                        wt_info.display_name,
                        wt.branch,
                        duration_str,
                    });
                }
            } else {
                // Both interactive and non-interactive show the same list
                if (non_interactive and no_color) {
                    try stdout.print("  {d}) {s} @ {s} - {s} ago\n", .{
                        idx,
                        wt_info.display_name,
                        wt.branch,
                        duration_str,
                    });
                } else {
                    try stdout.print("  {s}{d}{s}) {s}{s}{s} @ {s}{s}{s}\n", .{
                        colors.green,
                        idx,
                        colors.reset,
                        colors.path_color,
                        wt_info.display_name,
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
        }
        
        // In non-interactive mode without a specific selection, just list and exit
        if (non_interactive) {
            return;
        }
        
        // Try interactive selection first
        // Use interactive mode when:
        // - We have TTY for input/output
        // - Not in show_command mode (unless fd3 is enabled, in which case we still want interactive UI)
        const use_interactive = interactive.isStdinTty() and interactive.isStdoutTty() and (!show_command or fd.isEnabled());
        
        if (use_interactive) {
            // Build list of options for interactive selection
            var options_list = std.ArrayList([]u8).init(allocator);
            defer options_list.deinit();
            defer for (options_list.items) |item| allocator.free(item);
            
            for (worktrees_with_time) |wt_info| {
                const timestamp = @divFloor(wt_info.mod_time, std.time.ns_per_s);
                const time_ago_seconds = @as(u64, @intCast(std.time.timestamp() - timestamp));
                const duration_str = try formatDuration(allocator, time_ago_seconds);
                defer allocator.free(duration_str);
                
                const option_text = if (no_color) try std.fmt.allocPrint(allocator, "{s} @ {s} - {s} ago", .{
                    wt_info.display_name,
                    wt_info.worktree.branch,
                    duration_str,
                }) else try std.fmt.allocPrint(allocator, "{s}{s}{s} @ {s}{s}{s} - {s}{s} ago{s}", .{
                    colors.path_color,
                    wt_info.display_name,
                    colors.reset,
                    colors.magenta,
                    wt_info.worktree.branch,
                    colors.reset,
                    colors.yellow,
                    duration_str,
                    colors.reset,
                });
                try options_list.append(option_text);
            }
            
            // Show header
            try colors.printInfo(stdout, "Available worktrees:", .{});
            
            const selection = try interactive.selectFromList(
                allocator,
                options_list.items,
                .{
                    .show_instructions = true,
                    .use_colors = !no_color,
                },
            );
            
            if (selection) |idx| {
                const selected = worktrees_with_time[idx].worktree;
                
                // Check if we should output to fd3 for shell integration
                if (fd.isEnabled()) {
                    const cmd_writer = fd.CommandWriter.init();
                    try cmd_writer.print("cd {s}\n", .{selected.path});
                } else {
                    try colors.printPath(stdout, "📁 Navigating to worktree:", selected.path);
                    try process.changeCurDir(selected.path);
                }
            } else {
                try colors.printInfo(stdout, "Cancelled", .{});
            }
        } else {
            // Fall back to number-based selection
            const prompt = try std.fmt.allocPrint(allocator, "\n{s}Enter number to navigate to (or 'q' to quit):{s}", .{ colors.yellow, colors.reset });
            defer allocator.free(prompt);
            
            // Handle reading input differently in show_command mode
            const response = if (show_command) blk: {
                // In show_command mode, handle prompt and input manually to avoid stdout pollution
                try stderr.print("{s} ", .{prompt});
                const stdin = std.io.getStdIn().reader();
                break :blk try stdin.readUntilDelimiterOrEofAlloc(allocator, '\n', 1024);
            } else try input.readLine(allocator, prompt);
            
            if (response) |resp| {
                defer allocator.free(resp);
                const trimmed = std.mem.trim(u8, resp, " \t\r\n");
                
                if (trimmed.len > 0 and (trimmed[0] == 'q' or trimmed[0] == 'Q')) {
                    try colors.printInfo(stdout, "Cancelled", .{});
                    return;
                }
                
                const selection = if (trimmed.len == 0) 1 else std.fmt.parseInt(usize, trimmed, 10) catch {
                    try colors.printError(stderr, "Invalid selection", .{});
                    return error.InvalidSelection;
                };
                
                if (selection < 1 or selection > worktrees_with_time.len) {
                    try colors.printError(stderr, "Invalid selection", .{});
                    return error.InvalidSelection;
                }
                
                const selected = worktrees_with_time[selection - 1].worktree;
                if (show_command) {
                    const cmd_writer = fd.CommandWriter.init();
                    try cmd_writer.print("cd {s}\n", .{selected.path});
                } else {
                    try colors.printPath(stdout, "📁 Navigating to worktree:", selected.path);
                    try process.changeCurDir(selected.path);
                }
            }
        }
    }
}

test "formatDuration" {
    const allocator = std.testing.allocator;
    
    // Test seconds
    const s = try formatDuration(allocator, 45);
    defer allocator.free(s);
    try std.testing.expectEqualStrings("45s", s);
    
    // Test minutes
    const m = try formatDuration(allocator, 150);
    defer allocator.free(m);
    try std.testing.expectEqualStrings("2m 30s", m);
    
    // Test hours
    const h = try formatDuration(allocator, 3900);
    defer allocator.free(h);
    try std.testing.expectEqualStrings("1h 5m", h);
    
    // Test days
    const d = try formatDuration(allocator, 90000);
    defer allocator.free(d);
    try std.testing.expectEqualStrings("1d 1h", d);
    
    // Test years
    const y = try formatDuration(allocator, 31536000 + 2592000);
    defer allocator.free(y);
    try std.testing.expectEqualStrings("1y 1mo", y);
    
    // Test zero
    const zero = try formatDuration(allocator, 0);
    defer allocator.free(zero);
    try std.testing.expectEqualStrings("0s", zero);
}