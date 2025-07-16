const std = @import("std");
const process = std.process;

const git = @import("../utils/git.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");

pub fn execute(allocator: std.mem.Allocator) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    // Get repository info
    const repo_info = git.getRepoInfo(allocator) catch |err| {
        try colors.printError(stderr, "Not in a git repository", .{});
        return err;
    };
    defer allocator.free(repo_info.root);
    defer if (repo_info.main_repo_root) |root| allocator.free(root);
    
    // Check if we're in a worktree
    if (!repo_info.is_worktree) {
        try colors.printError(stderr, "You are in the main repository, not a worktree", .{});
        try stdout.print("{s}Tip:{s} This command should be run from within a git worktree\n", .{ colors.warning_prefix, colors.reset });
        return error.NotInWorktree;
    }
    
    // Get the current branch name
    const current_branch = try git.getCurrentBranch(allocator);
    defer allocator.free(current_branch);
    
    // Show what we're about to do
    try stdout.print("{s}⚠️  About to remove worktree:{s}\n", .{ colors.warning_prefix, colors.reset });
    try stdout.print("   {s}Path:{s} {s}\n", .{ colors.path_color, colors.reset, repo_info.root });
    try stdout.print("   {s}Currently on branch:{s} {s}\n", .{ colors.path_color, colors.reset, current_branch });
    
    if (!try input.confirm("\nAre you sure you want to continue?", false)) {
        try colors.printInfo(stdout, "Cancelled", .{});
        return;
    }
    
    // Get the main repository path
    const main_repo = repo_info.main_repo_root orelse return error.NoMainRepo;
    
    // Change to the main repository
    try colors.printPath(stdout, "📁 Changing to main repository:", main_repo);
    try process.changeCurDir(main_repo);
    
    // Remove the worktree
    try colors.printInfo(stdout, "Removing worktree...", .{});
    
    git.removeWorktree(allocator, repo_info.root) catch |err| {
        try colors.printError(stderr, "Failed to remove worktree", .{});
        return err;
    };
    
    try colors.printSuccess(stdout, "Worktree removed successfully", .{});
    
    // Ask about deleting the branch
    const prompt = try std.fmt.allocPrint(allocator, "\nWould you also like to delete the branch '{s}'?", .{current_branch});
    defer allocator.free(prompt);
    
    if (try input.confirm(prompt, false)) {
        try colors.printInfo(stdout, "Deleting branch...", .{});
        
        git.deleteBranch(allocator, current_branch, true) catch {
            try colors.printError(stderr, "Failed to delete branch", .{});
            try stdout.print("{s}You may need to delete it manually{s}\n", .{ colors.warning_prefix, colors.reset });
            return;
        };
        
        try colors.printSuccess(stdout, "Branch deleted successfully", .{});
    } else {
        try stdout.print("{s}Branch '{s}' was kept{s}\n", .{ colors.info_prefix, current_branch, colors.reset });
    }
}