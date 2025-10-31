const std = @import("std");
const process = std.process;
const print = std.debug.print;
const build_options = @import("build_options");

const git = @import("utils/git.zig");
const fs_utils = @import("utils/fs.zig");
const colors = @import("utils/colors.zig");
const debug = @import("utils/debug.zig");
const args_parser = @import("utils/args.zig");
const input = @import("utils/input.zig");
const interactive = @import("utils/interactive.zig");
const io = @import("utils/io.zig");
const config = @import("utils/config.zig");

const cmd_new = @import("commands/new.zig");
const cmd_remove = @import("commands/remove.zig");
const cmd_go = @import("commands/go.zig");
const cmd_list = @import("commands/list.zig");
const cmd_alias = @import("commands/alias.zig");
const cmd_clean = @import("commands/clean.zig");

const Command = struct {
    name: []const u8,
    min_args: usize,
    usage: []const u8,
    execute: *const fn (allocator: std.mem.Allocator, args: []const []const u8, cfg: *config.Config, non_interactive: bool, no_tty: bool) anyerror!void,
    help: *const fn () anyerror!void,
};

const commands = [_]Command{
    .{ .name = "new", .min_args = 1, .usage = "git-wt new <branch-name>", .execute = executeNew, .help = cmd_new.printHelp },
    .{ .name = "rm", .min_args = 0, .usage = "git-wt rm [branch-name]", .execute = executeRemove, .help = cmd_remove.printHelp },
    .{ .name = "go", .min_args = 0, .usage = "git-wt go [branch]", .execute = executeGo, .help = cmd_go.printHelp },
    .{ .name = "list", .min_args = 0, .usage = "git-wt list", .execute = executeList, .help = cmd_list.printHelp },
    .{ .name = "alias", .min_args = 1, .usage = "git-wt alias <name> [options]", .execute = executeAlias, .help = cmd_alias.printHelp },
    .{ .name = "clean", .min_args = 0, .usage = "git-wt clean [options]", .execute = executeClean, .help = cmd_clean.printHelp },
};

fn executeNew(allocator: std.mem.Allocator, args: []const []const u8, cfg: *config.Config, non_interactive: bool, no_tty: bool) !void {
    _ = no_tty; // Not used in new command yet

    // Parse arguments using the new parser
    var parsed = try args_parser.parseArgs(allocator, args);
    defer parsed.deinit(allocator);

    // Command-line flag overrides config
    const parent_dir = parsed.getFlag(&.{ "--parent-dir", "-p" }) orelse cfg.parent_dir;
    const branch_name = parsed.getPositional(0);

    if (branch_name) |branch| {
        try cmd_new.execute(allocator, branch, non_interactive, parent_dir);
    } else {
        const stderr = io.getStdErr();
        try colors.printError(stderr, "Missing required arguments", .{});
        print("Usage: {s}\n", .{commands[0].usage});
        return error.MissingBranchName;
    }
}

fn executeRemove(allocator: std.mem.Allocator, args: []const []const u8, cfg: *config.Config, non_interactive: bool, no_tty: bool) !void {
    _ = cfg; // Config not used yet in remove command

    // Parse arguments using the new parser
    var parsed = try args_parser.parseArgs(allocator, args);
    defer parsed.deinit(allocator);

    const force = parsed.hasFlag(&.{ "--force", "-f" });
    const branch_names = parsed.getPositionals();

    if (branch_names.len > 0) {
        // Multiple branch removal mode
        if (branch_names.len == 1) {
            // Single branch - use existing function
            try cmd_remove.execute(allocator, branch_names[0], non_interactive, force);
        } else {
            // Multiple branches - use new function
            try cmd_remove.executeMultiple(allocator, branch_names, non_interactive, force);
        }
    } else {
        // Interactive mode - let the remove command handle it
        try cmd_remove.executeInteractive(allocator, non_interactive or no_tty, force);
    }
}

fn executeGo(allocator: std.mem.Allocator, args: []const []const u8, cfg: *config.Config, non_interactive: bool, no_tty: bool) !void {
    // Parse arguments using the new parser
    var parsed = try args_parser.parseArgs(allocator, args);
    defer parsed.deinit(allocator);

    // Command-line flags override config
    const no_color = if (parsed.hasFlag(&.{"--no-color"})) true else cfg.no_color;
    const plain = if (parsed.hasFlag(&.{"--plain"})) true else cfg.plain_output;
    const show_command = parsed.hasFlag(&.{"--show-command"});
    const branch = parsed.getPositional(0);

    try cmd_go.execute(allocator, branch, non_interactive, no_tty, no_color, plain, show_command);
}

fn executeList(allocator: std.mem.Allocator, args: []const []const u8, cfg: *config.Config, non_interactive: bool, no_tty: bool) !void {
    _ = non_interactive; // List doesn't use this flag
    _ = no_tty; // List doesn't use this flag

    // Parse arguments using the new parser
    var parsed = try args_parser.parseArgs(allocator, args);
    defer parsed.deinit(allocator);

    // Command-line flags override config
    const no_color = if (parsed.hasFlag(&.{"--no-color"})) true else cfg.no_color;
    const plain = if (parsed.hasFlag(&.{"--plain"})) true else cfg.plain_output;
    const json = if (parsed.hasFlag(&.{ "--json", "-j" })) true else cfg.json_output;

    try cmd_list.execute(allocator, no_color, plain, json);
}

fn executeAlias(allocator: std.mem.Allocator, args: []const []const u8, cfg: *config.Config, non_interactive: bool, no_tty: bool) !void {
    _ = cfg; // Config not used yet in alias command
    try cmd_alias.execute(allocator, args, non_interactive, no_tty);
}

fn executeClean(allocator: std.mem.Allocator, args: []const []const u8, cfg: *config.Config, non_interactive: bool, no_tty: bool) !void {
    _ = non_interactive; // Clean doesn't use this flag
    _ = no_tty; // Clean doesn't use this flag

    // Parse arguments using the new parser
    var parsed = try args_parser.parseArgs(allocator, args);
    defer parsed.deinit(allocator);

    const dry_run = parsed.hasFlag(&.{ "--dry-run", "-n" });
    // Command-line flag overrides config
    const force = if (parsed.hasFlag(&.{ "--force", "-f" })) true else cfg.auto_confirm;

    try cmd_clean.execute(allocator, dry_run, force);
}

pub fn main() void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();
    
    mainImpl(allocator) catch |err| {
        // Handle all errors consistently at the top level
        switch (err) {
            error.MissingRequiredArguments,
            error.MissingBranchName,
            error.UnknownCommand => process.exit(1),
            else => {
                const stderr = io.getStdErr();
                stderr.print("Error: {}\n", .{err}) catch {};
                process.exit(1);
            }
        }
    };
}

fn mainImpl(allocator: std.mem.Allocator) !void {
    const args = try process.argsAlloc(allocator);
    defer process.argsFree(allocator, args);

    // Load configuration from files (user config + project config)
    var cfg = config.loadConfig(allocator) catch config.Config{};
    defer cfg.deinit(allocator);

    // Parse global flags (override config)
    var non_interactive = cfg.non_interactive;
    var no_tty = cfg.no_tty;
    var debug_mode = false;
    var filtered_args = std.ArrayList([]const u8).empty;
    defer filtered_args.deinit(allocator);

    // Check if this is the alias command (needs special handling)
    const is_alias_command = args.len >= 2 and std.mem.eql(u8, args[1], "alias");
    
    for (args) |arg| {
        if (std.mem.eql(u8, arg, "--non-interactive") or std.mem.eql(u8, arg, "-n")) {
            non_interactive = true;
            // Pass through to alias command
            if (is_alias_command) {
                try filtered_args.append(allocator, arg);
            }
        } else if (std.mem.eql(u8, arg, "--no-tty")) {
            no_tty = true;
            // Pass through to alias command
            if (is_alias_command) {
                try filtered_args.append(allocator, arg);
            }
        } else if (std.mem.eql(u8, arg, "--debug")) {
            debug_mode = true;
            debug.setEnabled(true);
            // Pass through to alias command
            if (is_alias_command) {
                try filtered_args.append(allocator, arg);
            }
        } else {
            try filtered_args.append(allocator, arg);
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
        debug.print("Arguments: {any}", .{args});
        debug.print("Filtered args: {any}", .{final_args});
        
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
        
        if (std.process.getEnvVarOwned(allocator, "GWT_USE_FD3")) |v| {
            defer allocator.free(v);
            debug.print("GWT_USE_FD3: {s}", .{v});
        } else |_| {
            debug.print("GWT_USE_FD3: not set", .{});
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
        print("git-wt version {s}\n", .{build_options.version});
        return;
    }


    const command_args = if (final_args.len > 2) final_args[2..] else &[_][]const u8{};
    
    // Find and execute command
    for (commands) |cmd| {
        if (std.mem.eql(u8, arg1, cmd.name)) {
            debug.printSection("Command Execution");
            debug.print("Command: {s}", .{cmd.name});
            debug.print("Arguments: {any}", .{command_args});
            // Check for help flag on the command
            if (command_args.len > 0 and (std.mem.eql(u8, command_args[0], "--help") or std.mem.eql(u8, command_args[0], "-h"))) {
                try cmd.help();
                return;
            }
            
            if (command_args.len < cmd.min_args) {
                const stderr = io.getStdErr();
                try colors.printError(stderr, "Missing required arguments", .{});
                print("Usage: {s}\n", .{cmd.usage});
                return error.MissingRequiredArguments;
            }
            try cmd.execute(allocator, command_args, &cfg, non_interactive, no_tty);
            return;
        }
    }
    
    // Unknown command
    const stderr = io.getStdErr();
    try stderr.print("{s}Error:{s} Unknown command '{s}'\n", .{ colors.error_prefix, colors.reset, arg1 });
    printUsage();
    return error.UnknownCommand;
}

fn printUsage() void {
    print("Usage: git-wt [--non-interactive] [--no-tty] <command> [options]\n", .{});
    print("\nCommands:\n", .{});
    print("  new <branch>  Create a new worktree\n", .{});
    print("  rm [branch...] Remove worktree(s) (interactive multi-select if none)\n", .{});
    print("  go [branch]   Navigate to worktree\n", .{});
    print("  list          List all worktrees\n", .{});
    print("  alias <name>  Generate shell function wrapper\n", .{});
    print("\nGlobal flags:\n", .{});
    print("  -n, --non-interactive  Run without prompts (for testing)\n", .{});
    print("  --no-tty               Force number-based selection (disable arrow keys)\n", .{});
    print("  -h, --help             Show help\n", .{});
    print("  -v, --version          Show version\n", .{});
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
    print("  git-wt rm branch1 branch2   Remove multiple worktrees at once\n", .{});
    print("  git-wt go                   Interactively select and navigate to a worktree\n", .{});
    print("  git-wt go main              Navigate to the main repository\n", .{});
    print("  git-wt go feature-branch    Navigate to the 'feature-branch' worktree\n", .{});
    print("  git-wt list                 List all worktrees with details\n", .{});
    print("  git-wt list --plain         List worktrees in machine-readable format\n", .{});
    print("  git-wt alias gwt            Generate shell function for 'gwt' command\n", .{});
    print("\nFor shell integration setup, use: git-wt --help setup\n", .{});
}

fn printSetupHelp() void {
    print("git-wt - Shell Integration Setup\n\n", .{});
    print("Since CLI tools cannot change the parent shell's directory, you need to set up\n", .{});
    print("a shell function wrapper to enable proper directory navigation.\n\n", .{});
    print("SETUP INSTRUCTIONS:\n", .{});
    print("\n1. Generate the shell function:\n", .{});
    print("   git-wt alias gwt\n", .{});
    print("\n2. Add to your shell configuration:\n", .{});
    print("   For zsh (.zshrc):\n", .{});
    print("     echo 'eval \"$(git-wt alias gwt)\"' >> ~/.zshrc\n", .{});
    print("     source ~/.zshrc\n", .{});
    print("\n   For bash (.bashrc):\n", .{});
    print("     echo 'eval \"$(git-wt alias gwt)\"' >> ~/.bashrc\n", .{});
    print("     source ~/.bashrc\n", .{});
    print("\n3. Use the alias for directory navigation:\n", .{});
    print("   gwt new feature-branch    # Creates worktree AND navigates to it\n", .{});
    print("   gwt go main              # Actually changes to main repository\n", .{});
    print("   gwt go feature-branch    # Actually changes to worktree\n", .{});
    print("   gwt rm feature-branch    # Removes the feature-branch worktree\n", .{});
    print("\nNOTE: You can use any alias name instead of 'gwt':\n", .{});
    print("  git-wt alias wt        # Creates 'wt' alias\n", .{});
    print("  git-wt alias gw        # Creates 'gw' alias\n", .{});
    print("\nAdvanced options:\n", .{});
    print("  git-wt alias gwt --plain              # Always use plain output\n", .{});
    print("  git-wt alias gwt --parent-dir '../{{repo}}-trees'  # Custom parent dir\n", .{});
    print("\nWithout this setup, git-wt commands will work but won't change directories.\n", .{});
}


test {
    _ = @import("utils/test_all.zig");
    _ = @import("commands/test_all.zig");
    _ = @import("tests/test_all.zig");
}