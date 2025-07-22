const std = @import("std");
const process = std.process;
const fs = std.fs;
const fs_utils = @import("fs.zig");

pub const GitError = error{
    NotInRepository,
    CommandFailed,
    InvalidOutput,
    WorktreeNotFound,
};

pub const RepoInfo = struct {
    root: []const u8,
    name: []const u8,
    is_worktree: bool,
    main_repo_root: ?[]const u8,
};

pub const Worktree = struct {
    path: []const u8,
    branch: []const u8,
    commit: []const u8,
    is_bare: bool = false,
    is_detached: bool = false,
    is_current: bool = false,
};

/// Helper to trim trailing newlines
fn trimNewline(str: []const u8) []const u8 {
    return std.mem.trimRight(u8, str, "\n");
}

// Thread-local storage for last git error - to be removed after refactoring
threadlocal var last_git_error: ?[]u8 = null;
threadlocal var last_git_error_allocator: ?std.mem.Allocator = null;

// Result type for git commands that can fail with error output
pub const GitResult = union(enum) {
    success: []u8,
    failure: struct {
        exit_code: u8,
        stderr: []u8,
    },
    
    pub fn deinit(self: GitResult, allocator: std.mem.Allocator) void {
        switch (self) {
            .success => |output| allocator.free(output),
            .failure => |err| allocator.free(err.stderr),
        }
    }
};

/// Execute a git command and return the output
pub fn exec(allocator: std.mem.Allocator, args: []const []const u8) ![]u8 {
    var argv = std.ArrayList([]const u8).init(allocator);
    defer argv.deinit();
    
    try argv.append("git");
    try argv.appendSlice(args);
    
    const result = try std.process.Child.run(.{
        .allocator = allocator,
        .argv = argv.items,
    });
    
    if (result.term.Exited != 0) {
        // Store the error output for later retrieval
        if (last_git_error) |err| {
            if (last_git_error_allocator) |alloc| {
                alloc.free(err);
            }
        }
        last_git_error = result.stderr;
        last_git_error_allocator = allocator;
        
        allocator.free(result.stdout);
        return GitError.CommandFailed;
    }
    
    allocator.free(result.stderr);
    return result.stdout;
}

/// Get the last git error output (if any) and clear the stored error
pub fn getLastErrorOutput(allocator: std.mem.Allocator) !?[]u8 {
    if (last_git_error) |err| {
        const trimmed = trimNewline(err);
        const result = try allocator.dupe(u8, trimmed);
        
        // Clear the stored error to prevent memory leak
        if (last_git_error_allocator) |alloc| {
            alloc.free(err);
        }
        last_git_error = null;
        last_git_error_allocator = null;
        
        return result;
    }
    return null;
}

/// Clean up thread-local error storage (call at program exit if needed)
pub fn cleanupErrorStorage() void {
    if (last_git_error) |err| {
        if (last_git_error_allocator) |alloc| {
            alloc.free(err);
        }
    }
    last_git_error = null;
    last_git_error_allocator = null;
}

/// Execute a git command and return trimmed output
pub fn execTrimmed(allocator: std.mem.Allocator, args: []const []const u8) ![]u8 {
    const output = try exec(allocator, args);
    defer allocator.free(output);
    const trimmed = trimNewline(output);
    return try allocator.dupe(u8, trimmed);
}

/// Get repository information
pub fn getRepoInfo(allocator: std.mem.Allocator) !RepoInfo {
    // Get the repository root
    const root_path = try execTrimmed(allocator, &.{ "rev-parse", "--show-toplevel" });
    
    // Get repository name
    const name = fs.path.basename(root_path);
    
    // Check if we're in a worktree
    const git_dir = try execTrimmed(allocator, &.{ "rev-parse", "--git-dir" });
    defer allocator.free(git_dir);
    
    // A worktree has a .git file, while main repo has .git directory
    // git rev-parse --git-dir returns:
    // - ".git" for main repository (relative path)
    // - absolute path like "/path/to/main/.git/worktrees/name" for worktrees
    const is_worktree = !std.mem.eql(u8, git_dir, ".git") and !std.mem.endsWith(u8, git_dir, "/.git");
    
    var main_repo_root: ?[]const u8 = null;
    if (is_worktree) {
        // Get main repository from worktree list
        const worktree_list = try exec(allocator, &.{"worktree", "list"});
        defer allocator.free(worktree_list);
        
        // First line is the main repository
        if (std.mem.indexOfScalar(u8, worktree_list, '\n')) |idx| {
            const first_line = worktree_list[0..idx];
            // Extract path (first field)
            if (std.mem.indexOfScalar(u8, first_line, ' ')) |space_idx| {
                main_repo_root = try allocator.dupe(u8, first_line[0..space_idx]);
            }
        }
    }
    
    return RepoInfo{
        .root = root_path,
        .name = name,
        .is_worktree = is_worktree,
        .main_repo_root = main_repo_root,
    };
}

/// Free worktree list memory
pub fn freeWorktrees(allocator: std.mem.Allocator, worktrees: []Worktree) void {
    for (worktrees) |wt| {
        allocator.free(wt.path);
        allocator.free(wt.branch);
        allocator.free(wt.commit);
    }
    allocator.free(worktrees);
}

// Large repository performance thresholds
const LARGE_REPO_THRESHOLD = 200; // Above this, use memory optimizations
const MAX_INTERACTIVE_ITEMS = 50;  // Max items to show in interactive mode

/// Count worktrees efficiently without loading all data
pub fn countWorktrees(allocator: std.mem.Allocator) !usize {
    const result = try exec(allocator, &.{ "worktree", "list" });
    defer allocator.free(result);
    
    var count: usize = 0;
    var lines = std.mem.tokenizeScalar(u8, result, '\n');
    while (lines.next() != null) count += 1;
    
    return count;
}

/// Find a specific worktree by branch name with early exit
pub fn findWorktreeByBranch(allocator: std.mem.Allocator, target_branch: []const u8) !?Worktree {
    const result = try exec(allocator, &.{ "worktree", "list", "--porcelain" });
    defer allocator.free(result);
    
    var current_path: ?[]const u8 = null;
    var current_branch: ?[]const u8 = null;
    var current_commit: ?[]const u8 = null;
    
    var lines = std.mem.tokenizeScalar(u8, result, '\n');
    while (lines.next()) |line| {
        if (std.mem.startsWith(u8, line, "worktree ")) {
            // Finalize previous worktree if we found a match
            if (current_branch != null and std.mem.eql(u8, current_branch.?, target_branch)) {
                return Worktree{
                    .path = try allocator.dupe(u8, current_path.?),
                    .branch = try allocator.dupe(u8, current_branch.?),
                    .commit = try allocator.dupe(u8, current_commit orelse ""),
                };
            }
            
            // Start new worktree
            const path = std.mem.trim(u8, line[9..], " \t\n\r");
            current_path = path;
            current_branch = null;
            current_commit = null;
        } else if (std.mem.startsWith(u8, line, "branch ")) {
            if (line.len > 7) {
                current_branch = std.mem.trim(u8, line[7..], " \t\n\r");
            }
        } else if (std.mem.startsWith(u8, line, "HEAD ")) {
            if (line.len > 5) {
                current_commit = std.mem.trim(u8, line[5..], " \t\n\r");
            }
        }
    }
    
    // Check final worktree
    if (current_branch != null and std.mem.eql(u8, current_branch.?, target_branch)) {
        return Worktree{
            .path = try allocator.dupe(u8, current_path.?),
            .branch = try allocator.dupe(u8, current_branch.?),
            .commit = try allocator.dupe(u8, current_commit orelse ""),
        };
    }
    
    return null; // Not found
}

/// Get list of worktrees with smart loading for large repositories
pub fn listWorktreesSmart(allocator: std.mem.Allocator, for_interactive: bool) ![]Worktree {
    if (for_interactive) {
        const count = countWorktrees(allocator) catch {
            // If count fails, fall back to normal loading
            return listWorktrees(allocator);
        };
        
        if (count > LARGE_REPO_THRESHOLD) {
            return listWorktreesLimited(allocator, MAX_INTERACTIVE_ITEMS);
        }
    }
    
    // Use existing implementation for normal cases
    return listWorktrees(allocator);
}

/// Get modification time for a path (returns 0 on error)
fn getPathModTime(allocator: std.mem.Allocator, path: []const u8) !i128 {
    _ = allocator; // unused but kept for API consistency
    const stat = try std.fs.cwd().statFile(path);
    return stat.mtime;
}

/// Check if a given worktree path is the current directory
fn checkIsCurrentWorktree(cwd_path: []const u8, worktree_path: []const u8) bool {
    // Exact match
    if (std.mem.eql(u8, cwd_path, worktree_path)) return true;
    
    // Check if cwd is a subdirectory of this worktree
    // Ensure we have a proper path separator after the prefix
    if (std.mem.startsWith(u8, cwd_path, worktree_path)) {
        if (cwd_path.len > worktree_path.len and cwd_path[worktree_path.len] == '/') {
            return true;
        }
    }
    return false;
}

/// Get limited list of worktrees for large repository optimization
fn listWorktreesLimited(allocator: std.mem.Allocator, max_items: usize) ![]Worktree {
    const output = try exec(allocator, &.{ "worktree", "list", "--porcelain" });
    defer allocator.free(output);
    
    // Get current directory to mark current worktree
    const cwd_path = try std.process.getCwdAlloc(allocator);
    defer allocator.free(cwd_path);
    
    var worktrees = std.ArrayList(Worktree).init(allocator);
    errdefer {
        for (worktrees.items) |wt| {
            allocator.free(wt.path);
            allocator.free(wt.branch);
            allocator.free(wt.commit);
        }
        worktrees.deinit();
    }
    
    var current_path: ?[]const u8 = null;
    var current_branch: ?[]const u8 = null;
    var current_commit: ?[]const u8 = null;
    var is_bare = false;
    var is_detached = false;
    
    var lines = std.mem.tokenizeScalar(u8, output, '\n');
    var processed_count: usize = 0;
    
    while (lines.next()) |line| {
        if (processed_count >= max_items) {
            break; // Stop processing when we hit the limit
        }
        
        if (std.mem.startsWith(u8, line, "worktree ")) {
            // Finalize previous worktree if we have one
            if (current_path != null) {
                const is_current = checkIsCurrentWorktree(cwd_path, current_path.?);
                
                try worktrees.append(Worktree{
                    .path = try allocator.dupe(u8, current_path.?),
                    .branch = try allocator.dupe(u8, current_branch orelse "HEAD"),
                    .commit = try allocator.dupe(u8, current_commit orelse ""),
                    .is_bare = is_bare,
                    .is_detached = is_detached,
                    .is_current = is_current,
                });
                
                processed_count += 1;
            }
            
            // Start new worktree
            const path = std.mem.trim(u8, line[9..], " \t\n\r");
            current_path = path;
            current_branch = null;
            current_commit = null;
            is_bare = false;
            is_detached = false;
        } else if (std.mem.startsWith(u8, line, "branch ")) {
            if (line.len > 7) {
                current_branch = std.mem.trim(u8, line[7..], " \t\n\r");
            }
        } else if (std.mem.startsWith(u8, line, "HEAD ")) {
            if (line.len > 5) {
                current_commit = std.mem.trim(u8, line[5..], " \t\n\r");
            }
        } else if (std.mem.eql(u8, line, "bare")) {
            is_bare = true;
        } else if (std.mem.eql(u8, line, "detached")) {
            is_detached = true;
        }
    }
    
    // Process final worktree if we haven't hit the limit
    if (current_path != null and processed_count < max_items) {
        const is_current = checkIsCurrentWorktree(cwd_path, current_path.?);
        
        try worktrees.append(Worktree{
            .path = try allocator.dupe(u8, current_path.?),
            .branch = try allocator.dupe(u8, current_branch orelse "HEAD"),
            .commit = try allocator.dupe(u8, current_commit orelse ""),
            .is_bare = is_bare,
            .is_detached = is_detached,
            .is_current = is_current,
        });
    }
    
    return worktrees.toOwnedSlice();
}

/// Get list of worktrees
pub fn listWorktrees(allocator: std.mem.Allocator) ![]Worktree {
    const output = try exec(allocator, &.{ "worktree", "list", "--porcelain" });
    defer allocator.free(output);
    
    // Get current directory to mark current worktree
    const cwd_path = try std.process.getCwdAlloc(allocator);
    defer allocator.free(cwd_path);
    
    var worktrees = std.ArrayList(Worktree).init(allocator);
    var lines = std.mem.tokenizeScalar(u8, output, '\n');
    
    var current_path: ?[]const u8 = null;
    var current_commit: ?[]const u8 = null;
    var current_branch: ?[]const u8 = null;
    var is_bare = false;
    var is_detached = false;
    
    while (lines.next()) |line| {
        if (std.mem.startsWith(u8, line, "worktree ")) {
            // Save previous worktree if exists
            if (current_path) |path| {
                // Check if this is the current worktree
                // We need to check if cwd is within this worktree path
                const is_current = blk: {
                    // Exact match
                    if (std.mem.eql(u8, cwd_path, path)) break :blk true;
                    
                    // Check if cwd is a subdirectory of this worktree
                    // Ensure we have a proper path separator after the prefix
                    if (std.mem.startsWith(u8, cwd_path, path)) {
                        if (cwd_path.len > path.len and cwd_path[path.len] == '/') {
                            break :blk true;
                        }
                    }
                    break :blk false;
                };
                
                try worktrees.append(.{
                    .path = try allocator.dupe(u8, path),
                    .branch = if (current_branch) |b| blk: {
                        // Remove refs/heads/ prefix if present
                        if (std.mem.startsWith(u8, b, "refs/heads/")) {
                            break :blk try allocator.dupe(u8, b[11..]);
                        }
                        break :blk try allocator.dupe(u8, b);
                    } else try allocator.dupe(u8, "HEAD"),
                    .commit = try allocator.dupe(u8, current_commit orelse "unknown"),
                    .is_bare = is_bare,
                    .is_detached = is_detached,
                    .is_current = is_current,
                });
            }
            current_path = line[9..]; // Skip "worktree "
            current_branch = null;
            current_commit = null;
            is_bare = false;
            is_detached = false;
        } else if (std.mem.startsWith(u8, line, "HEAD ")) {
            current_commit = line[5..]; // Skip "HEAD "
        } else if (std.mem.startsWith(u8, line, "branch ")) {
            current_branch = line[7..]; // Skip "branch "
        } else if (std.mem.eql(u8, line, "bare")) {
            is_bare = true;
        } else if (std.mem.eql(u8, line, "detached")) {
            is_detached = true;
        }
    }
    
    // Don't forget the last worktree
    if (current_path) |path| {
        const is_current = blk: {
            // Exact match
            if (std.mem.eql(u8, cwd_path, path)) break :blk true;
            
            // Check if cwd is a subdirectory of this worktree
            // Ensure we have a proper path separator after the prefix
            if (std.mem.startsWith(u8, cwd_path, path)) {
                if (cwd_path.len > path.len and cwd_path[path.len] == '/') {
                    break :blk true;
                }
            }
            break :blk false;
        };
        
        try worktrees.append(.{
            .path = try allocator.dupe(u8, path),
            .branch = if (current_branch) |b| blk: {
                // Remove refs/heads/ prefix if present
                if (std.mem.startsWith(u8, b, "refs/heads/")) {
                    break :blk try allocator.dupe(u8, b[11..]);
                }
                break :blk try allocator.dupe(u8, b);
            } else try allocator.dupe(u8, "HEAD"),
            .commit = try allocator.dupe(u8, current_commit orelse "unknown"),
            .is_bare = is_bare,
            .is_detached = is_detached,
            .is_current = is_current,
        });
    }
    
    return worktrees.toOwnedSlice();
}

/// Create a new worktree
pub fn createWorktree(allocator: std.mem.Allocator, path: []const u8, branch: []const u8) !void {
    const result = try exec(allocator, &.{ "worktree", "add", path, "-b", branch });
    defer allocator.free(result);
}

/// Remove a worktree
pub fn removeWorktree(allocator: std.mem.Allocator, path: []const u8) !void {
    const result = try exec(allocator, &.{ "worktree", "remove", path });
    defer allocator.free(result);
}

/// Get current branch name
pub fn getCurrentBranch(allocator: std.mem.Allocator) ![]u8 {
    return try execTrimmed(allocator, &.{ "branch", "--show-current" });
}

/// Delete a branch
pub fn deleteBranch(allocator: std.mem.Allocator, branch: []const u8, force: bool) !void {
    const flag = if (force) "-D" else "-d";
    const result = try exec(allocator, &.{ "branch", flag, branch });
    defer allocator.free(result);
}

/// Check if repository is in a clean state (no ongoing rebase, merge, etc.)
pub fn isRepositoryClean(allocator: std.mem.Allocator) !bool {
    return isRepositoryCleanWithGitDir(allocator, null);
}

/// Check if repository is in a clean state with optional git dir
pub fn isRepositoryCleanWithGitDir(allocator: std.mem.Allocator, git_dir_opt: ?[]const u8) !bool {
    // Check for various git state files that indicate ongoing operations
    const git_dir = if (git_dir_opt) |dir| 
        try allocator.dupe(u8, dir) 
    else 
        execTrimmed(allocator, &.{ "rev-parse", "--git-dir" }) catch return false;
    defer allocator.free(git_dir);
    
    const state_files = [_][]const u8{
        "MERGE_HEAD",
        "CHERRY_PICK_HEAD", 
        "REVERT_HEAD",
        "BISECT_LOG",
        "rebase-merge",
        "rebase-apply",
    };
    
    for (state_files) |state_file| {
        const state_path = std.fmt.allocPrint(allocator, "{s}/{s}", .{ git_dir, state_file }) catch continue;
        defer allocator.free(state_path);
        
        // Check if file or directory exists
        std.fs.cwd().access(state_path, .{}) catch continue;
        // If we can access it, repo is not clean
        return false;
    }
    
    return true;
}

/// Check if there are uncommitted changes in the repository
pub fn hasUncommittedChanges(allocator: std.mem.Allocator) !bool {
    const status_output = exec(allocator, &.{ "status", "--porcelain" }) catch return true;
    defer allocator.free(status_output);
    
    // If output is empty, no uncommitted changes
    return status_output.len > 0;
}

/// Check if a branch already exists
pub fn branchExists(allocator: std.mem.Allocator, branch: []const u8) !bool {
    const ref_name = try std.fmt.allocPrint(allocator, "refs/heads/{s}", .{branch});
    defer allocator.free(ref_name);
    
    const result = exec(allocator, &.{ "show-ref", "--verify", "--quiet", ref_name }) catch return false;
    defer allocator.free(result);
    
    return true;
}

/// Check if we're in a bare repository
pub fn isBareRepository(allocator: std.mem.Allocator) !bool {
    const result = try execTrimmed(allocator, &.{ "rev-parse", "--is-bare-repository" });
    defer allocator.free(result);
    return std.mem.eql(u8, result, "true");
}

/// Check if we're inside a git work tree
pub fn isInsideWorkTree(allocator: std.mem.Allocator) !bool {
    const result = try execTrimmed(allocator, &.{ "rev-parse", "--is-inside-work-tree" });
    defer allocator.free(result);
    return std.mem.eql(u8, result, "true");
}

/// Get what operation is currently in progress (if any)
pub fn getCurrentOperation(allocator: std.mem.Allocator) !?[]const u8 {
    const git_dir = try execTrimmed(allocator, &.{ "rev-parse", "--git-dir" });
    defer allocator.free(git_dir);
    
    // Check for merge
    const merge_path = try std.fmt.allocPrint(allocator, "{s}/MERGE_HEAD", .{git_dir});
    defer allocator.free(merge_path);
    if (fs.cwd().access(merge_path, .{})) |_| {
        return try allocator.dupe(u8, "merge");
    } else |_| {}
    
    // Check for cherry-pick
    const cherry_path = try std.fmt.allocPrint(allocator, "{s}/CHERRY_PICK_HEAD", .{git_dir});
    defer allocator.free(cherry_path);
    if (fs.cwd().access(cherry_path, .{})) |_| {
        return try allocator.dupe(u8, "cherry-pick");
    } else |_| {}
    
    // Check for revert
    const revert_path = try std.fmt.allocPrint(allocator, "{s}/REVERT_HEAD", .{git_dir});
    defer allocator.free(revert_path);
    if (fs.cwd().access(revert_path, .{})) |_| {
        return try allocator.dupe(u8, "revert");
    } else |_| {}
    
    // Check for rebase
    const rebase_merge_path = try std.fmt.allocPrint(allocator, "{s}/rebase-merge", .{git_dir});
    defer allocator.free(rebase_merge_path);
    if (fs.cwd().access(rebase_merge_path, .{})) |_| {
        return try allocator.dupe(u8, "rebase");
    } else |_| {}
    
    const rebase_apply_path = try std.fmt.allocPrint(allocator, "{s}/rebase-apply", .{git_dir});
    defer allocator.free(rebase_apply_path);
    if (fs.cwd().access(rebase_apply_path, .{})) |_| {
        return try allocator.dupe(u8, "rebase");
    } else |_| {}
    
    // Check for bisect
    const bisect_path = try std.fmt.allocPrint(allocator, "{s}/BISECT_LOG", .{git_dir});
    defer allocator.free(bisect_path);
    if (fs.cwd().access(bisect_path, .{})) |_| {
        return try allocator.dupe(u8, "bisect");
    } else |_| {}
    
    return null;
}

/// Get the current worktree path (handles being in subdirectories)
pub fn getCurrentWorktree(allocator: std.mem.Allocator) !?[]const u8 {
    // Get the repository info to find out if we're in a worktree
    const repo_info = getRepoInfo(allocator) catch return null;
    defer allocator.free(repo_info.root);
    defer if (repo_info.main_repo_root) |root| allocator.free(root);
    
    // If we're in a worktree, return its root path
    if (repo_info.is_worktree) {
        return try allocator.dupe(u8, repo_info.root);
    }
    
    // If we're in the main repository, return null to indicate main
    return null;
}

test "git exec" {
    const allocator = std.testing.allocator;
    
    // Skip test if git is not available
    const version = exec(allocator, &.{"--version"}) catch |err| {
        if (err == error.FileNotFound) return; // Git not installed
        return err;
    };
    defer allocator.free(version);
    
    try std.testing.expect(std.mem.startsWith(u8, version, "git version"));
}

test "trimNewline" {
    // Test with newline
    const with_nl = "hello\n";
    const trimmed1 = trimNewline(with_nl);
    try std.testing.expectEqualStrings("hello", trimmed1);
    
    // Test without newline
    const without_nl = "world";
    const trimmed2 = trimNewline(without_nl);
    try std.testing.expectEqualStrings("world", trimmed2);
    
    // Test empty string
    const empty = "";
    const trimmed3 = trimNewline(empty);
    try std.testing.expectEqualStrings("", trimmed3);
    
    // Test multiple newlines
    const multi_nl = "test\n\n\n";
    const trimmed4 = trimNewline(multi_nl);
    try std.testing.expectEqualStrings("test", trimmed4);
}

test "execTrimmed" {
    const allocator = std.testing.allocator;
    
    // Skip test if git is not available
    const version = execTrimmed(allocator, &.{"--version"}) catch |err| {
        if (err == error.FileNotFound) return; // Git not installed
        return err;
    };
    defer allocator.free(version);
    
    try std.testing.expect(std.mem.startsWith(u8, version, "git version"));
    try std.testing.expect(!std.mem.endsWith(u8, version, "\n"));
}

test "RepoInfo struct" {
    // Test that RepoInfo can be created and used
    const info = RepoInfo{
        .root = "/path/to/repo",
        .name = "repo",
        .is_worktree = true,
        .main_repo_root = "/path/to/main",
    };
    
    try std.testing.expectEqualStrings("/path/to/repo", info.root);
    try std.testing.expectEqualStrings("repo", info.name);
    try std.testing.expect(info.is_worktree);
    try std.testing.expectEqualStrings("/path/to/main", info.main_repo_root.?);
}

test "is_current detection logic" {
    // Test the logic used in listWorktrees for is_current detection
    const TestCase = struct {
        cwd_path: []const u8,
        worktree_path: []const u8,
        expected: bool,
        description: []const u8,
    };
    
    const test_cases = [_]TestCase{
        // Exact match cases
        .{
            .cwd_path = "/Users/paul/projects/my-repo",
            .worktree_path = "/Users/paul/projects/my-repo",
            .expected = true,
            .description = "exact match should be current",
        },
        .{
            .cwd_path = "/Users/paul/projects/my-repo-trees/feature",
            .worktree_path = "/Users/paul/projects/my-repo-trees/feature",
            .expected = true,
            .description = "exact match in worktree should be current",
        },
        
        // Subdirectory cases
        .{
            .cwd_path = "/Users/paul/projects/my-repo/src/utils",
            .worktree_path = "/Users/paul/projects/my-repo",
            .expected = true,
            .description = "subdirectory should be current",
        },
        .{
            .cwd_path = "/Users/paul/projects/my-repo-trees/feature/src",
            .worktree_path = "/Users/paul/projects/my-repo-trees/feature",
            .expected = true,
            .description = "subdirectory in worktree should be current",
        },
        
        // False positive cases that should NOT match
        .{
            .cwd_path = "/Users/paul/projects/my-repo",
            .worktree_path = "/Users/paul/projects/my-repo-trees/feature",
            .expected = false,
            .description = "main repo should not match worktree with similar prefix",
        },
        .{
            .cwd_path = "/Users/paul/projects/my-repo-trees/feature",
            .worktree_path = "/Users/paul/projects/my-repo",
            .expected = false,
            .description = "worktree should not match main repo",
        },
        .{
            .cwd_path = "/Users/paul/projects/my-repo2",
            .worktree_path = "/Users/paul/projects/my-repo",
            .expected = false,
            .description = "different repo with similar name should not match",
        },
        .{
            .cwd_path = "/Users/paul/projects/my-repo-something",
            .worktree_path = "/Users/paul/projects/my-repo",
            .expected = false,
            .description = "repo with longer similar name should not match",
        },
    };
    
    for (test_cases) |tc| {
        const is_current = std.mem.eql(u8, tc.cwd_path, tc.worktree_path) or 
            (std.mem.startsWith(u8, tc.cwd_path, tc.worktree_path) and 
             tc.cwd_path.len > tc.worktree_path.len and 
             tc.cwd_path[tc.worktree_path.len] == '/');
             
        try std.testing.expectEqual(tc.expected, is_current);
    }
}

test "Worktree struct" {
    // Test that Worktree can be created and used
    const wt = Worktree{
        .path = "/path/to/worktree",
        .branch = "feature-branch",
        .commit = "abc123",
        .is_bare = false,
        .is_detached = false,
        .is_current = true,
    };
    
    try std.testing.expectEqualStrings("/path/to/worktree", wt.path);
    try std.testing.expectEqualStrings("feature-branch", wt.branch);
    try std.testing.expectEqualStrings("abc123", wt.commit);
    try std.testing.expect(!wt.is_bare);
    try std.testing.expect(!wt.is_detached);
    try std.testing.expect(wt.is_current);
}

/// Worktree with modification time and display name
pub const WorktreeWithTime = struct {
    worktree: Worktree,
    mod_time: i128,
    display_name: []const u8,
};

/// List worktrees with modification times, sorted by newest first
/// Caller owns returned memory and must call freeWorktreesWithTime
/// Get worktrees with time information, optimized for large repositories
pub fn listWorktreesWithTimeSmart(allocator: std.mem.Allocator, exclude_current: bool, for_interactive: bool) ![]WorktreeWithTime {
    const worktrees = try listWorktreesSmart(allocator, for_interactive);
    defer freeWorktrees(allocator, worktrees);
    
    // Use arena allocator for easier cleanup
    var arena = std.heap.ArenaAllocator.init(allocator);
    defer arena.deinit();
    const arena_allocator = arena.allocator();
    
    var worktrees_list = std.ArrayList(WorktreeWithTime).init(allocator);
    errdefer {
        for (worktrees_list.items) |wt| {
            allocator.free(wt.worktree.path);
            allocator.free(wt.worktree.branch);
            allocator.free(wt.worktree.commit);
            allocator.free(wt.display_name);
        }
        worktrees_list.deinit();
    }
    
    for (worktrees) |wt| {
        if (exclude_current and wt.is_current) {
            continue;
        }
        
        // Get modification time (0 if unavailable)
        const mod_time = getPathModTime(arena_allocator, wt.path) catch 0;
        
        // Create display name
        const display_name = try fs_utils.extractDisplayPath(allocator, wt.path);
        errdefer allocator.free(display_name);
        
        const path = try allocator.dupe(u8, wt.path);
        errdefer allocator.free(path);
        
        const branch = try allocator.dupe(u8, wt.branch);
        errdefer allocator.free(branch);
        
        const commit = try allocator.dupe(u8, wt.commit);
        errdefer allocator.free(commit);
        
        // Create the worktree copy
        const wt_item = WorktreeWithTime{
            .worktree = Worktree{
                .path = path,
                .branch = branch,
                .commit = commit,
                .is_bare = wt.is_bare,
                .is_detached = wt.is_detached,
                .is_current = wt.is_current,
            },
            .mod_time = mod_time,
            .display_name = display_name,
        };
        
        try worktrees_list.append(wt_item);
    }
    
    // Sort by modification time (most recent first)
    const items = try worktrees_list.toOwnedSlice();
    std.mem.sort(WorktreeWithTime, items, {}, struct {
        fn lessThan(_: void, a: WorktreeWithTime, b: WorktreeWithTime) bool {
            return a.mod_time > b.mod_time;
        }
    }.lessThan);
    return items;
}

pub fn listWorktreesWithTime(allocator: std.mem.Allocator, exclude_current: bool) ![]WorktreeWithTime {
    const worktrees = try listWorktrees(allocator);
    defer freeWorktrees(allocator, worktrees);
    
    // Use arena allocator for easier cleanup
    var arena = std.heap.ArenaAllocator.init(allocator);
    defer arena.deinit();
    const arena_allocator = arena.allocator();
    
    var worktrees_list = std.ArrayList(WorktreeWithTime).init(allocator);
    defer worktrees_list.deinit();
    
    // Get modification times for each worktree
    for (worktrees) |wt| {
        // Skip current worktree if requested
        if (exclude_current and wt.is_current) continue;
        
        const stat = std.fs.cwd().statFile(wt.path) catch continue;
        
        // Build worktree item using arena for temporary allocations
        var wt_item: WorktreeWithTime = undefined;
        
        // Determine display name (allocated from arena temporarily)
        const temp_display_name = if (std.mem.indexOf(u8, wt.path, "-trees") == null)
            try arena_allocator.dupe(u8, "[main]")
        else blk: {
            const basename = std.fs.path.basename(wt.path);
            break :blk try arena_allocator.dupe(u8, basename);
        };
        
        // Now allocate everything from the main allocator in a safe order
        const display_name = try allocator.dupe(u8, temp_display_name);
        errdefer allocator.free(display_name);
        
        const path = try allocator.dupe(u8, wt.path);
        errdefer allocator.free(path);
        
        const branch = try allocator.dupe(u8, wt.branch);
        errdefer allocator.free(branch);
        
        const commit = try allocator.dupe(u8, wt.commit);
        errdefer allocator.free(commit);
        
        // Create the worktree copy
        wt_item = .{
            .worktree = Worktree{
                .path = path,
                .branch = branch,
                .commit = commit,
                .is_bare = wt.is_bare,
                .is_detached = wt.is_detached,
                .is_current = wt.is_current,
            },
            .mod_time = stat.mtime,
            .display_name = display_name,
        };
        
        // Add to list - if this fails, the errdefers above will clean up
        try worktrees_list.append(wt_item);
    }
    
    const result = try worktrees_list.toOwnedSlice();
    errdefer {
        // If toOwnedSlice succeeds but something after fails, clean up
        for (result) |wt| {
            allocator.free(wt.display_name);
            allocator.free(wt.worktree.path);
            allocator.free(wt.worktree.branch);
            allocator.free(wt.worktree.commit);
        }
        allocator.free(result);
    }
    
    // Sort by modification time (newest first)
    std.mem.sort(WorktreeWithTime, result, {}, struct {
        fn lessThan(_: void, a: WorktreeWithTime, b: WorktreeWithTime) bool {
            return a.mod_time > b.mod_time;
        }
    }.lessThan);
    
    return result;
}

/// Free memory allocated by listWorktreesWithTime
pub fn freeWorktreesWithTime(allocator: std.mem.Allocator, worktrees: []WorktreeWithTime) void {
    for (worktrees) |wt| {
        allocator.free(wt.display_name);
        allocator.free(wt.worktree.path);
        allocator.free(wt.worktree.branch);
        allocator.free(wt.worktree.commit);
    }
    allocator.free(worktrees);
}

test "parseWorktreeList porcelain output" {
    // Test parsing of git worktree list --porcelain output
    const allocator = std.testing.allocator;
    
    // Test helper function to parse porcelain output
    const parseOutput = struct {
        fn parse(alloc: std.mem.Allocator, output: []const u8, cwd: []const u8) ![]Worktree {
            var worktrees = std.ArrayList(Worktree).init(alloc);
            var lines = std.mem.tokenizeScalar(u8, output, '\n');
            
            var current_path: ?[]const u8 = null;
            var current_commit: ?[]const u8 = null;
            var current_branch: ?[]const u8 = null;
            var is_bare = false;
            var is_detached = false;
            
            while (lines.next()) |line| {
                if (std.mem.startsWith(u8, line, "worktree ")) {
                    if (current_path) |path| {
                        const is_current = std.mem.eql(u8, cwd, path) or 
                            (std.mem.startsWith(u8, cwd, path) and 
                             cwd.len > path.len and 
                             cwd[path.len] == '/');
                        
                        try worktrees.append(.{
                            .path = try alloc.dupe(u8, path),
                            .branch = if (current_branch) |b| blk: {
                                if (std.mem.startsWith(u8, b, "refs/heads/")) {
                                    break :blk try alloc.dupe(u8, b[11..]);
                                }
                                break :blk try alloc.dupe(u8, b);
                            } else try alloc.dupe(u8, "HEAD"),
                            .commit = try alloc.dupe(u8, current_commit orelse "unknown"),
                            .is_bare = is_bare,
                            .is_detached = is_detached,
                            .is_current = is_current,
                        });
                    }
                    current_path = line[9..];
                    current_branch = null;
                    current_commit = null;
                    is_bare = false;
                    is_detached = false;
                } else if (std.mem.startsWith(u8, line, "HEAD ")) {
                    current_commit = line[5..];
                } else if (std.mem.startsWith(u8, line, "branch ")) {
                    current_branch = line[7..];
                } else if (std.mem.eql(u8, line, "bare")) {
                    is_bare = true;
                } else if (std.mem.eql(u8, line, "detached")) {
                    is_detached = true;
                }
            }
            
            // Last worktree
            if (current_path) |path| {
                const is_current = std.mem.eql(u8, cwd, path) or 
                    (std.mem.startsWith(u8, cwd, path) and 
                     cwd.len > path.len and 
                     cwd[path.len] == '/');
                
                try worktrees.append(.{
                    .path = try alloc.dupe(u8, path),
                    .branch = if (current_branch) |b| blk: {
                        if (std.mem.startsWith(u8, b, "refs/heads/")) {
                            break :blk try alloc.dupe(u8, b[11..]);
                        }
                        break :blk try alloc.dupe(u8, b);
                    } else try alloc.dupe(u8, "HEAD"),
                    .commit = try alloc.dupe(u8, current_commit orelse "unknown"),
                    .is_bare = is_bare,
                    .is_detached = is_detached,
                    .is_current = is_current,
                });
            }
            
            return worktrees.toOwnedSlice();
        }
    }.parse;
    
    // Test case 1: Multiple worktrees with refs/heads prefix
    const output1 =
        \\worktree /home/user/project
        \\HEAD abcd1234
        \\branch refs/heads/main
        \\
        \\worktree /home/user/project-trees/feature
        \\HEAD efgh5678
        \\branch refs/heads/feature-branch
    ;
    
    const worktrees1 = try parseOutput(allocator, output1, "/home/user/project");
    defer freeWorktrees(allocator, worktrees1);
    
    try std.testing.expectEqual(@as(usize, 2), worktrees1.len);
    try std.testing.expectEqualStrings("main", worktrees1[0].branch);
    try std.testing.expectEqualStrings("feature-branch", worktrees1[1].branch);
    try std.testing.expect(worktrees1[0].is_current);
    try std.testing.expect(!worktrees1[1].is_current);
    
    // Test case 2: Detached HEAD
    const output2 =
        \\worktree /home/user/project
        \\HEAD abcd1234
        \\detached
    ;
    
    const worktrees2 = try parseOutput(allocator, output2, "/home/user/project");
    defer freeWorktrees(allocator, worktrees2);
    
    try std.testing.expectEqual(@as(usize, 1), worktrees2.len);
    try std.testing.expectEqualStrings("HEAD", worktrees2[0].branch);
    try std.testing.expect(worktrees2[0].is_detached);
    
    // Test case 3: Current detection from subdirectory
    const worktrees3 = try parseOutput(allocator, output1, "/home/user/project/src/utils");
    defer freeWorktrees(allocator, worktrees3);
    
    try std.testing.expect(worktrees3[0].is_current); // Should still detect main as current
    try std.testing.expect(!worktrees3[1].is_current);
}

test "countWorktrees parsing" {
    // Mock git output for counting
    const mock_output = 
        \\/home/user/project
        \\/home/user/project-trees/feature-branch
        \\/home/user/project-trees/bugfix
    ;
    
    // Test the parsing logic (we can't easily test the actual exec call)
    var count: usize = 0;
    var lines = std.mem.tokenizeScalar(u8, mock_output, '\n');
    while (lines.next() != null) count += 1;
    
    try std.testing.expectEqual(@as(usize, 3), count);
}

test "findWorktreeByBranch parsing" {
    const allocator = std.testing.allocator;
    
    // Mock git --porcelain output
    const mock_output = 
        \\worktree /home/user/project
        \\HEAD abcd1234
        \\
        \\worktree /home/user/project-trees/feature-branch
        \\branch refs/heads/feature-branch
        \\HEAD efgh5678
        \\
        \\worktree /home/user/project-trees/bugfix
        \\branch refs/heads/bugfix
        \\HEAD ijkl9012
    ;
    
    // Test parsing logic to find specific branch
    var current_path: ?[]const u8 = null;
    var current_branch: ?[]const u8 = null;
    var current_commit: ?[]const u8 = null;
    var found_worktree: ?Worktree = null;
    
    const target_branch = "refs/heads/feature-branch";
    
    var lines = std.mem.tokenizeScalar(u8, mock_output, '\n');
    while (lines.next()) |line| {
        if (std.mem.startsWith(u8, line, "worktree ")) {
            // Check if previous worktree was our target
            if (current_branch != null and std.mem.eql(u8, current_branch.?, target_branch)) {
                found_worktree = Worktree{
                    .path = try allocator.dupe(u8, current_path.?),
                    .branch = try allocator.dupe(u8, current_branch.?),
                    .commit = try allocator.dupe(u8, current_commit orelse ""),
                    .is_bare = false,
                    .is_detached = false,
                    .is_current = false,
                };
                break;
            }
            
            const path = std.mem.trim(u8, line[9..], " \t\n\r");
            current_path = path;
            current_branch = null;
            current_commit = null;
        } else if (std.mem.startsWith(u8, line, "branch ")) {
            if (line.len > 7) {
                current_branch = std.mem.trim(u8, line[7..], " \t\n\r");
            }
        } else if (std.mem.startsWith(u8, line, "HEAD ")) {
            if (line.len > 5) {
                current_commit = std.mem.trim(u8, line[5..], " \t\n\r");
            }
        }
    }
    
    // Check final worktree too
    if (found_worktree == null and current_branch != null and std.mem.eql(u8, current_branch.?, target_branch)) {
        found_worktree = Worktree{
            .path = try allocator.dupe(u8, current_path.?),
            .branch = try allocator.dupe(u8, current_branch.?),
            .commit = try allocator.dupe(u8, current_commit orelse ""),
            .is_bare = false,
            .is_detached = false,
            .is_current = false,
        };
    }
    
    try std.testing.expect(found_worktree != null);
    if (found_worktree) |wt| {
        defer allocator.free(wt.path);
        defer allocator.free(wt.branch);
        defer allocator.free(wt.commit);
        
        try std.testing.expectEqualStrings("/home/user/project-trees/feature-branch", wt.path);
        try std.testing.expectEqualStrings("refs/heads/feature-branch", wt.branch);
        try std.testing.expectEqualStrings("efgh5678", wt.commit);
    }
    
    // Test not found case
    const target_not_found = "refs/heads/nonexistent";
    var found_not_exist: ?Worktree = null;
    
    var lines2 = std.mem.tokenizeScalar(u8, mock_output, '\n');
    current_path = null;
    current_branch = null;
    current_commit = null;
    
    while (lines2.next()) |line| {
        if (std.mem.startsWith(u8, line, "worktree ")) {
            if (current_branch != null and std.mem.eql(u8, current_branch.?, target_not_found)) {
                found_not_exist = Worktree{
                    .path = try allocator.dupe(u8, current_path.?),
                    .branch = try allocator.dupe(u8, current_branch.?),
                    .commit = try allocator.dupe(u8, current_commit orelse ""),
                    .is_bare = false,
                    .is_detached = false,
                    .is_current = false,
                };
                break;
            }
            current_path = std.mem.trim(u8, line[9..], " \t\n\r");
            current_branch = null;
            current_commit = null;
        } else if (std.mem.startsWith(u8, line, "branch ")) {
            if (line.len > 7) {
                current_branch = std.mem.trim(u8, line[7..], " \t\n\r");
            }
        }
    }
    
    try std.testing.expect(found_not_exist == null);
}