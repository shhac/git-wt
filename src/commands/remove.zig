const std = @import("std");

const git = @import("../utils/git.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");

pub fn printHelp() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("Usage: git-wt rm <branch-name>\n\n", .{});
    try stdout.print("Remove a git worktree by branch name.\n\n", .{});
    try stdout.print("Arguments:\n", .{});
    try stdout.print("  <branch-name>    Name of the branch/worktree to remove (required)\n\n", .{});
    try stdout.print("Options:\n", .{});
    try stdout.print("  -h, --help       Show this help message\n", .{});
    try stdout.print("  -n, --non-interactive  Run without prompts\n", .{});
    try stdout.print("  -f, --force      Force removal even with uncommitted changes\n\n", .{});
    try stdout.print("Examples:\n", .{});
    try stdout.print("  git-wt rm feature-branch       # Remove feature-branch worktree\n", .{});
    try stdout.print("  git-wt rm feature/auth         # Remove worktree with slash in name\n", .{});
    try stdout.print("  git-wt rm test-branch -n       # Remove without prompts\n", .{});
    try stdout.print("  git-wt rm old-feature -f       # Force remove with uncommitted changes\n\n", .{});
    try stdout.print("This command will:\n", .{});
    try stdout.print("  1. Find the worktree for the specified branch\n", .{});
    try stdout.print("  2. Remove the worktree directory\n", .{});
    try stdout.print("  3. Optionally delete the associated branch\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, branch_name: []const u8, non_interactive: bool, force: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
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
        try stderr.print("{s}Error:{s} No worktree found for branch '{s}'\n", .{ 
            colors.error_prefix, colors.reset, branch_name 
        });
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
                try colors.printError(stderr, "Failed to remove worktree", .{});
            }
            return err;
        };
    } else {
        _ = git.exec(allocator, &.{ "worktree", "remove", worktree_path.? }) catch |err| {
            if (err == git.GitError.CommandFailed) {
                try colors.printError(stderr, "Failed to remove worktree", .{});
                try stderr.print("{s}Tip:{s} Use -f/--force to remove worktrees with uncommitted changes\n", .{
                    colors.info_prefix, colors.reset
                });
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