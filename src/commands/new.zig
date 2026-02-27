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
const interactive = @import("../utils/interactive.zig");
const io = @import("../utils/io.zig");
const mode_mod = @import("../utils/mode.zig");

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

pub fn execute(allocator: std.mem.Allocator, branch_name: []const u8, _: bool, parent_dir: ?[]const u8, current_mode: mode_mod.Mode) !void {
    const stdout = io.getStdOut();
    const stderr = io.getStdErr();

    // In bare-piped mode, route informational output to stderr
    // so only the raw worktree path goes to stdout for scripting
    const is_bare_piped = current_mode.isBare() and !interactive.isStdoutTty();
    const info_writer = if (is_bare_piped) stderr else stdout;
    
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
    try worktree_lock.acquireWithUserFeedback(30000, stderr);
    
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
                else => try colors.printError(stderr, "{s}", .{@errorName(err)}),
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
    try colors.printDisplayPath(info_writer, "Creating worktree for branch:", worktree_path, allocator);
    
    const create_result = try git.createWorktree(allocator, worktree_path, branch_name);
    defer create_result.deinit(allocator);
    
    switch (create_result) {
        .success => {},
        .failure => |err| {
            const trimmed_err = git.trimNewline(err.stderr);
            try colors.printError(stderr, "Failed to create worktree", .{});
            try stderr.print("{s}Git error:{s} {s}\n", .{ colors.yellow, colors.reset, trimmed_err });
            try stderr.print("{s}Possible causes:{s}\n", .{ colors.info_prefix, colors.reset });
            try stderr.print("  - The branch name may contain invalid characters\n", .{});
            try stderr.print("  - The worktree path may not be accessible\n", .{});
            try stderr.print("  - You may not have write permissions\n", .{});
            return error.CreateWorktreeFailed;
        },
    }
    worktree_created = true;
    
    try colors.printSuccess(info_writer, "Worktree created successfully", .{});
    
    // Copy configuration files BEFORE changing directory
    try colors.printInfo(info_writer, "📋 Copying local configuration files...", .{});
    fs_utils.copyConfigFiles(allocator, repo_info.root, worktree_path) catch |err| {
        try colors.printError(stderr, "Failed to copy configuration files: {}", .{err});
        // Continue anyway - this is not fatal
    };
    
    // Output worktree path based on mode
    if (current_mode.isWrapper()) {
        // Wrapper mode: shell alias handles navigation via go command
        try colors.printDisplayPath(info_writer, "📁 Changed to worktree:", worktree_path, allocator);
    } else if (interactive.isStdoutTty()) {
        // Bare TTY: confirmation on stderr with copy-paste hint
        try colors.printDisplayPath(stderr, "📁 Created worktree:", worktree_path, allocator);
        try stderr.print("\x1b[33m→\x1b[0m cd '{s}'\n", .{worktree_path});
    } else {
        // Bare piped: raw path on stdout for scripting
        try stdout.print("{s}\n", .{worktree_path});
    }
}