const std = @import("std");
const process = std.process;
const fs = std.fs;

const git = @import("../utils/git.zig");
const fs_utils = @import("../utils/fs.zig");
const colors = @import("../utils/colors.zig");
const input = @import("../utils/input.zig");

const WorktreeInfo = struct {
    path: []const u8,
    branch: []const u8,
    mod_time: i128,
    is_main: bool,
};

pub fn printHelp() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("Usage: git-wt go [branch-name]\n\n", .{});
    try stdout.print("Navigate to a git worktree or the main repository.\n\n", .{});
    try stdout.print("Arguments:\n", .{});
    try stdout.print("  [branch-name]    Name of the branch/worktree to navigate to (optional)\n\n", .{});
    try stdout.print("Options:\n", .{});
    try stdout.print("  -h, --help       Show this help message\n", .{});
    try stdout.print("  -n, --non-interactive  List worktrees without interaction\n\n", .{});
    try stdout.print("Examples:\n", .{});
    try stdout.print("  git-wt go                      # Interactive selection of worktrees\n", .{});
    try stdout.print("  git-wt go main                 # Navigate to main repository\n", .{});
    try stdout.print("  git-wt go feature-branch       # Navigate to feature-branch worktree\n", .{});
    try stdout.print("  git-wt go --non-interactive    # List all worktrees only\n\n", .{});
    try stdout.print("This command will:\n", .{});
    try stdout.print("  1. List all available worktrees (sorted by modification time)\n", .{});
    try stdout.print("  2. Allow interactive selection if no branch specified\n", .{});
    try stdout.print("  3. Navigate to the selected worktree\n", .{});
    try stdout.print("  4. Change the current working directory\n\n", .{});
    try stdout.print("Note: Use 'main' as the branch name to navigate to the main repository.\n", .{});
}

pub fn execute(allocator: std.mem.Allocator, branch_name: ?[]const u8, non_interactive: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    // Get repository info
    const repo_info = git.getRepoInfo(allocator) catch |err| {
        try colors.printError(stderr, "Not in a git repository", .{});
        return err;
    };
    defer allocator.free(repo_info.root);
    defer if (repo_info.main_repo_root) |root| allocator.free(root);
    
    const main_repo = if (repo_info.is_worktree) repo_info.main_repo_root.? else repo_info.root;
    const repo_name = fs.path.basename(main_repo);
    const parent_dir = fs.path.dirname(main_repo) orelse ".";
    const trees_dir = try std.fmt.allocPrint(allocator, "{s}/{s}-trees", .{ parent_dir, repo_name });
    defer allocator.free(trees_dir);
    
    if (branch_name) |branch| {
        // Direct navigation
        if (std.mem.eql(u8, branch, "main")) {
            try colors.printPath(stdout, "📁 Navigating to main repository:", main_repo);
            try process.changeCurDir(main_repo);
            return;
        }
        
        // Navigate to specific worktree
        const worktree_path = try std.fmt.allocPrint(allocator, "{s}/{s}", .{ trees_dir, branch });
        defer allocator.free(worktree_path);
        
        // Check if directory exists
        fs.cwd().access(worktree_path, .{}) catch {
            try stderr.print("{s}Error:{s} Worktree for branch '{s}' not found at:\n", .{ colors.error_prefix, colors.reset, branch });
            try stderr.print("       {s}{s}{s}\n", .{ colors.path_color, worktree_path, colors.reset });
            return error.WorktreeNotFound;
        };
        
        if (non_interactive) {
            try stdout.print("cd {s}\n", .{worktree_path});
        } else {
            try colors.printPath(stdout, "📁 Navigating to worktree:", worktree_path);
            try process.changeCurDir(worktree_path);
        }
    } else {
        // Interactive selection (or just list if non-interactive)
        var worktrees = std.ArrayList(WorktreeInfo).init(allocator);
        defer {
            for (worktrees.items) |wt| {
                allocator.free(wt.path);
                allocator.free(wt.branch);
            }
            worktrees.deinit();
        }
        
        // Add main repository if we're not already in it
        if (!std.mem.eql(u8, repo_info.root, main_repo)) {
            const stat = try fs.cwd().statFile(main_repo);
            const main_branch = blk: {
                var saved_cwd = try fs.cwd().openDir(".", .{});
                defer saved_cwd.close();
                try process.changeCurDir(main_repo);
                const branch = try git.getCurrentBranch(allocator);
                try saved_cwd.setAsCwd();
                break :blk branch;
            };
            
            try worktrees.append(.{
                .path = try allocator.dupe(u8, main_repo),
                .branch = main_branch,
                .mod_time = stat.mtime,
                .is_main = true,
            });
        }
        
        // Find worktrees in trees directory
        if (fs.cwd().openDir(trees_dir, .{ .iterate = true })) |dir| {
            var trees_dir_handle = dir;
            defer trees_dir_handle.close();
            
            var iter = trees_dir_handle.iterate();
            while (try iter.next()) |entry| {
                if (entry.kind != .directory) continue;
                
                const wt_path = try std.fmt.allocPrint(allocator, "{s}/{s}", .{ trees_dir, entry.name });
                
                // Check if it's a valid worktree
                const git_file = try std.fmt.allocPrint(allocator, "{s}/.git", .{ wt_path });
                defer allocator.free(git_file);
                
                if (fs.cwd().statFile(git_file)) |_| {
                    const stat = try fs.cwd().statFile(wt_path);
                    
                    // Get current branch
                    var saved_cwd = try fs.cwd().openDir(".", .{});
                    defer saved_cwd.close();
                    try process.changeCurDir(wt_path);
                    const branch = try git.getCurrentBranch(allocator);
                    try saved_cwd.setAsCwd();
                    
                    try worktrees.append(.{
                        .path = wt_path,
                        .branch = branch,
                        .mod_time = stat.mtime,
                        .is_main = false,
                    });
                } else |_| {
                    allocator.free(wt_path);
                }
            }
        } else |_| {}
        
        if (worktrees.items.len == 0) {
            try stdout.print("{s}No worktrees found in:{s} {s}{s}{s}\n", .{
                colors.warning_prefix,
                colors.reset,
                colors.path_color,
                trees_dir,
                colors.reset,
            });
            return;
        }
        
        // Sort by modification time (newest first)
        std.mem.sort(WorktreeInfo, worktrees.items, {}, struct {
            fn lessThan(_: void, a: WorktreeInfo, b: WorktreeInfo) bool {
                return a.mod_time > b.mod_time;
            }
        }.lessThan);
        
        // Display worktrees
        try colors.printInfo(stdout, "Available worktrees:\n", .{});
        
        for (worktrees.items, 1..) |wt, idx| {
            const display_name = if (wt.is_main) "[main repository]" else fs.path.basename(wt.path);
            const timestamp = @divFloor(wt.mod_time, std.time.ns_per_s);
            
            try stdout.print("  {s}{d}{s}) {s}{s}{s} @ {s}{s}{s}\n", .{
                colors.green,
                idx,
                colors.reset,
                colors.path_color,
                display_name,
                colors.reset,
                colors.magenta,
                wt.branch,
                colors.reset,
            });
            
            // Format timestamp (simplified)
            try stdout.print("     {s}Last modified:{s} {d} seconds ago\n", .{
                colors.yellow,
                colors.reset,
                std.time.timestamp() - timestamp,
            });
        }
        
        // In non-interactive mode, just list and exit
        if (non_interactive) {
            return;
        }
        
        const prompt = try std.fmt.allocPrint(allocator, "\n{s}Enter number to navigate to (or 'q' to quit):{s}", .{ colors.yellow, colors.reset });
        defer allocator.free(prompt);
        
        if (try input.readLine(allocator, prompt)) |response| {
            defer allocator.free(response);
            
            if (response.len > 0 and (response[0] == 'q' or response[0] == 'Q')) {
                try colors.printInfo(stdout, "Cancelled", .{});
                return;
            }
            
            const selection = if (response.len == 0) 1 else std.fmt.parseInt(usize, response, 10) catch {
                try colors.printError(stderr, "Invalid selection", .{});
                return error.InvalidSelection;
            };
            
            if (selection < 1 or selection > worktrees.items.len) {
                try colors.printError(stderr, "Invalid selection", .{});
                return error.InvalidSelection;
            }
            
            const selected = worktrees.items[selection - 1];
            try colors.printPath(stdout, "📁 Navigating to worktree:", selected.path);
            try process.changeCurDir(selected.path);
        }
    }
}