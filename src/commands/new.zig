const std = @import("std");
const process = std.process;
const fs = std.fs;

const git = @import("../utils/git.zig");
const fs_utils = @import("../utils/fs.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");
const proc = @import("../utils/process.zig");
const validation = @import("../utils/validation.zig");
const lock = @import("../utils/lock.zig");
const io = @import("../utils/io.zig");

pub fn printHelp() !void {
    const stdout = io.getStdOut();
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
    try stdout.print("  3. Copy configuration files (.env, .claude, etc.)\n\n", .{});
    try stdout.print("Note: Parent directory must exist, be writable, and not be inside\n", .{});
    try stdout.print("      the current repository. Paths are resolved to absolute paths.\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, branch_name: []const u8, _: bool, parent_dir: ?[]const u8) !void {
    const stdout = io.getStdOut();
    const stderr = io.getStdErr();
    
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
    
    // Check if branch already exists
    if (try git.branchExists(allocator, branch_name)) {
        try colors.printError(stderr, "Branch '{s}' already exists", .{branch_name});
        return error.BranchAlreadyExists;
    }
    
    // Check if we're in a bare repository
    if (try git.isBareRepository(allocator)) {
        try colors.printError(stderr, "Cannot create worktrees in a bare repository", .{});
        return error.BareRepository;
    }
    
    // Check if we're actually inside a work tree
    if (!try git.isInsideWorkTree(allocator)) {
        try colors.printError(stderr, "Not inside a git work tree", .{});
        return error.NotInWorkTree;
    }
    
    // Check if repository is in a clean state
    if (!try git.isRepositoryClean(allocator)) {
        const operation = try git.getCurrentOperation(allocator);
        defer if (operation) |op| allocator.free(op);
        
        if (operation) |op| {
            try colors.printError(stderr, "Cannot create worktree: {s} in progress", .{op});
            try stderr.print("{s}Tip:{s} Complete or abort the current {s} first\n", .{
                colors.info_prefix, colors.reset, op
            });
        } else {
            try colors.printError(stderr, "Repository is not in a clean state", .{});
        }
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
    
    // Check for case-insensitive conflicts on macOS/Windows
    const target_os = @import("builtin").target.os.tag;
    if (target_os == .macos or target_os == .windows) {
        const worktree_parent_dir = fs.path.dirname(worktree_path) orelse ".";
        const worktree_basename = fs.path.basename(worktree_path);
        
        // Only check if parent directory exists
        if (fs.cwd().openDir(worktree_parent_dir, .{ .iterate = true })) |_| {
            var dir_for_iter = try fs.cwd().openDir(worktree_parent_dir, .{ .iterate = true });
            defer dir_for_iter.close();
            
            var iter = dir_for_iter.iterate();
            while (try iter.next()) |entry| {
                if (std.ascii.eqlIgnoreCase(entry.name, worktree_basename)) {
                    try colors.printError(stderr, "Case-insensitive conflict: '{s}' already exists", .{entry.name});
                    try stderr.print("{s}Tip:{s} Use a different branch name to avoid conflicts\n", .{
                        colors.info_prefix, colors.reset
                    });
                    return error.CaseInsensitiveConflict;
                }
            }
        } else |_| {
            // Parent doesn't exist yet, no conflict possible
        }
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
    try colors.printDisplayPath(stdout, "Creating worktree for branch:", worktree_path, allocator);
    
    git.createWorktree(allocator, worktree_path, branch_name) catch |err| {
        const err_output = git.getLastErrorOutput(allocator) catch null;
        defer if (err_output) |output| allocator.free(output);
        
        try colors.printError(stderr, "Failed to create worktree", .{});
        if (err_output) |output| {
            try stderr.print("{s}Git error:{s} {s}\n", .{ colors.yellow, colors.reset, output });
        }
        try stderr.print("{s}Possible causes:{s}\n", .{ colors.info_prefix, colors.reset });
        try stderr.print("  - The branch name may contain invalid characters\n", .{});
        try stderr.print("  - The worktree path may not be accessible\n", .{});
        try stderr.print("  - You may not have write permissions\n", .{});
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
    try colors.printDisplayPath(stdout, "📁 Changed to worktree:", worktree_path, allocator);
}