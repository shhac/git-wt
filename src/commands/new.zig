const std = @import("std");
const process = std.process;
const fs = std.fs;

const git = @import("../utils/git.zig");
const fs_utils = @import("../utils/fs.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");
const proc = @import("../utils/process.zig");
const validation = @import("../utils/validation.zig");

pub fn printHelp() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("Usage: git-wt new <branch-name>\n\n", .{});
    try stdout.print("Create a new git worktree with the specified branch name.\n\n", .{});
    try stdout.print("Arguments:\n", .{});
    try stdout.print("  <branch-name>    Name of the new branch (required)\n\n", .{});
    try stdout.print("Options:\n", .{});
    try stdout.print("  -h, --help       Show this help message\n", .{});
    try stdout.print("  -n, --non-interactive  Run without prompts\n\n", .{});
    try stdout.print("Examples:\n", .{});
    try stdout.print("  git-wt new feature-auth\n", .{});
    try stdout.print("  git-wt new bugfix-123\n", .{});
    try stdout.print("  git-wt new feature/ui-update    # Creates subdirectory structure\n", .{});
    try stdout.print("  git-wt new --non-interactive hotfix-security\n\n", .{});
    try stdout.print("This command will:\n", .{});
    try stdout.print("  1. Create a new worktree in ../repo-trees/branch-name\n", .{});
    try stdout.print("  2. Create and checkout the new branch\n", .{});
    try stdout.print("  3. Copy configuration files (.env, .claude, node_modules, etc.)\n", .{});
    try stdout.print("  4. Run nvm use if .nvmrc exists\n", .{});
    try stdout.print("  5. Install dependencies if yarn project detected\n", .{});
    try stdout.print("  6. Optionally start claude (interactive mode only)\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, branch_name: []const u8, non_interactive: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    // Validate branch name
    validation.validateBranchName(branch_name) catch |err| {
        try colors.printError(stderr, "Invalid branch name: {s}", .{validation.getValidationErrorMessage(err)});
        return err;
    };
    
    // Get repository info first (needed for other checks)
    const repo_info = git.getRepoInfo(allocator) catch |err| {
        try colors.printError(stderr, "Not in a git repository", .{});
        return err;
    };
    defer allocator.free(repo_info.root);
    defer if (repo_info.main_repo_root) |root| allocator.free(root);
    
    // Check if branch already exists
    if (try git.branchExists(allocator, branch_name)) {
        try colors.printError(stderr, "Branch '{s}' already exists", .{branch_name});
        return error.BranchAlreadyExists;
    }
    
    // Check if repository is in a clean state
    if (!try git.isRepositoryClean(allocator)) {
        try colors.printError(stderr, "Repository is not in a clean state (ongoing merge, rebase, etc.)", .{});
        return error.RepositoryNotClean;
    }
    
    // Construct worktree path
    const worktree_path = try fs_utils.constructWorktreePath(allocator, repo_info.root, branch_name);
    defer allocator.free(worktree_path);
    
    // Check if worktree path already exists
    if (fs_utils.pathExists(worktree_path)) {
        try colors.printError(stderr, "Worktree path already exists: {s}", .{worktree_path});
        return error.WorktreePathExists;
    }
    
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
            
            // Start claude in a detached process
            var claude_process = std.process.Child.init(&.{"claude"}, allocator);
            claude_process.stdin_behavior = .Ignore;
            claude_process.stdout_behavior = .Ignore;
            claude_process.stderr_behavior = .Ignore;
            
            claude_process.spawn() catch |err| {
                try colors.printError(stderr, "Failed to start claude: {}", .{err});
                // Don't fail the whole command if claude can't start
            };
        } else {
            try colors.printInfo(stdout, "Skipped starting claude", .{});
        }
    }
}