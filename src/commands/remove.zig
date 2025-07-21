const std = @import("std");

const git = @import("../utils/git.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");
const interactive = @import("../utils/interactive.zig");
const time = @import("../utils/time.zig");
const lock = @import("../utils/lock.zig");

pub fn printHelp() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("Usage: git-wt rm [branch-name]\n\n", .{});
    try stdout.print("Remove a git worktree by branch name.\n\n", .{});
    try stdout.print("Arguments:\n", .{});
    try stdout.print("  [branch-name]    Name of the branch/worktree to remove (optional)\n", .{});
    try stdout.print("                   If not provided, shows interactive selection\n\n", .{});
    try stdout.print("Options:\n", .{});
    try stdout.print("  -h, --help       Show this help message\n", .{});
    try stdout.print("  -n, --non-interactive  Run without prompts\n", .{});
    try stdout.print("  --no-tty         Force number-based selection (disable arrow keys)\n", .{});
    try stdout.print("  -f, --force      Force removal even with uncommitted changes\n\n", .{});
    try stdout.print("Examples:\n", .{});
    try stdout.print("  git-wt rm                      # Interactive selection of worktree to remove\n", .{});
    try stdout.print("  git-wt rm feature-branch       # Remove feature-branch worktree\n", .{});
    try stdout.print("  git-wt rm feature/auth         # Remove worktree with slash in name\n", .{});
    try stdout.print("  git-wt rm test-branch -n       # Remove without prompts\n", .{});
    try stdout.print("  git-wt rm old-feature -f       # Force remove with uncommitted changes\n", .{});
    try stdout.print("  git-wt rm --no-tty             # Use number-based selection\n\n", .{});
    try stdout.print("This command will:\n", .{});
    try stdout.print("  1. Find the worktree for the specified branch\n", .{});
    try stdout.print("  2. Remove the worktree directory\n", .{});
    try stdout.print("  3. Optionally delete the associated branch\n", .{});
    try stdout.print("\nNote: The current worktree cannot be removed.\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, branch_name: []const u8, non_interactive: bool, force: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    // Get repository info for lock path
    const repo_info = git.getRepoInfo(allocator) catch |err| {
        try colors.printError(stderr, "Not in a git repository", .{});
        return err;
    };
    defer allocator.free(repo_info.root);
    defer if (repo_info.main_repo_root) |root| allocator.free(root);
    
    // Acquire lock to prevent concurrent worktree operations
    const lock_path = try std.fmt.allocPrint(allocator, "{s}/.git/git-wt.lock", .{repo_info.root});
    defer allocator.free(lock_path);
    
    var worktree_lock = lock.Lock.init(allocator, lock_path);
    defer worktree_lock.deinit();
    
    // Clean up any stale locks first
    try worktree_lock.cleanStale();
    
    // Try to acquire lock with 30 second timeout
    worktree_lock.acquire(30000) catch |err| {
        if (err == lock.LockError.LockTimeout) {
            try colors.printError(stderr, "Another git-wt operation is in progress", .{});
            try stderr.print("{s}Tip:{s} Wait for the other operation to complete or check for stale locks\n", .{
                colors.info_prefix, colors.reset
            });
        }
        return err;
    };
    
    // Prevent removing main branch
    if (std.mem.eql(u8, branch_name, "main") or std.mem.eql(u8, branch_name, "master")) {
        try colors.printError(stderr, "Cannot remove the main branch worktree", .{});
        return error.CannotRemoveMain;
    }
    
    // Get all worktrees to find the one we want to remove
    const worktrees = try git.listWorktrees(allocator);
    defer git.freeWorktrees(allocator, worktrees);
    
    // Find the worktree for the specified branch
    var worktree_path: ?[]const u8 = null;
    for (worktrees) |wt| {
        if (std.mem.eql(u8, wt.branch, branch_name)) {
            worktree_path = wt.path;
            break;
        }
    }
    
    if (worktree_path == null) {
        try colors.printError(stderr, "No worktree found for branch '{s}'", .{branch_name});
        try stderr.print("{s}Tip:{s} Use 'git-wt list' to see available worktrees\n", .{
            colors.info_prefix, colors.reset
        });
        
        // Try to find similar branch names
        var similar_branches = std.ArrayList([]const u8).init(allocator);
        defer similar_branches.deinit();
        
        for (worktrees) |wt| {
            if (std.ascii.indexOfIgnoreCase(wt.branch, branch_name) != null) {
                try similar_branches.append(wt.branch);
            }
        }
        
        if (similar_branches.items.len > 0) {
            try stderr.print("{s}Did you mean one of these?{s}\n", .{ colors.yellow, colors.reset });
            for (similar_branches.items) |branch| {
                try stderr.print("  - {s}\n", .{branch});
            }
        }
        
        return error.WorktreeNotFound;
    }
    
    // Show what we're about to remove
    try stdout.print("{s}⚠️  About to remove worktree:{s}\n", .{ colors.warning_prefix, colors.reset });
    try stdout.print("   {s}Branch:{s} {s}\n", .{ colors.yellow, colors.reset, branch_name });
    try stdout.print("   {s}Path:{s} {s}\n", .{ colors.yellow, colors.reset, worktree_path.? });
    
    // Confirm removal (unless non-interactive or force)
    if (!non_interactive and !force) {
        if (!try input.confirm("\nAre you sure you want to continue?", false)) {
            try colors.printInfo(stdout, "Cancelled", .{});
            return;
        }
    }
    
    // Remove the worktree using git
    try colors.printInfo(stdout, "Removing worktree...", .{});
    
    if (force) {
        _ = git.exec(allocator, &.{ "worktree", "remove", "--force", worktree_path.? }) catch |err| {
            if (err == git.GitError.CommandFailed) {
                const err_output = git.getLastErrorOutput(allocator) catch null;
                defer if (err_output) |output| allocator.free(output);
                
                try colors.printError(stderr, "Failed to remove worktree", .{});
                if (err_output) |output| {
                    try stderr.print("{s}Git error:{s} {s}\n", .{ colors.yellow, colors.reset, output });
                }
            }
            return err;
        };
    } else {
        _ = git.exec(allocator, &.{ "worktree", "remove", worktree_path.? }) catch |err| {
            if (err == git.GitError.CommandFailed) {
                const err_output = git.getLastErrorOutput(allocator) catch null;
                defer if (err_output) |output| allocator.free(output);
                
                try colors.printError(stderr, "Failed to remove worktree", .{});
                if (err_output) |output| {
                    try stderr.print("{s}Git error:{s} {s}\n", .{ colors.yellow, colors.reset, output });
                    
                    // Check for common error patterns
                    if (std.mem.indexOf(u8, output, "contains modified or untracked files") != null) {
                        try stderr.print("{s}Tip:{s} Use -f/--force to remove worktrees with uncommitted changes\n", .{
                            colors.info_prefix, colors.reset
                        });
                        try stderr.print("{s}Warning:{s} This will discard all uncommitted changes!\n", .{
                            colors.warning_prefix, colors.reset
                        });
                    } else if (std.mem.indexOf(u8, output, "is a current working directory") != null) {
                        try stderr.print("{s}Tip:{s} You cannot remove the worktree you're currently in\n", .{
                            colors.info_prefix, colors.reset
                        });
                        try stderr.print("      Navigate to a different worktree first with 'git-wt go'\n", .{});
                    }
                }
            }
            return err;
        };
    }
    
    try colors.printSuccess(stdout, "✓ Worktree removed successfully", .{});
    
    // Ask about deleting the branch
    if (!non_interactive) {
        const prompt = try std.fmt.allocPrint(allocator, "\nWould you also like to delete the branch '{s}'?", .{branch_name});
        defer allocator.free(prompt);
        
        if (try input.confirm(prompt, false)) {
            try colors.printInfo(stdout, "Deleting branch...", .{});
            
            git.deleteBranch(allocator, branch_name, true) catch |err| {
                try colors.printError(stderr, "Failed to delete branch", .{});
                try stdout.print("{s}You may need to delete it manually with: git branch -D {s}{s}\n", .{ 
                    colors.info_prefix, branch_name, colors.reset 
                });
                return err;
            };
            
            try colors.printSuccess(stdout, "✓ Branch deleted successfully", .{});
        } else {
            try stdout.print("{s}Branch '{s}' was kept{s}\n", .{ colors.info_prefix, branch_name, colors.reset });
        }
    }
}

/// Execute remove command with interactive selection
pub fn executeInteractive(allocator: std.mem.Allocator, force_non_interactive: bool, force: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    // Get worktrees with modification times, excluding current
    const worktrees_with_time = try git.listWorktreesWithTime(allocator, true);
    defer git.freeWorktreesWithTime(allocator, worktrees_with_time);
    
    if (worktrees_with_time.len == 0) {
        try colors.printInfo(stdout, "No worktrees available to remove\n", .{});
        return;
    }
    
    
    // Check if we can use interactive mode
    const use_interactive = !force_non_interactive and interactive.isStdinTty() and interactive.isStdoutTty();
    
    var selected_idx: ?usize = null;
    
    if (use_interactive) {
        // Build list of options for interactive selection
        var options_list = std.ArrayList([]u8).init(allocator);
        defer options_list.deinit();
        defer for (options_list.items) |item| allocator.free(item);
        
        for (worktrees_with_time) |wt_info| {
            const timestamp = @divFloor(wt_info.mod_time, std.time.ns_per_s);
            const time_ago_seconds = @as(u64, @intCast(std.time.timestamp() - timestamp));
            const duration_str = try time.formatDuration(allocator, time_ago_seconds);
            defer allocator.free(duration_str);
            
            const option_text = try std.fmt.allocPrint(allocator, "{s}{s}{s} @ {s}{s}{s} - {s}{s} ago{s}", .{
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
        try colors.printInfo(stdout, "Select worktree to remove:\n", .{});
        
        const selection = try interactive.selectFromList(
            allocator,
            options_list.items,
            .{
                .show_instructions = true,
                .use_colors = true,
            },
        );
        
        selected_idx = selection;
    } else {
        // Number-based selection mode
        try colors.printInfo(stdout, "Available worktrees:\n", .{});
        
        for (worktrees_with_time, 1..) |wt_info, idx| {
            const timestamp = @divFloor(wt_info.mod_time, std.time.ns_per_s);
            const time_ago_seconds = @as(u64, @intCast(std.time.timestamp() - timestamp));
            const duration_str = try time.formatDuration(allocator, time_ago_seconds);
            defer allocator.free(duration_str);
            
            try stdout.print("  {s}{d}{s}) {s}{s}{s} @ {s}{s}{s}\n", .{
                colors.green,
                idx,
                colors.reset,
                colors.path_color,
                wt_info.display_name,
                colors.reset,
                colors.magenta,
                wt_info.worktree.branch,
                colors.reset,
            });
            
            // Format timestamp
            try stdout.print("     {s}Last modified:{s} {s} ago\n", .{
                colors.yellow,
                colors.reset,
                duration_str,
            });
        }
        
        const prompt = try std.fmt.allocPrint(allocator, "\n{s}Enter number to remove (or 'q' to quit):{s}", .{ colors.yellow, colors.reset });
        defer allocator.free(prompt);
        
        const response = try input.readLine(allocator, prompt);
        if (response) |resp| {
            defer allocator.free(resp);
            const trimmed = std.mem.trim(u8, resp, " \t\r\n");
            
            if (trimmed.len > 0 and (trimmed[0] == 'q' or trimmed[0] == 'Q')) {
                try colors.printInfo(stdout, "Cancelled\n", .{});
                return;
            }
            
            const selection = std.fmt.parseInt(usize, trimmed, 10) catch {
                try colors.printError(stderr, "Invalid selection", .{});
                return error.InvalidSelection;
            };
            
            if (selection < 1 or selection > worktrees_with_time.len) {
                try colors.printError(stderr, "Invalid selection", .{});
                return error.InvalidSelection;
            }
            
            selected_idx = selection - 1;
        }
    }
    
    // If a worktree was selected, remove it
    if (selected_idx) |idx| {
        const selected = worktrees_with_time[idx];
        try colors.printInfo(stdout, "Selected worktree: ", .{});
        try stdout.print("{s}{s}{s} @ {s}{s}{s}\n", .{
            colors.path_color,
            selected.display_name,
            colors.reset,
            colors.magenta,
            selected.worktree.branch,
            colors.reset,
        });
        
        // Call the regular execute function with the selected branch
        try execute(allocator, selected.worktree.branch, force_non_interactive, force);
    } else {
        try colors.printInfo(stdout, "Cancelled\n", .{});
    }
}