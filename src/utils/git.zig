const std = @import("std");
const process = std.process;
const fs = std.fs;

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
    defer allocator.free(result.stderr);
    
    if (result.term.Exited != 0) {
        allocator.free(result.stdout);
        return GitError.CommandFailed;
    }
    
    return result.stdout;
}

/// Get repository information
pub fn getRepoInfo(allocator: std.mem.Allocator) !RepoInfo {
    // Get the repository root
    const root = try exec(allocator, &.{ "rev-parse", "--show-toplevel" });
    defer allocator.free(root);
    
    // Remove trailing newline
    const root_trimmed = std.mem.trimRight(u8, root, "\n");
    const root_path = try allocator.dupe(u8, root_trimmed);
    
    // Get repository name
    const name = fs.path.basename(root_path);
    
    // Check if we're in a worktree
    const git_dir = try exec(allocator, &.{ "rev-parse", "--git-dir" });
    defer allocator.free(git_dir);
    
    const git_dir_trimmed = std.mem.trimRight(u8, git_dir, "\n");
    const is_worktree = !std.mem.endsWith(u8, git_dir_trimmed, "/.git");
    
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

/// Get list of worktrees
pub fn listWorktrees(allocator: std.mem.Allocator) ![]Worktree {
    const output = try exec(allocator, &.{ "worktree", "list", "--porcelain" });
    defer allocator.free(output);
    
    var worktrees = std.ArrayList(Worktree).init(allocator);
    var lines = std.mem.tokenize(u8, output, "\n");
    
    var current_path: ?[]const u8 = null;
    var current_commit: ?[]const u8 = null;
    var current_branch: ?[]const u8 = null;
    
    while (lines.next()) |line| {
        if (std.mem.startsWith(u8, line, "worktree ")) {
            // Save previous worktree if exists
            if (current_path) |path| {
                try worktrees.append(.{
                    .path = try allocator.dupe(u8, path),
                    .branch = if (current_branch) |b| try allocator.dupe(u8, b) else try allocator.dupe(u8, "detached"),
                    .commit = try allocator.dupe(u8, current_commit orelse "unknown"),
                });
            }
            current_path = line[9..]; // Skip "worktree "
            current_branch = null;
            current_commit = null;
        } else if (std.mem.startsWith(u8, line, "HEAD ")) {
            current_commit = line[5..]; // Skip "HEAD "
        } else if (std.mem.startsWith(u8, line, "branch ")) {
            current_branch = line[7..]; // Skip "branch "
        }
    }
    
    // Don't forget the last worktree
    if (current_path) |path| {
        try worktrees.append(.{
            .path = try allocator.dupe(u8, path),
            .branch = if (current_branch) |b| try allocator.dupe(u8, b) else try allocator.dupe(u8, "detached"),
            .commit = try allocator.dupe(u8, current_commit orelse "unknown"),
        });
    }
    
    return worktrees.toOwnedSlice();
}

/// Create a new worktree
pub fn createWorktree(allocator: std.mem.Allocator, path: []const u8, branch: []const u8) !void {
    _ = try exec(allocator, &.{ "worktree", "add", path, "-b", branch });
}

/// Remove a worktree
pub fn removeWorktree(allocator: std.mem.Allocator, path: []const u8) !void {
    _ = try exec(allocator, &.{ "worktree", "remove", path });
}

/// Get current branch name
pub fn getCurrentBranch(allocator: std.mem.Allocator) ![]u8 {
    const output = try exec(allocator, &.{ "branch", "--show-current" });
    // Remove trailing newline
    if (std.mem.endsWith(u8, output, "\n")) {
        output[output.len - 1] = 0;
        return output[0 .. output.len - 1];
    }
    return output;
}

/// Delete a branch
pub fn deleteBranch(allocator: std.mem.Allocator, branch: []const u8, force: bool) !void {
    const flag = if (force) "-D" else "-d";
    _ = try exec(allocator, &.{ "branch", flag, branch });
}

test "git exec" {
    const allocator = std.testing.allocator;
    
    // This should work in any git repository
    const version = try exec(allocator, &.{"--version"});
    defer allocator.free(version);
    
    try std.testing.expect(std.mem.startsWith(u8, version, "git version"));
}

test "RepoInfo parsing" {
    // This test would need to be run in a git repository
    // Skipping for now as it requires a specific environment
}