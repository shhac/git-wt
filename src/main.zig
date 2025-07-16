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
    execute: *const fn (allocator: std.mem.Allocator, args: []const []const u8, non_interactive: bool) anyerror!void,
    help: *const fn () anyerror!void,
};

const commands = [_]Command{
    .{ .name = "new", .min_args = 1, .usage = "git-wt new <branch-name>", .execute = executeNew, .help = cmd_new.printHelp },
    .{ .name = "rm", .min_args = 0, .usage = "git-wt rm", .execute = executeRemove, .help = cmd_remove.printHelp },
    .{ .name = "go", .min_args = 0, .usage = "git-wt go [branch]", .execute = executeGo, .help = cmd_go.printHelp },
};

fn executeNew(allocator: std.mem.Allocator, args: []const []const u8, non_interactive: bool) !void {
    try cmd_new.execute(allocator, args[0], non_interactive);
}

fn executeRemove(allocator: std.mem.Allocator, args: []const []const u8, non_interactive: bool) !void {
    _ = args;
    try cmd_remove.execute(allocator, non_interactive);
}

fn executeGo(allocator: std.mem.Allocator, args: []const []const u8, non_interactive: bool) !void {
    var no_color = false;
    var plain = false;
    var show_command = false;
    var branch: ?[]const u8 = null;
    
    // Parse go-specific flags
    for (args) |arg| {
        if (std.mem.eql(u8, arg, "--no-color")) {
            no_color = true;
        } else if (std.mem.eql(u8, arg, "--plain")) {
            plain = true;
        } else if (std.mem.eql(u8, arg, "--show-command")) {
            show_command = true;
        } else if (arg.len > 0 and arg[0] != '-') {
            // First non-flag argument is the branch
            if (branch == null) {
                branch = arg;
            }
        }
    }
    
    try cmd_go.execute(allocator, branch, non_interactive, no_color, plain, show_command);
}

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const args = try process.argsAlloc(allocator);
    defer process.argsFree(allocator, args);

    // Parse global flags
    var non_interactive = false;
    var filtered_args = std.ArrayList([]const u8).init(allocator);
    defer filtered_args.deinit();
    
    for (args) |arg| {
        if (std.mem.eql(u8, arg, "--non-interactive") or std.mem.eql(u8, arg, "-n")) {
            non_interactive = true;
        } else {
            try filtered_args.append(arg);
        }
    }
    
    const final_args = filtered_args.items;

    if (final_args.len < 2) {
        printUsage();
        return;
    }

    // Check for help/version flags
    const arg1 = final_args[1];
    if (std.mem.eql(u8, arg1, "--help") or std.mem.eql(u8, arg1, "-h")) {
        // Check if there's a specific help topic
        if (final_args.len > 2 and std.mem.eql(u8, final_args[2], "setup")) {
            printSetupHelp();
        } else {
            printHelp();
        }
        return;
    }

    if (std.mem.eql(u8, arg1, "--version") or std.mem.eql(u8, arg1, "-v")) {
        print("git-wt version 0.1.0\n", .{});
        return;
    }

    if (std.mem.eql(u8, arg1, "--alias")) {
        if (final_args.len < 3) {
            printAliasUsage();
            return;
        }
        const exe_path = try std.fs.selfExePathAlloc(allocator);
        defer allocator.free(exe_path);
        printAliasFunction(final_args[2], exe_path);
        return;
    }

    const command_args = if (final_args.len > 2) final_args[2..] else &[_][]const u8{};
    
    // Find and execute command
    for (commands) |cmd| {
        if (std.mem.eql(u8, arg1, cmd.name)) {
            // Check for help flag on the command
            if (command_args.len > 0 and (std.mem.eql(u8, command_args[0], "--help") or std.mem.eql(u8, command_args[0], "-h"))) {
                try cmd.help();
                return;
            }
            
            if (command_args.len < cmd.min_args) {
                const stderr = std.io.getStdErr().writer();
                try colors.printError(stderr, "Missing required arguments", .{});
                print("Usage: {s}\n", .{cmd.usage});
                process.exit(1);
            }
            try cmd.execute(allocator, command_args, non_interactive);
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
    print("Usage: git-wt [--non-interactive] <command> [options]\n", .{});
    print("\nCommands:\n", .{});
    print("  new <branch>  Create a new worktree\n", .{});
    print("  rm            Remove current worktree\n", .{});
    print("  go [branch]   Navigate to worktree\n", .{});
    print("\nGlobal flags:\n", .{});
    print("  -n, --non-interactive  Run without prompts (for testing)\n", .{});
    print("  -h, --help             Show help\n", .{});
    print("  -v, --version          Show version\n", .{});
    print("  --alias <name>         Generate shell function for directory navigation\n", .{});
    print("\nUse 'git-wt --help' for more information\n", .{});
    print("Use 'git-wt --help setup' for shell integration setup\n", .{});
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
    print("\nFor shell integration setup, use: git-wt --help setup\n", .{});
}

fn printSetupHelp() void {
    print("git-wt - Shell Integration Setup\n\n", .{});
    print("Since CLI tools cannot change the parent shell's directory, you need to set up\n", .{});
    print("a shell function wrapper to enable proper directory navigation.\n\n", .{});
    print("SETUP INSTRUCTIONS:\n", .{});
    print("\n1. Generate the shell function:\n", .{});
    print("   git-wt --alias gwt\n", .{});
    print("\n2. Add to your shell configuration:\n", .{});
    print("   For zsh (.zshrc):\n", .{});
    print("     echo 'eval \"$(git-wt --alias gwt)\"' >> ~/.zshrc\n", .{});
    print("     source ~/.zshrc\n", .{});
    print("\n   For bash (.bashrc):\n", .{});
    print("     echo 'eval \"$(git-wt --alias gwt)\"' >> ~/.bashrc\n", .{});
    print("     source ~/.bashrc\n", .{});
    print("\n3. Use the alias for directory navigation:\n", .{});
    print("   gwt new feature-branch    # Creates worktree AND navigates to it\n", .{});
    print("   gwt go main              # Actually changes to main repository\n", .{});
    print("   gwt go feature-branch    # Actually changes to worktree\n", .{});
    print("   gwt rm                   # Works the same as git-wt rm\n", .{});
    print("\nNOTE: You can use any alias name instead of 'gwt':\n", .{});
    print("  git-wt --alias wt        # Creates 'wt' alias\n", .{});
    print("  git-wt --alias gw        # Creates 'gw' alias\n", .{});
    print("\nWithout this setup, git-wt commands will work but won't change directories.\n", .{});
}

fn printAliasUsage() void {
    print("Usage: git-wt --alias <alias-name>\n\n", .{});
    print("Generate shell function for proper directory navigation.\n\n", .{});
    print("Example:\n", .{});
    print("  # Add to your .zshrc or .bashrc:\n", .{});
    print("  eval \"$(git-wt --alias gwt)\"\n\n", .{});
    print("  # Then use:\n", .{});
    print("  gwt go feature-branch    # This will actually change directories\n", .{});
}

fn printAliasFunction(alias_name: []const u8, exe_path: []const u8) void {
    print("{s}() {{\n", .{alias_name});
    print("    local git_wt_bin=\"{s}\"\n", .{exe_path});
    print("    if [ \"$1\" = \"go\" ]; then\n", .{});
    print("        shift\n", .{});
    print("        local has_branch=0\n", .{});
    print("        for arg in \"$@\"; do\n", .{});
    print("            if [[ \"$arg\" != -* ]]; then\n", .{});
    print("                has_branch=1\n", .{});
    print("                break\n", .{});
    print("            fi\n", .{});
    print("        done\n", .{});
    print("        if [ $has_branch -eq 0 ]; then\n", .{});
    print("            \"$git_wt_bin\" go \"$@\"\n", .{});
    print("        else\n", .{});
    print("            local cd_cmd=$(\"$git_wt_bin\" go --show-command \"$@\")\n", .{});
    print("            if echo \"$cd_cmd\" | grep -q '^cd '; then\n", .{});
    print("                eval \"$cd_cmd\"\n", .{});
    print("            else\n", .{});
    print("                echo \"$cd_cmd\"\n", .{});
    print("            fi\n", .{});
    print("        fi\n", .{});
    print("    elif [ \"$1\" = \"new\" ]; then\n", .{});
    print("        shift\n", .{});
    print("        local branch=\"$1\"\n", .{});
    print("        \"$git_wt_bin\" new \"$@\"\n", .{});
    print("        if [ $? -eq 0 ] && [ -n \"$branch\" ]; then\n", .{});
    print("            local cd_cmd=$(\"$git_wt_bin\" go --show-command \"$branch\")\n", .{});
    print("            if [ -n \"$cd_cmd\" ]; then\n", .{});
    print("                eval \"$cd_cmd\"\n", .{});
    print("            fi\n", .{});
    print("        fi\n", .{});
    print("    else\n", .{});
    print("        \"$git_wt_bin\" \"$@\"\n", .{});
    print("    fi\n", .{});
    print("}}\n", .{});
}

test {
    _ = @import("utils/test_all.zig");
}