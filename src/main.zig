const std = @import("std");
const process = std.process;
const print = std.debug.print;

pub fn main() !void {
    var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
    defer arena.deinit();
    const allocator = arena.allocator();

    const args = try process.argsAlloc(allocator);
    defer process.argsFree(allocator, args);

    if (args.len < 2) {
        printUsage();
        return;
    }

    const command = args[1];

    if (std.mem.eql(u8, command, "new")) {
        if (args.len < 3) {
            print("Error: Please provide a branch name\n", .{});
            print("Usage: git-wt new <branch-name>\n", .{});
            process.exit(1);
        }
        print("TODO: Implement 'new' command for branch: {s}\n", .{args[2]});
    } else if (std.mem.eql(u8, command, "rm")) {
        print("TODO: Implement 'rm' command\n", .{});
    } else if (std.mem.eql(u8, command, "go")) {
        if (args.len >= 3) {
            print("TODO: Implement 'go' command for branch: {s}\n", .{args[2]});
        } else {
            print("TODO: Implement interactive 'go' command\n", .{});
        }
    } else if (std.mem.eql(u8, command, "--help") or std.mem.eql(u8, command, "-h")) {
        printHelp();
    } else if (std.mem.eql(u8, command, "--version") or std.mem.eql(u8, command, "-v")) {
        print("git-wt version 0.1.0\n", .{});
    } else {
        print("Error: Unknown command '{s}'\n", .{command});
        printUsage();
        process.exit(1);
    }
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