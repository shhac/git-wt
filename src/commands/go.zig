


const std = @import("std");
const process = std.process;
const fs = std.fs;

const git = @import("../utils/git.zig");
const fs_utils = @import("../utils/fs.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");
const fd = @import("../utils/fd.zig");
const interactive = @import("../utils/interactive.zig");
const time = @import("../utils/time.zig");
const debug = @import("../utils/debug.zig");
const io = @import("../utils/io.zig");
pub fn printHelp() !void {
    const stdout = io.getStdOut();
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
    try stdout.print("\nShell Integration:\n", .{});
    try stdout.print("  To enable directory changes from git-wt, use the shell alias:\n", .{});
    try stdout.print("  eval \"$(git-wt --alias gwt)\"\n", .{});
    try stdout.print("  Then use 'gwt go' instead of 'git-wt go' to change directories.\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, branch_name: ?[]const u8, non_interactive: bool, no_tty: bool, no_color: bool, plain: bool, show_command: bool) !void {
    const stdout = io.getStdOut();
    const stderr = io.getStdErr();
    
    // Get all worktrees using git worktree list
    const worktrees = try git.listWorktrees(allocator);
    defer git.freeWorktrees(allocator, worktrees);
    
    if (worktrees.len == 0) {
        try stderr.print("{s}Error:{s} No worktrees found\n", .{ colors.error_prefix, colors.reset });
        return error.NoWorktrees;
    }
    
    if (branch_name) |branch| {
        // Optimize: Use early-exit branch search for large repositories  
        if (git.findWorktreeByBranch(allocator, branch)) |maybe_target_wt| {
            if (maybe_target_wt) |target_wt| {
                defer {
                    allocator.free(target_wt.path);
                    allocator.free(target_wt.branch);
                    allocator.free(target_wt.commit);
                }
                
                // Use fd 3 if available for cleaner shell integration
                const cmd_writer = fd.CommandWriter.init();
                try cmd_writer.print("cd {s}\n", .{target_wt.path});
                return;
            }
        } else |_| {
            // Fallback to existing logic if fast search fails
        }
        
        // Direct navigation to specific branch (fallback)
        for (worktrees) |wt| {
            // Check if branch matches (handle both "main" for the main worktree and regular branch names)
            const matches = if (std.mem.eql(u8, branch, "main") and std.mem.indexOf(u8, wt.path, "-trees") == null) 
                true
            else 
                std.mem.eql(u8, wt.branch, branch) or std.mem.endsWith(u8, wt.path, branch);
                
            if (matches) {
                // Use fd3 if available for shell integration
                if (fd.isEnabled()) {
                    const cmd_writer = fd.CommandWriter.init();
                    try cmd_writer.print("cd {s}\n", .{wt.path});
                } else if (show_command) {
                    // If fd3 is not available but show_command is requested, output to stdout
                    try stdout.print("cd {s}\n", .{wt.path});
                } else {
                    try colors.printDisplayPath(stdout, "📁 Navigating to:", wt.path, allocator);
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
        // Use smart loading for large repositories in interactive mode
        const worktrees_with_time = try git.listWorktreesWithTimeSmart(allocator, true, !non_interactive);
        defer git.freeWorktreesWithTime(allocator, worktrees_with_time);
        
        if (worktrees_with_time.len == 0) {
            try stdout.print("{s}No other worktrees found{s}\n", .{
                colors.warning_prefix,
                colors.reset,
            });
            return;
        }
        
        // Check if we'll use interactive mode
        const will_use_interactive = !non_interactive and !no_tty and interactive.isStdinTty() and interactive.isStdoutTty() and (!show_command or fd.isEnabled());
        
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
                const duration_str = try time.formatDuration(allocator, time_ago_seconds);
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
        // - Not in no_tty mode (which forces number-based selection)
        const use_interactive = !no_tty and interactive.isStdinTty() and interactive.isStdoutTty() and (!show_command or fd.isEnabled());
        
        if (use_interactive) {
            // Build list of options for interactive selection
            var options_list = std.ArrayList([]u8).empty;
            defer options_list.deinit(allocator);
            defer for (options_list.items) |item| allocator.free(item);
            
            for (worktrees_with_time) |wt_info| {
                const timestamp = @divFloor(wt_info.mod_time, std.time.ns_per_s);
                const time_ago_seconds = @as(u64, @intCast(std.time.timestamp() - timestamp));
                const duration_str = try time.formatDuration(allocator, time_ago_seconds);
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
                try options_list.append(allocator, option_text);
            }
            
            // Don't show header - the interactive UI will handle display
            const selection = try interactive.selectFromList(
                allocator,
                options_list.items,
                .{
                    .mode = .single,
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
                    try colors.printDisplayPath(stdout, "📁 Navigating to:", selected.path, allocator);
                    try process.changeCurDir(selected.path);
                }
            } else {
                // Selection was cancelled - no need for extra output since
                // the interactive UI already cleaned up its display
                try stdout.print("Cancelled\n", .{});
            }
        } else {
            // Fall back to number-based selection
            const prompt = try std.fmt.allocPrint(allocator, "\n{s}Enter number to navigate to (or 'q' to quit):{s}", .{ colors.yellow, colors.reset });
            defer allocator.free(prompt);
            
            // Handle reading input differently in show_command mode
            const response = if (show_command) blk: {
                // In show_command mode, handle prompt and input manually to avoid stdout pollution
                try stderr.print("{s} ", .{prompt});
                const stdin = io.getStdIn();
                // Read input directly
                var buf: [1024]u8 = undefined;
                const bytes_read = try stdin.read(&buf);
                
                const line: ?[]u8 = if (bytes_read > 0) inner: {
                    for (buf[0..bytes_read], 0..) |c, i| {
                        if (c == '\n') {
                            const result = try allocator.dupe(u8, buf[0..i]);
                            break :inner result;
                        }
                    }
                    const result = try allocator.dupe(u8, buf[0..bytes_read]);
                    break :inner result;
                } else null;
                break :blk line;
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
                
                // Use fd3 if available for shell integration
                const fd_enabled = fd.isEnabled();
                if (debug.isEnabled()) {
                    std.debug.print("[DEBUG] go: fd_enabled={}, show_command={}, path={s}\n", .{ fd_enabled, show_command, selected.path });
                }
                if (fd_enabled) {
                    const cmd_writer = fd.CommandWriter.init();
                    try cmd_writer.print("cd {s}\n", .{selected.path});
                } else if (show_command) {
                    // If fd3 is not available but show_command is requested, output to stdout
                    try stdout.print("cd {s}\n", .{selected.path});
                } else {
                    try colors.printDisplayPath(stdout, "📁 Navigating to:", selected.path, allocator);
                    try process.changeCurDir(selected.path);
                }
            }
        }
    }
}
