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
    is_bare: bool = false,
    is_detached: bool = false,
    is_current: bool = false,
};

/// Helper to trim trailing newlines
fn trimNewline(str: []const u8) []const u8 {
    return std.mem.trimRight(u8, str, "\n");
}

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
                const is_current = std.mem.eql(u8, cwd_path, path) or 
                    (std.mem.startsWith(u8, cwd_path, path) and 
                     cwd_path.len > path.len and 
                     cwd_path[path.len] == '/');
                
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
        const is_current = std.mem.eql(u8, cwd_path, path) or 
            (std.mem.startsWith(u8, cwd_path, path) and 
             cwd_path.len > path.len and 
             cwd_path[path.len] == '/');
        
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
    // Check for various git state files that indicate ongoing operations
    const git_dir = execTrimmed(allocator, &.{ "rev-parse", "--git-dir" }) catch return false;
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
    
    // This should work in any git repository
    const version = try exec(allocator, &.{"--version"});
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
    
    // Test that it trims output
    const version = try execTrimmed(allocator, &.{"--version"});
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
pub fn listWorktreesWithTime(allocator: std.mem.Allocator, exclude_current: bool) ![]WorktreeWithTime {
    const worktrees = try listWorktrees(allocator);
    defer freeWorktrees(allocator, worktrees);
    
    var worktrees_list = std.ArrayList(WorktreeWithTime).init(allocator);
    errdefer {
        for (worktrees_list.items) |wt| {
            allocator.free(wt.display_name);
            allocator.free(wt.worktree.path);
            allocator.free(wt.worktree.branch);
            allocator.free(wt.worktree.commit);
        }
        worktrees_list.deinit();
    }
    
    // Get modification times for each worktree
    for (worktrees) |wt| {
        // Skip current worktree if requested
        if (exclude_current and wt.is_current) continue;
        
        const stat = std.fs.cwd().statFile(wt.path) catch continue;
        
        // Determine display name
        const display_name = if (std.mem.indexOf(u8, wt.path, "-trees") == null)
            try allocator.dupe(u8, "[main]")
        else blk: {
            const basename = std.fs.path.basename(wt.path);
            break :blk try allocator.dupe(u8, basename);
        };
        
        // Duplicate worktree data since original will be freed
        const wt_copy = Worktree{
            .path = try allocator.dupe(u8, wt.path),
            .branch = try allocator.dupe(u8, wt.branch),
            .commit = try allocator.dupe(u8, wt.commit),
            .is_bare = wt.is_bare,
            .is_detached = wt.is_detached,
            .is_current = wt.is_current,
        };
        
        try worktrees_list.append(.{
            .worktree = wt_copy,
            .mod_time = stat.mtime,
            .display_name = display_name,
        });
    }
    
    const result = try worktrees_list.toOwnedSlice();
    
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