const std = @import("std");
const colors = @import("../utils/colors.zig");
const args = @import("../utils/args.zig");

pub fn printHelp() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.writeAll("Usage: git-wt alias <name> [options]\n\n");
    try stdout.writeAll("Generate a shell function wrapper for proper directory navigation.\n\n");
    try stdout.writeAll("Arguments:\n");
    try stdout.writeAll("  <name>                Name of the shell alias to create\n\n");
    try stdout.writeAll("Options:\n");
    try stdout.writeAll("  -h, --help            Show this help message\n");
    try stdout.writeAll("  --no-tty              Forward --no-tty flag to all commands\n");
    try stdout.writeAll("  -n, --non-interactive Forward --non-interactive flag to all commands\n");
    try stdout.writeAll("  --plain               Forward --plain flag for machine-readable output\n");
    try stdout.writeAll("  --debug               Add debug logging to the shell function\n");
    try stdout.writeAll("  -p, --parent-dir <path>  Set default parent directory for worktrees\n");
    try stdout.writeAll("                           Supports {repo} template substitution\n\n");
    try stdout.writeAll("Examples:\n");
    try stdout.writeAll("  # Basic alias:\n");
    try stdout.writeAll("  git-wt alias gwt\n");
    try stdout.writeAll("  eval \"$(git-wt alias gwt)\"\n\n");
    try stdout.writeAll("  # Alias with flags:\n");
    try stdout.writeAll("  eval \"$(git-wt alias gwt --no-tty --plain)\"\n\n");
    try stdout.writeAll("  # Alias with custom parent directory:\n");
    try stdout.writeAll("  eval \"$(git-wt alias gwt --parent-dir '../{repo}-trees')\"\n");
    try stdout.writeAll("  eval \"$(git-wt alias gwt --parent-dir ~/worktrees/{repo})\"\n\n");
    try stdout.writeAll("Usage in shell:\n");
    try stdout.writeAll("  gwt new feature       # Create worktree and navigate to it\n");
    try stdout.writeAll("  gwt go main          # Navigate to main repository\n");
    try stdout.writeAll("  gwt rm feature       # Remove worktree\n");
}

pub fn execute(allocator: std.mem.Allocator, command_args: []const []const u8, _: bool, _: bool) !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();
    
    // Parse arguments
    var parsed = try args.parseArgs(allocator, command_args);
    defer parsed.deinit();
    
    // Get alias name
    const alias_name = parsed.getPositional(0) orelse {
        try colors.printError(stderr, "Missing alias name", .{});
        try stderr.writeAll("Usage: git-wt alias <name> [options]\n");
        return error.MissingAliasName;
    };
    
    // Get git-wt executable path
    const exe_path = try std.fs.selfExePathAlloc(allocator);
    defer allocator.free(exe_path);
    
    // Parse flags
    const no_tty = parsed.hasFlag(&.{"--no-tty"});
    const non_interactive = parsed.hasFlag(&.{ "--non-interactive", "-n" });
    const plain = parsed.hasFlag(&.{"--plain"});
    const parent_dir = parsed.getFlag(&.{ "--parent-dir", "-p" });
    const debug = parsed.hasFlag(&.{"--debug"});
    
    // Generate shell function
    try stdout.writeAll("# Shell function wrapper for git-wt to enable directory navigation\n");
    try stdout.print("{s}() {{\n", .{alias_name});
    try stdout.print("    local git_wt_bin=\"{s}\"\n", .{exe_path});
    
    // Build flags string
    try stdout.writeAll("    local flags=\"\"\n");
    if (no_tty) try stdout.writeAll("    flags=\"$flags --no-tty\"\n");
    if (non_interactive) try stdout.writeAll("    flags=\"$flags --non-interactive\"\n");
    if (plain) try stdout.writeAll("    flags=\"$flags --plain\"\n");
    
    // Handle parent-dir with {repo} substitution
    if (parent_dir) |dir| {
        if (std.mem.indexOf(u8, dir, "{repo}") != null) {
            // Dynamic parent dir - need to get repo name at runtime
            try stdout.writeAll("    # Get repository name for parent-dir substitution\n");
            try stdout.writeAll("    local repo_name=\"\"\n");
            try stdout.writeAll("    if [ -d .git ] || git rev-parse --git-dir >/dev/null 2>&1; then\n");
            try stdout.writeAll("        repo_name=$(basename \"$(git rev-parse --show-toplevel 2>/dev/null)\" 2>/dev/null)\n");
            try stdout.writeAll("    fi\n");
            try stdout.writeAll("    if [ -n \"$repo_name\" ]; then\n");
            
            // Escape the parent_dir for shell
            const escaped_dir = try escapeShellString(allocator, dir);
            defer allocator.free(escaped_dir);
            
            try stdout.print("        local parent_dir=\"{s}\"\n", .{escaped_dir});
            try stdout.writeAll("        parent_dir=\"${parent_dir//\\{repo\\}/$repo_name}\"\n");
            try stdout.writeAll("        flags=\"$flags --parent-dir \\\"$parent_dir\\\"\"\n");
            try stdout.writeAll("    fi\n");
        } else {
            // Static parent dir
            const escaped_dir = try escapeShellString(allocator, dir);
            defer allocator.free(escaped_dir);
            try stdout.print("    flags=\"$flags --parent-dir \\\"{s}\\\"\"\n", .{escaped_dir});
        }
    }
    
    try stdout.writeAll("    \n");
    try stdout.writeAll("    if [ \"$1\" = \"go\" ]; then\n");
    try stdout.writeAll("        shift\n");
    try stdout.writeAll("        # Check for help flag\n");
    try stdout.writeAll("        for arg in \"$@\"; do\n");
    try stdout.writeAll("            if [[ \"$arg\" = \"--help\" ]] || [[ \"$arg\" = \"-h\" ]]; then\n");
    try stdout.writeAll("                eval \"$git_wt_bin\" go \"$@\" $flags\n");
    try stdout.writeAll("                return\n");
    try stdout.writeAll("            fi\n");
    try stdout.writeAll("        done\n");
    try stdout.writeAll("        # Run go command with fd3 support\n");
    try stdout.writeAll("        # Force --no-tty when no arguments to ensure fd3 works in interactive mode\n");
    try stdout.writeAll("        local cd_cmd\n");
    try stdout.writeAll("        if [ $# -eq 0 ]; then\n");
    try stdout.writeAll("            # Interactive mode - force number selection for reliable fd3\n");
    try stdout.writeAll("            cd_cmd=$(GWT_USE_FD3=1 eval \"$git_wt_bin\" go --no-tty $flags 3>&1 1>&2)\n");
    try stdout.writeAll("        else\n");
    try stdout.writeAll("            # Direct branch navigation\n");
    try stdout.writeAll("            cd_cmd=$(GWT_USE_FD3=1 eval \"$git_wt_bin\" go \"$@\" $flags 3>&1 1>&2)\n");
    try stdout.writeAll("        fi\n");
    try stdout.writeAll("        local exit_code=$?\n");
    if (debug) {
        try stdout.writeAll("        [ -n \"$cd_cmd\" ] && echo \"[DEBUG] cd_cmd: '$cd_cmd'\" >&2\n");
    }
    try stdout.writeAll("        if [ $exit_code -eq 0 ] && [ -n \"$cd_cmd\" ] && echo \"$cd_cmd\" | grep -q '^cd '; then\n");
    try stdout.writeAll("            eval \"$cd_cmd\"\n");
    try stdout.writeAll("        fi\n");
    try stdout.writeAll("    elif [ \"$1\" = \"new\" ]; then\n");
    try stdout.writeAll("        shift\n");
    try stdout.writeAll("        local branch=\"$1\"\n");
    try stdout.writeAll("        eval \"$git_wt_bin\" new \"$@\" $flags\n");
    try stdout.writeAll("        if [ $? -eq 0 ] && [ -n \"$branch\" ] && [[ \"$branch\" != -* ]]; then\n");
    try stdout.writeAll("            local cd_cmd=$(GWT_USE_FD3=1 eval \"$git_wt_bin\" go --show-command \"$branch\" $flags 3>&1 1>&2)\n");
    if (debug) {
        try stdout.writeAll("            [ -n \"$cd_cmd\" ] && echo \"[DEBUG] cd_cmd: '$cd_cmd'\" >&2\n");
    }
    try stdout.writeAll("            if [ -n \"$cd_cmd\" ] && echo \"$cd_cmd\" | grep -q '^cd '; then\n");
    try stdout.writeAll("                eval \"$cd_cmd\"\n");
    try stdout.writeAll("            fi\n");
    try stdout.writeAll("        fi\n");
    try stdout.writeAll("    else\n");
    try stdout.writeAll("        eval \"$git_wt_bin\" \"$@\" $flags\n");
    try stdout.writeAll("    fi\n");
    try stdout.writeAll("}\n");
}

fn escapeShellString(allocator: std.mem.Allocator, str: []const u8) ![]u8 {
    var result = std.ArrayList(u8).init(allocator);
    errdefer result.deinit();
    
    for (str) |c| {
        switch (c) {
            '"', '$', '`', '\\' => {
                try result.append('\\');
                try result.append(c);
            },
            else => try result.append(c),
        }
    }
    
    return result.toOwnedSlice();
}