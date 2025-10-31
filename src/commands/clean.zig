const std = @import("std");

const git = @import("../utils/git.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");
const lock = @import("../utils/lock.zig");
const io = @import("../utils/io.zig");
const process = @import("../utils/process.zig");

pub fn printHelp() !void {
    const stdout = io.getStdOut();
    try stdout.print("Usage: git-wt clean [options]\n\n", .{});
    try stdout.print("Remove all worktrees for deleted branches.\n\n", .{});
    try stdout.print("Options:\n", .{});
    try stdout.print("  -h, --help        Show this help message\n", .{});
    try stdout.print("  -n, --dry-run     Show what would be cleaned without removing\n", .{});
    try stdout.print("  -f, --force       Skip confirmation prompt\n\n", .{});
    try stdout.print("Examples:\n", .{});
    try stdout.print("  git-wt clean               # List and confirm before removal\n", .{});
    try stdout.print("  git-wt clean --dry-run     # Show what would be removed\n", .{});
    try stdout.print("  git-wt clean --force       # Remove without confirmation\n\n", .{});
    try stdout.print("This command will:\n", .{});
    try stdout.print("  1. Find all worktrees whose branches have been deleted\n", .{});
    try stdout.print("  2. List them for review\n", .{});
    try stdout.print("  3. Remove them after confirmation (unless --dry-run)\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, dry_run: bool, force: bool) !void {
    const stdout = io.getStdOut();
    const stderr = io.getStdErr();

    // Get repository info for lock path
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

    // Get all worktrees
    const worktrees = try git.listWorktrees(allocator);
    defer {
        for (worktrees) |wt| {
            allocator.free(wt.path);
            allocator.free(wt.branch);
            allocator.free(wt.commit);
        }
        allocator.free(worktrees);
    }

    // Find worktrees with deleted branches
    var to_clean = std.ArrayList(git.Worktree).empty;
    defer to_clean.deinit(allocator);

    for (worktrees) |wt| {
        // Skip current worktree
        if (wt.is_current) continue;

        // Check if branch still exists
        const branch_exists = git.branchExists(allocator, wt.branch) catch false;
        if (!branch_exists) {
            // Branch is deleted, add to cleanup list
            try to_clean.append(allocator, wt);
        }
    }

    // Report findings
    if (to_clean.items.len == 0) {
        try colors.printSuccess(stdout, "✓ No worktrees need cleaning", .{});
        try stdout.print("All worktree branches still exist.\n", .{});
        return;
    }

    try stdout.print("\n{s}Found {d} worktree(s) with deleted branches:{s}\n\n", .{
        colors.yellow,
        to_clean.items.len,
        colors.reset,
    });

    // List worktrees to be cleaned
    for (to_clean.items) |wt| {
        try stdout.print("  {s}{s}{s} @ {s}{s}{s}\n", .{
            colors.red,
            wt.branch,
            colors.reset,
            colors.path_color,
            wt.path,
            colors.reset,
        });
    }

    // In dry-run mode, just exit
    if (dry_run) {
        try stdout.print("\n{s}Dry run - no changes made{s}\n", .{
            colors.yellow,
            colors.reset,
        });
        return;
    }

    // Confirm removal unless force flag is set
    if (!force) {
        try stdout.print("\n", .{});
        const confirmed = try input.confirm("Remove these worktrees?", true);
        if (!confirmed) {
            try stdout.print("Cancelled\n", .{});
            return;
        }
    }

    // Remove each worktree
    var removed_count: usize = 0;
    for (to_clean.items) |wt| {
        // Remove the worktree
        const remove_result = try git.execWithResult(allocator, &[_][]const u8{
            "git", "worktree", "remove", wt.path, "--force",
        });
        defer switch (remove_result) {
            .success => |output| allocator.free(output),
            .failure => |err| allocator.free(err.stderr),
        };

        switch (remove_result) {
            .success => {
                try colors.printSuccess(stdout, "✓ Removed {s}", .{wt.branch});
                removed_count += 1;
            },
            .failure => |err| {
                try colors.printError(stderr, "Failed to remove {s}: {s}", .{
                    wt.branch,
                    git.trimNewline(err.stderr),
                });
            },
        }
    }

    // Summary
    try stdout.print("\n{s}Removed {d}/{d} worktrees{s}\n", .{
        if (removed_count == to_clean.items.len) colors.bright_green else colors.yellow,
        removed_count,
        to_clean.items.len,
        colors.reset,
    });
}
