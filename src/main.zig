const std = @import("std");
const process = std.process;
const print = std.debug.print;

const git = @import("utils/git.zig");
const fs_utils = @import("utils/fs.zig");
const colors = @import("utils/colors.zig");
const debug = @import("utils/debug.zig");

const cmd_new = @import("commands/new.zig");
const cmd_remove = @import("commands/remove.zig");
const cmd_go = @import("commands/go.zig");
const cmd_list = @import("commands/list.zig");

const Command = struct {
    name: []const u8,
    min_args: usize,
    usage: []const u8,
    execute: *const fn (allocator: std.mem.Allocator, args: []const []const u8, non_interactive: bool) anyerror!void,
    help: *const fn () anyerror!void,
};

const commands = [_]Command{
    .{ .name = "new", .min_args = 1, .usage = "git-wt new <branch-name>", .execute = executeNew, .help = cmd_new.printHelp },
    .{ .name = "rm", .min_args = 1, .usage = "git-wt rm <branch-name>", .execute = executeRemove, .help = cmd_remove.printHelp },
    .{ .name = "go", .min_args = 0, .usage = "git-wt go [branch]", .execute = executeGo, .help = cmd_go.printHelp },
    .{ .name = "list", .min_args = 0, .usage = "git-wt list", .execute = executeList, .help = cmd_list.printHelp },
};

fn executeNew(allocator: std.mem.Allocator, args: []const []const u8, non_interactive: bool) !void {
    var parent_dir: ?[]const u8 = null;
    var branch_name: ?[]const u8 = null;
    
    // Parse new-specific flags
    var i: usize = 0;
    while (i < args.len) : (i += 1) {
        const arg = args[i];
        if (std.mem.eql(u8, arg, "--parent-dir") or std.mem.eql(u8, arg, "-p")) {
            if (i + 1 >= args.len) {
                const stderr = std.io.getStdErr().writer();
                try colors.printError(stderr, "--parent-dir requires a directory path", .{});
                std.process.exit(1);
            }
            i += 1;
            parent_dir = args[i];
        } else if (branch_name == null and arg.len > 0 and arg[0] != '-') {
            branch_name = arg;
        }
    }
    
    if (branch_name) |branch| {
        try cmd_new.execute(allocator, branch, non_interactive, parent_dir);
    } else {
        const stderr = std.io.getStdErr().writer();
        try colors.printError(stderr, "Missing required arguments", .{});
        print("Usage: {s}\n", .{commands[0].usage});
        process.exit(1);
    }
}

fn executeRemove(allocator: std.mem.Allocator, args: []const []const u8, non_interactive: bool) !void {
    var force = false;
    var branch_name: ?[]const u8 = null;
    
    // Parse rm-specific flags
    for (args) |arg| {
        if (std.mem.eql(u8, arg, "--force") or std.mem.eql(u8, arg, "-f")) {
            force = true;
        } else if (branch_name == null) {
            branch_name = arg;
        }
    }
    
    if (branch_name) |branch| {
        try cmd_remove.execute(allocator, branch, non_interactive, force);
    } else {
        // This should not happen due to min_args check, but just in case
        const stderr = std.io.getStdErr().writer();
        try stderr.print("Error: Branch name is required\n", .{});
        std.process.exit(1);
    }
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

fn executeList(allocator: std.mem.Allocator, args: []const []const u8, non_interactive: bool) !void {
    _ = non_interactive; // List doesn't use this flag
    var no_color = false;
    var plain = false;
    
    // Parse list-specific flags
    for (args) |arg| {
        if (std.mem.eql(u8, arg, "--no-color")) {
            no_color = true;
        } else if (std.mem.eql(u8, arg, "--plain")) {
            plain = true;
        }
    }
    
    try cmd_list.execute(allocator, no_color, plain);
}

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const args = try process.argsAlloc(allocator);
    defer process.argsFree(allocator, args);

    // Parse global flags
    var non_interactive = false;
    var debug_mode = false;
    var filtered_args = std.ArrayList([]const u8).init(allocator);
    defer filtered_args.deinit();
    
    for (args) |arg| {
        if (std.mem.eql(u8, arg, "--non-interactive") or std.mem.eql(u8, arg, "-n")) {
            non_interactive = true;
        } else if (std.mem.eql(u8, arg, "--debug")) {
            debug_mode = true;
            debug.setEnabled(true);
        } else {
            try filtered_args.append(arg);
        }
    }
    
    const final_args = filtered_args.items;
    
    // Print debug information if enabled
    if (debug_mode) {
        debug.printSection("Environment");
        
        // Current working directory
        const cwd = try std.fs.cwd().realpathAlloc(allocator, ".");
        defer allocator.free(cwd);
        debug.print("Current directory: {s}", .{cwd});
        
        // Command line arguments
        debug.print("Arguments: {s}", .{args});
        debug.print("Filtered args: {s}", .{final_args});
        
        // Environment variables
        if (std.process.getEnvVarOwned(allocator, "NON_INTERACTIVE")) |v| {
            defer allocator.free(v);
            debug.print("NON_INTERACTIVE: {s}", .{v});
        } else |_| {
            debug.print("NON_INTERACTIVE: not set", .{});
        }
        
        if (std.process.getEnvVarOwned(allocator, "NO_COLOR")) |v| {
            defer allocator.free(v);
            debug.print("NO_COLOR: {s}", .{v});
        } else |_| {
            debug.print("NO_COLOR: not set", .{});
        }
        
        // Git version
        if (git.execTrimmed(allocator, &.{"--version"})) |version| {
            defer allocator.free(version);
            debug.print("Git version: {s}", .{version});
        } else |_| {
            debug.print("Git version: unable to determine", .{});
        }
        
        // Git repository info
        if (git.getRepoInfo(allocator)) |repo_info| {
            defer allocator.free(repo_info.root);
            defer if (repo_info.main_repo_root) |root| allocator.free(root);
            
            debug.printSection("Repository Info");
            debug.print("Repository root: {s}", .{repo_info.root});
            debug.print("Repository name: {s}", .{repo_info.name});
            debug.print("Is worktree: {}", .{repo_info.is_worktree});
            if (repo_info.main_repo_root) |main_root| {
                debug.print("Main repo root: {s}", .{main_root});
            }
        } else |_| {
            debug.print("Not in a git repository", .{});
        }
    }

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
            debug.printSection("Command Execution");
            debug.print("Command: {s}", .{cmd.name});
            debug.print("Arguments: {s}", .{command_args});
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
    print("  rm <branch>   Remove worktree by branch name\n", .{});
    print("  go [branch]   Navigate to worktree\n", .{});
    print("  list          List all worktrees\n", .{});
    print("\nGlobal flags:\n", .{});
    print("  -n, --non-interactive  Run without prompts (for testing)\n", .{});
    print("  -h, --help             Show help\n", .{});
    print("  -v, --version          Show version\n", .{});
    print("  --alias <name>         Generate shell function for directory navigation\n", .{});
    print("  --debug                Show diagnostic information\n", .{});
    print("\nUse 'git-wt --help' for more information\n", .{});
    print("Use 'git-wt --help setup' for shell integration setup\n", .{});
}

fn printHelp() void {
    print("git-wt - Git worktree management tool\n\n", .{});
    printUsage();
    print("\nExamples:\n", .{});
    print("  git-wt new feature-branch   Create a new worktree for 'feature-branch'\n", .{});
    print("  git-wt rm feature-branch    Remove the 'feature-branch' worktree\n", .{});
    print("  git-wt go                   Interactively select and navigate to a worktree\n", .{});
    print("  git-wt go main              Navigate to the main repository\n", .{});
    print("  git-wt go feature-branch    Navigate to the 'feature-branch' worktree\n", .{});
    print("  git-wt list                 List all worktrees with details\n", .{});
    print("  git-wt list --plain         List worktrees in machine-readable format\n", .{});
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
    print("   gwt rm feature-branch    # Removes the feature-branch worktree\n", .{});
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
    const stdout = std.io.getStdOut().writer();
    stdout.print("# Shell function wrapper for git-wt to enable directory navigation\n", .{}) catch return;
    stdout.print("{s}() {{\n", .{alias_name}) catch return;
    stdout.print("    local git_wt_bin=\"{s}\"\n", .{exe_path}) catch return;
    stdout.print("    if [ \"$1\" = \"go\" ]; then\n", .{}) catch return;
    stdout.print("        shift\n", .{}) catch return;
    stdout.print("        local has_branch=0\n", .{}) catch return;
    stdout.print("        for arg in \"$@\"; do\n", .{}) catch return;
    stdout.print("            if [[ \"$arg\" != -* ]]; then\n", .{}) catch return;
    stdout.print("                has_branch=1\n", .{}) catch return;
    stdout.print("                break\n", .{}) catch return;
    stdout.print("            fi\n", .{}) catch return;
    stdout.print("        done\n", .{}) catch return;
    stdout.print("        if [ $has_branch -eq 0 ]; then\n", .{}) catch return;
    stdout.print("            # Interactive mode - capture cd command from fd 3, show UI on stdout/stderr\n", .{}) catch return;
    stdout.print("            local cd_cmd=$(GWT_USE_FD3=1 \"$git_wt_bin\" go --show-command \"$@\" 3>&1 1>&2)\n", .{}) catch return;
    stdout.print("            if [ $? -eq 0 ] && [ -n \"$cd_cmd\" ] && echo \"$cd_cmd\" | grep -q '^cd '; then\n", .{}) catch return;
    stdout.print("                eval \"$cd_cmd\"\n", .{}) catch return;
    stdout.print("            fi\n", .{}) catch return;
    stdout.print("        else\n", .{}) catch return;
    stdout.print("            # Direct navigation - capture cd command from fd 3\n", .{}) catch return;
    stdout.print("            local cd_cmd=$(GWT_USE_FD3=1 \"$git_wt_bin\" go --show-command \"$@\" 3>&1 1>&2)\n", .{}) catch return;
    stdout.print("            if [ $? -eq 0 ] && echo \"$cd_cmd\" | grep -q '^cd '; then\n", .{}) catch return;
    stdout.print("                eval \"$cd_cmd\"\n", .{}) catch return;
    stdout.print("            else\n", .{}) catch return;
    stdout.print("                # If no cd command, something went wrong - show the output\n", .{}) catch return;
    stdout.print("                \"$git_wt_bin\" go \"$@\"\n", .{}) catch return;
    stdout.print("            fi\n", .{}) catch return;
    stdout.print("        fi\n", .{}) catch return;
    stdout.print("    elif [ \"$1\" = \"new\" ]; then\n", .{}) catch return;
    stdout.print("        shift\n", .{}) catch return;
    stdout.print("        local branch=\"$1\"\n", .{}) catch return;
    stdout.print("        \"$git_wt_bin\" new \"$@\"\n", .{}) catch return;
    stdout.print("        if [ $? -eq 0 ] && [ -n \"$branch\" ]; then\n", .{}) catch return;
    stdout.print("            local cd_cmd=$(GWT_USE_FD3=1 \"$git_wt_bin\" go --show-command \"$branch\" 3>&1 1>&2)\n", .{}) catch return;
    stdout.print("            if [ -n \"$cd_cmd\" ]; then\n", .{}) catch return;
    stdout.print("                eval \"$cd_cmd\"\n", .{}) catch return;
    stdout.print("            fi\n", .{}) catch return;
    stdout.print("        fi\n", .{}) catch return;
    stdout.print("    else\n", .{}) catch return;
    stdout.print("        \"$git_wt_bin\" \"$@\"\n", .{}) catch return;
    stdout.print("    fi\n", .{}) catch return;
    stdout.print("}}\n", .{}) catch return;
}

test {
    _ = @import("utils/test_all.zig");
}