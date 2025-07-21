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
    try stdout.print("  -n, --non-interactive  Run without prompts\n", .{});
    try stdout.print("  -p, --parent-dir <path>  Use custom parent directory for worktree\n", .{});
    try stdout.print("                           (default: ../repo-trees/)\n\n", .{});
    try stdout.print("Examples:\n", .{});
    try stdout.print("  git-wt new feature-auth\n", .{});
    try stdout.print("  git-wt new bugfix-123\n", .{});
    try stdout.print("  git-wt new feature/ui-update    # Creates subdirectory structure\n", .{});
    try stdout.print("  git-wt new feature --parent-dir ~/worktrees\n", .{});
    try stdout.print("  git-wt new hotfix -p /tmp/quick-fix\n", .{});
    try stdout.print("  git-wt new --non-interactive hotfix-security\n\n", .{});
    try stdout.print("This command will:\n", .{});
    try stdout.print("  1. Create a new worktree in ../repo-trees/branch-name\n", .{});
    try stdout.print("  2. Create and checkout the new branch\n", .{});
    try stdout.print("  3. Copy configuration files (.env, .claude, node_modules, etc.)\n", .{});
    try stdout.print("  4. Run nvm use if .nvmrc exists\n", .{});
    try stdout.print("  5. Install dependencies if yarn project detected\n", .{});
    try stdout.print("  6. Optionally start claude (interactive mode only)\n\n", .{});
    try stdout.print("Note: Parent directory must exist, be writable, and not be inside\n", .{});
    try stdout.print("      the current repository. Paths are resolved to absolute paths.\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, branch_name: []const u8, non_interactive: bool, parent_dir: ?[]const u8) !void {
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
    
    // Validate parent directory if provided
    const validated_parent = if (parent_dir) |dir| blk: {
        const abs_parent = validation.validateParentDir(allocator, dir, repo_info) catch |err| {
            switch (err) {
                validation.ParentDirError.ParentDirNotFound => {
                    try colors.printError(stderr, "Parent directory does not exist", .{});
                    try stderr.print("{s}Tip:{s} Create the directory first: mkdir -p {s}\n", .{
                        colors.info_prefix, colors.reset, dir
                    });
                },
                validation.ParentDirError.ParentDirNotDirectory => {
                    try colors.printError(stderr, "Parent path is not a directory", .{});
                },
                validation.ParentDirError.ParentDirNotWritable => {
                    try colors.printError(stderr, "Parent directory is not writable", .{});
                },
                validation.ParentDirError.ParentDirInsideRepo => {
                    try colors.printError(stderr, "Parent directory cannot be inside the repository", .{});
                },
                validation.ParentDirError.PathTraversalAttempt => {
                    try colors.printError(stderr, "Path traversal attempts are not allowed", .{});
                },
                validation.ParentDirError.InvalidPath => {
                    try colors.printError(stderr, "Invalid parent directory path", .{});
                },
                else => try colors.printError(stderr, "Error: {s}", .{@errorName(err)}),
            }
            return err;
        };
        break :blk abs_parent;
    } else null;
    defer if (validated_parent) |p| allocator.free(p);
    
    // Construct worktree path
    const worktree_path = try fs_utils.constructWorktreePath(allocator, repo_info.root, repo_info.name, branch_name, validated_parent);
    defer allocator.free(worktree_path);
    
    // Check if worktree path already exists
    if (fs_utils.pathExists(worktree_path)) {
        try colors.printError(stderr, "Worktree path already exists: {s}", .{worktree_path});
        return error.WorktreePathExists;
    }
    
    // Track whether we need to clean up on failure
    var worktree_created = false;
    var parent_dir_created = false;
    
    // Check if parent directory needs to be created (for branches with slashes)
    const worktree_parent = fs.path.dirname(worktree_path);
    if (worktree_parent) |parent| {
        if (!fs_utils.pathExists(parent)) {
            parent_dir_created = true;
        }
    }
    
    // Set up cleanup on failure
    errdefer {
        if (worktree_created) {
            // Try to remove the worktree using git
            git.removeWorktree(allocator, worktree_path) catch |err| {
                // If git removal fails, try manual cleanup
                std.log.warn("Failed to remove worktree via git: {}", .{err});
                fs.cwd().deleteTree(worktree_path) catch |del_err| {
                    std.log.warn("Failed to manually delete worktree: {}", .{del_err});
                };
            };
        } else if (parent_dir_created) {
            // Only clean up parent directory if we created it and worktree wasn't created
            if (worktree_parent) |parent| {
                fs.cwd().deleteDir(parent) catch |err| {
                    std.log.warn("Failed to clean up parent directory: {}", .{err});
                };
            }
        }
    }
    
    // Create the worktree
    try colors.printPath(stdout, "Creating worktree at:", worktree_path);
    
    git.createWorktree(allocator, worktree_path, branch_name) catch |err| {
        try colors.printError(stderr, "Failed to create worktree", .{});
        return err;
    };
    worktree_created = true;
    
    try colors.printSuccess(stdout, "Worktree created successfully", .{});
    
    // Copy configuration files BEFORE changing directory
    try colors.printInfo(stdout, "📋 Copying local configuration files...", .{});
    fs_utils.copyConfigFiles(allocator, repo_info.root, worktree_path) catch |err| {
        try colors.printError(stderr, "Failed to copy configuration files: {}", .{err});
        // Continue anyway - this is not fatal
    };
    
    // Change to the new worktree directory
    try process.changeCurDir(worktree_path);
    try colors.printPath(stdout, "📁 Changed directory to:", worktree_path);
    
    // Check for .nvmrc and run nvm use if it exists
    if (try fs_utils.hasNvmrc(worktree_path)) {
        try colors.printInfo(stdout, "📋 Found .nvmrc, running nvm use...", .{});
        _ = proc.runWithOutput(allocator, &.{ "nvm", "use" }) catch |err| {
            try colors.printError(stderr, "Failed to run nvm use: {}", .{err});
            // Continue anyway - this is not fatal
        };
    }
    
    // Check for package.json with yarn
    if (try fs_utils.hasNodeProject(worktree_path) and try fs_utils.usesYarn(allocator, worktree_path)) {
        try colors.printInfo(stdout, "📦 Found package.json with yarn, running yarn install...", .{});
        
        if (try proc.runSilent(allocator, &.{"yarn"})) {
            try colors.printSuccess(stdout, "Dependencies installed", .{});
        }
    }
    
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