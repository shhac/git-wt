const std = @import("std");
const process = std.process;
const print = std.debug.print;

const git = @import("utils/git.zig");
const fs_utils = @import("utils/fs.zig");
const colors = @import("utils/colors.zig");

const cmd_new = @import("commands/new.zig");
const cmd_remove = @import("commands/remove.zig");
const cmd_go = @import("commands/go.zig");

const Command = struct {
    name: []const u8,
    min_args: usize,
    usage: []const u8,
    execute: *const fn (allocator: std.mem.Allocator, args: []const []const u8) anyerror!void,
};

const commands = [_]Command{
    .{ .name = "new", .min_args = 1, .usage = "git-wt new <branch-name>", .execute = executeNew },
    .{ .name = "rm", .min_args = 0, .usage = "git-wt rm", .execute = executeRemove },
    .{ .name = "go", .min_args = 0, .usage = "git-wt go [branch]", .execute = executeGo },
};

fn executeNew(allocator: std.mem.Allocator, args: []const []const u8) !void {
    try cmd_new.execute(allocator, args[0]);
}

fn executeRemove(allocator: std.mem.Allocator, args: []const []const u8) !void {
    _ = args;
    try cmd_remove.execute(allocator);
}

fn executeGo(allocator: std.mem.Allocator, args: []const []const u8) !void {
    const branch = if (args.len > 0) args[0] else null;
    try cmd_go.execute(allocator, branch);
}

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const args = try process.argsAlloc(allocator);
    defer process.argsFree(allocator, args);

    if (args.len < 2) {
        printUsage();
        return;
    }

    // Check for help/version flags
    const arg1 = args[1];
    if (std.mem.eql(u8, arg1, "--help") or std.mem.eql(u8, arg1, "-h")) {
        printHelp();
        return;
    }

    if (std.mem.eql(u8, arg1, "--version") or std.mem.eql(u8, arg1, "-v")) {
        print("git-wt version 0.1.0\n", .{});
        return;
    }

    const command_args = if (args.len > 2) args[2..] else &[_][]const u8{};
    
    // Find and execute command
    for (commands) |cmd| {
        if (std.mem.eql(u8, arg1, cmd.name)) {
            if (command_args.len < cmd.min_args) {
                const stderr = std.io.getStdErr().writer();
                try colors.printError(stderr, "Missing required arguments", .{});
                print("Usage: {s}\n", .{cmd.usage});
                process.exit(1);
            }
            try cmd.execute(allocator, command_args);
            return;
        }
    }
    
    // Unknown command
    const stderr = std.io.getStdErr().writer();
    try stderr.print("{s}Error:{s} Unknown command '{s}'\n", .{ colors.error_prefix, colors.reset, arg1 });
    printUsage();
    process.exit(1);
}

fn printUsage() void {
    print("Usage: git-wt <command> [options]\n", .{});
    print("\nCommands:\n", .{});
    print("  new <branch>  Create a new worktree\n", .{});
    print("  rm            Remove current worktree\n", .{});
    print("  go [branch]   Navigate to worktree\n", .{});
    print("\nUse 'git-wt --help' for more information\n", .{});
}

fn printHelp() void {
    print("git-wt - Git worktree management tool\n\n", .{});
    printUsage();
    print("\nExamples:\n", .{});
    print("  git-wt new feature-branch   Create a new worktree for 'feature-branch'\n", .{});
    print("  git-wt rm                   Remove the current worktree\n", .{});
    print("  git-wt go                   Interactively select and navigate to a worktree\n", .{});
    print("  git-wt go main              Navigate to the main repository\n", .{});
    print("  git-wt go feature-branch    Navigate to the 'feature-branch' worktree\n", .{});
}