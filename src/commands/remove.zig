const std = @import("std");

const git = @import("../utils/git.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");
const interactive = @import("../utils/interactive.zig");
const time = @import("../utils/time.zig");
const lock = @import("../utils/lock.zig");
const validation = @import("../utils/validation.zig");
const fs_utils = @import("../utils/fs.zig");

pub fn printHelp() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("Usage: git-wt rm [branch-name...]\n\n", .{});
    try stdout.print("Remove one or more git worktrees by branch name.\n\n", .{});
    try stdout.print("Arguments:\n", .{});
    try stdout.print("  [branch-name...]  Names of branches/worktrees to remove (optional)\n", .{});
    try stdout.print("                    If not provided, shows interactive multi-selection\n\n", .{});
    try stdout.print("Options:\n", .{});
    try stdout.print("  -h, --help        Show this help message\n", .{});
    try stdout.print("  -n, --non-interactive  Run without prompts\n", .{});
    try stdout.print("  --no-tty          Force number-based selection (disable arrow keys)\n", .{});
    try stdout.print("  -f, --force       Force removal even with uncommitted changes\n\n", .{});
    try stdout.print("Examples:\n", .{});
    try stdout.print("  git-wt rm                        # Interactive multi-selection\n", .{});
    try stdout.print("  git-wt rm feature-branch         # Remove single worktree\n", .{});
    try stdout.print("  git-wt rm branch1 branch2 branch3 # Remove multiple worktrees\n", .{});
    try stdout.print("  git-wt rm feature/auth test-*    # Remove worktrees with special names\n", .{});
    try stdout.print("  git-wt rm test-branch -n         # Remove without prompts\n", .{});
    try stdout.print("  git-wt rm old-feature -f         # Force remove with uncommitted changes\n", .{});
    try stdout.print("  git-wt rm --no-tty               # Use number-based selection\n\n", .{});
    try stdout.print("Interactive mode:\n", .{});
    try stdout.print("  ↑/↓       Navigate selection\n", .{});
    try stdout.print("  Space     Toggle selection (☑/☐)\n", .{});
    try stdout.print("  Enter     Confirm current selection\n", .{});
    try stdout.print("  ESC/Q     Cancel operation\n\n", .{});
    try stdout.print("This command will:\n", .{});
    try stdout.print("  1. Find worktrees for the specified branches\n", .{});
    try stdout.print("  2. Remove the worktree directories\n", .{});
    try stdout.print("  3. Optionally delete the associated branches\n", .{});
    try stdout.print("\nNote: The current worktree cannot be removed.\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, branch_name: []const u8, non_interactive: bool, force: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    // Validate branch name
    validation.validateBranchName(branch_name) catch |err| {
        try colors.printError(stderr, "Invalid branch name: {s}", .{validation.getValidationErrorMessage(err)});
        return err;
    };
    
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
    
    // Get all worktrees (optimized search or full load)
    var worktree_path: ?[]const u8 = null;
    var target_worktree_opt: ?git.Worktree = null;
    
    // Try fast branch search first for large repositories
    if (git.findWorktreeByBranch(allocator, branch_name)) |maybe_target_wt| {
        if (maybe_target_wt) |target_wt| {
            target_worktree_opt = target_wt;
            worktree_path = target_wt.path;
        }
    } else |_| {
        // Fast search failed, continue with fallback logic
    }
    
    // If fast search didn't work, load all worktrees for fallback logic and error handling
    const worktrees = if (worktree_path == null) blk: {
        break :blk try git.listWorktrees(allocator);
    } else null;
    defer if (worktrees) |wts| git.freeWorktrees(allocator, wts);
    
    if (worktrees) |wts| {
        // Find the worktree for the specified branch
        // Try both the original branch name and sanitized version for special characters
        const sanitized_branch = try fs_utils.sanitizeBranchPath(allocator, branch_name);
        defer allocator.free(sanitized_branch);
        
        for (wts) |wt| {
            // Try direct match first (most common case)
            if (std.mem.eql(u8, wt.branch, branch_name)) {
                worktree_path = wt.path;
                break;
            }
            
            // Try match with sanitized branch name (for branches with special characters)
            if (std.mem.eql(u8, wt.branch, sanitized_branch)) {
                worktree_path = wt.path;
                break;
            }
            
            // Try reverse: unsanitize the stored branch name and compare
            // This handles cases where the stored name is sanitized
            const unsanitized_stored = fs_utils.unsanitizeBranchPath(allocator, wt.branch) catch continue;
            defer allocator.free(unsanitized_stored);
            if (std.mem.eql(u8, unsanitized_stored, branch_name)) {
                worktree_path = wt.path;
                break;
            }
        }
    }
    
    // Cleanup for optimized search result
    defer if (target_worktree_opt) |wt| {
        allocator.free(wt.path);
        allocator.free(wt.branch);
        allocator.free(wt.commit);
    };
    
    const confirmed_worktree_path = worktree_path orelse {
        try colors.printError(stderr, "No worktree found for branch '{s}'", .{branch_name});
        try stderr.print("{s}Tip:{s} Use 'git-wt list' to see available worktrees\n", .{
            colors.info_prefix, colors.reset
        });
        
        // Try to find similar branch names (including unsanitized versions)
        var similar_branches = std.ArrayList([]const u8).init(allocator);
        defer similar_branches.deinit();
        
        for (worktrees orelse &.{}) |wt| {
            // Check direct branch name match
            if (std.ascii.indexOfIgnoreCase(wt.branch, branch_name) != null) {
                try similar_branches.append(wt.branch);
                continue;
            }
            
            // Check unsanitized branch name match (for display purposes)
            const unsanitized_stored = fs_utils.unsanitizeBranchPath(allocator, wt.branch) catch continue;
            defer allocator.free(unsanitized_stored);
            if (std.ascii.indexOfIgnoreCase(unsanitized_stored, branch_name) != null) {
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
    };
    
    // Show what we're about to remove
    try stdout.print("{s}⚠️  About to remove worktree:{s}\n", .{ colors.warning_prefix, colors.reset });
    
    const display_path = try fs_utils.extractDisplayPath(allocator, confirmed_worktree_path);
    defer allocator.free(display_path);
    
    try stdout.print("   {s}Branch:{s} {s}\n", .{ colors.yellow, colors.reset, branch_name });
    try stdout.print("   {s}Worktree:{s} {s}\n", .{ colors.yellow, colors.reset, display_path });
    
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
        _ = git.exec(allocator, &.{ "worktree", "remove", "--force", confirmed_worktree_path }) catch |err| {
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
        _ = git.exec(allocator, &.{ "worktree", "remove", confirmed_worktree_path }) catch |err| {
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

/// Execute remove command for multiple branches
pub fn executeMultiple(allocator: std.mem.Allocator, branch_names: []const []const u8, non_interactive: bool, force: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    if (branch_names.len == 0) return;
    
    // Show summary of what we're about to remove
    try colors.printInfo(stdout, "About to remove worktrees:\n", .{});
    for (branch_names) |branch| {
        try stdout.print("  - {s}{s}{s}\n", .{ colors.yellow, branch, colors.reset });
    }
    
    // Confirm removal (unless non-interactive or force)
    if (!non_interactive and !force) {
        const prompt = try std.fmt.allocPrint(allocator, "\nAre you sure you want to remove {d} worktree{s}?", .{ branch_names.len, if (branch_names.len == 1) "" else "s" });
        defer allocator.free(prompt);
        
        if (!try input.confirm(prompt, false)) {
            try colors.printInfo(stdout, "Cancelled", .{});
            return;
        }
    }
    
    // Process each branch
    var failed_count: usize = 0;
    var success_count: usize = 0;
    var failed_branches = std.ArrayList([]const u8).init(allocator);
    defer failed_branches.deinit();
    
    for (branch_names) |branch| {
        try stdout.print("\n{s}Removing worktree for branch '{s}'...{s}\n", .{
            colors.info_prefix, branch, colors.reset
        });
        
        // Execute single branch removal
        execute(allocator, branch, true, force) catch |err| {
            failed_count += 1;
            try failed_branches.append(branch);
            try colors.printError(stderr, "Failed to remove worktree for branch '{s}': {}", .{ branch, err });
            continue;
        };
        
        success_count += 1;
        try colors.printSuccess(stdout, "✓ Removed worktree for '{s}'", .{branch});
    }
    
    // Summary
    try stdout.print("\n{s}=== SUMMARY ==={s}\n", .{ colors.yellow, colors.reset });
    try colors.printSuccess(stdout, "✓ Successfully removed: {d} worktree{s}", .{ success_count, if (success_count == 1) "" else "s" });
    
    if (failed_count > 0) {
        try colors.printError(stderr, "✗ Failed to remove: {d} worktree{s}", .{ failed_count, if (failed_count == 1) "" else "s" });
        try stderr.print("{s}Failed branches:{s}\n", .{ colors.yellow, colors.reset });
        for (failed_branches.items) |branch| {
            try stderr.print("  - {s}\n", .{branch});
        }
    }
}

/// Execute remove command with interactive selection
pub fn executeInteractive(allocator: std.mem.Allocator, force_non_interactive: bool, force: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    // Get worktrees with modification times, excluding current
    const worktrees_with_time = try git.listWorktreesWithTimeSmart(allocator, true, true);
    defer git.freeWorktreesWithTime(allocator, worktrees_with_time);
    
    if (worktrees_with_time.len == 0) {
        try colors.printInfo(stdout, "No worktrees available to remove\n", .{});
        return;
    }
    
    
    // Check if we can use interactive mode
    const use_interactive = !force_non_interactive and interactive.isStdinTty() and interactive.isStdoutTty();
    
    var selected_indices: ?[]usize = null;
    defer if (selected_indices) |indices| allocator.free(indices);
    
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
        try colors.printInfo(stdout, "Select worktree(s) to remove:\n", .{});
        
        const selection = try interactive.selectMultipleFromList(
            allocator,
            options_list.items,
            .{
                .show_instructions = true,
                .use_colors = true,
                .multi_select = true,
            },
        );
        
        selected_indices = selection;
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
        
        const prompt = try std.fmt.allocPrint(allocator, "\n{s}Enter numbers to remove (space-separated, or 'q' to quit):{s}", .{ colors.yellow, colors.reset });
        defer allocator.free(prompt);
        
        const response = try input.readLine(allocator, prompt);
        if (response) |resp| {
            defer allocator.free(resp);
            const trimmed = std.mem.trim(u8, resp, " \t\r\n");
            
            if (trimmed.len > 0 and (trimmed[0] == 'q' or trimmed[0] == 'Q')) {
                try colors.printInfo(stdout, "Cancelled\n", .{});
                return;
            }
            
            // Parse multiple numbers
            var indices = std.ArrayList(usize).init(allocator);
            var it = std.mem.tokenizeAny(u8, trimmed, " \t,");
            while (it.next()) |token| {
                const selection = std.fmt.parseInt(usize, token, 10) catch {
                    try colors.printError(stderr, "Invalid selection: '{s}'", .{token});
                    return error.InvalidSelection;
                };
                
                if (selection < 1 or selection > worktrees_with_time.len) {
                    try colors.printError(stderr, "Selection out of range: {d}", .{selection});
                    return error.InvalidSelection;
                }
                
                const idx = selection - 1;
                // Check for duplicates
                var already_selected = false;
                for (indices.items) |existing| {
                    if (existing == idx) {
                        already_selected = true;
                        break;
                    }
                }
                if (!already_selected) {
                    try indices.append(idx);
                }
            }
            
            if (indices.items.len > 0) {
                selected_indices = try indices.toOwnedSlice();
            }
        }
    }
    
    // If worktrees were selected, remove them
    if (selected_indices) |indices| {
        if (indices.len == 0) {
            try colors.printInfo(stdout, "No worktrees selected\n", .{});
            return;
        }
        
        // Show selected worktrees
        try colors.printInfo(stdout, "Selected worktree{s}:\n", .{if (indices.len == 1) "" else "s"});
        var branch_names = std.ArrayList([]const u8).init(allocator);
        defer branch_names.deinit();
        
        for (indices) |idx| {
            const selected = worktrees_with_time[idx];
            try stdout.print("  - {s}{s}{s} @ {s}{s}{s}\n", .{
                colors.path_color,
                selected.display_name,
                colors.reset,
                colors.magenta,
                selected.worktree.branch,
                colors.reset,
            });
            try branch_names.append(selected.worktree.branch);
        }
        
        // Call the multiple execute function with selected branches
        if (branch_names.items.len == 1) {
            try execute(allocator, branch_names.items[0], force_non_interactive, force);
        } else {
            try executeMultiple(allocator, branch_names.items, force_non_interactive, force);
        }
    } else {
        try colors.printInfo(stdout, "Cancelled\n", .{});
    }
}