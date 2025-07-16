const std = @import("std");
const process = std.process;
const fs = std.fs;

const git = @import("../utils/git.zig");
const fs_utils = @import("../utils/fs.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");
const proc = @import("../utils/process.zig");

pub fn execute(allocator: std.mem.Allocator, branch_name: []const u8, non_interactive: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    // Get repository info
    const repo_info = git.getRepoInfo(allocator) catch |err| {
        try colors.printError(stderr, "Not in a git repository", .{});
        return err;
    };
    defer allocator.free(repo_info.root);
    defer if (repo_info.main_repo_root) |root| allocator.free(root);
    
    // Construct worktree path
    const worktree_path = try fs_utils.constructWorktreePath(allocator, repo_info.root, branch_name);
    defer allocator.free(worktree_path);
    
    // Create the worktree
    try colors.printPath(stdout, "Creating worktree at:", worktree_path);
    
    git.createWorktree(allocator, worktree_path, branch_name) catch |err| {
        try colors.printError(stderr, "Failed to create worktree", .{});
        return err;
    };
    
    try colors.printSuccess(stdout, "Worktree created successfully", .{});
    
    // Change to the new worktree directory
    try process.changeCurDir(worktree_path);
    try colors.printPath(stdout, "📁 Changed directory to:", worktree_path);
    
    // Check for .nvmrc and run nvm use if it exists
    if (try fs_utils.hasNvmrc(worktree_path)) {
        try colors.printInfo(stdout, "📋 Found .nvmrc, running nvm use...", .{});
        _ = try proc.runWithOutput(allocator, &.{ "nvm", "use" });
    }
    
    // Check for package.json with yarn
    if (try fs_utils.hasNodeProject(worktree_path) and try fs_utils.usesYarn(allocator, worktree_path)) {
        try colors.printInfo(stdout, "📦 Found package.json with yarn, running yarn install...", .{});
        
        if (try proc.runSilent(allocator, &.{"yarn"})) {
            try colors.printSuccess(stdout, "Dependencies installed", .{});
        }
    }
    
    // Copy configuration files
    try colors.printInfo(stdout, "📋 Copying local configuration files...", .{});
    try fs_utils.copyConfigFiles(allocator, repo_info.root, worktree_path);
    
    // Ask if user wants to run claude (skip in non-interactive mode)
    if (!non_interactive) {
        if (try input.confirm("\nWould you like to start claude?", true)) {
            try colors.printSuccess(stdout, "🚀 Starting claude...", .{});
            
            // Start claude
            var claude_process = std.process.Child.init(&.{"claude"}, allocator);
            try claude_process.spawn();
        } else {
            try colors.printInfo(stdout, "Skipped starting claude", .{});
        }
    }
}