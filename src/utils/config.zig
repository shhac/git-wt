const std = @import("std");
const fs = std.fs;
const git = @import("git.zig");

pub const ConfigError = error{
    InvalidConfig,
    ParseError,
};

/// Configuration structure matching the TOML schema
pub const Config = struct {
    // [worktree]
    parent_dir: ?[]const u8 = null,

    // [behavior]
    auto_confirm: bool = false,
    non_interactive: bool = false,
    plain_output: bool = false,
    json_output: bool = false,

    // [ui]
    no_color: bool = false,
    no_tty: bool = false,

    // [sync]
    extra_files: [][]const u8 = &.{},
    exclude_files: [][]const u8 = &.{},

    pub fn deinit(self: *Config, allocator: std.mem.Allocator) void {
        if (self.parent_dir) |dir| allocator.free(dir);
        for (self.extra_files) |file| allocator.free(file);
        if (self.extra_files.len > 0) allocator.free(self.extra_files);
        for (self.exclude_files) |file| allocator.free(file);
        if (self.exclude_files.len > 0) allocator.free(self.exclude_files);
    }
};

/// Load and merge configuration from multiple sources
/// Priority (highest to lowest):
/// 1. Command-line flags (handled by caller)
/// 2. Environment variables (handled by caller)
/// 3. Project config (.git-wt.toml in repo root)
/// 4. User config (~/.config/git-wt/config)
/// 5. Built-in defaults
pub fn loadConfig(allocator: std.mem.Allocator) !Config {
    var config = Config{};

    // Try to load user config first
    if (loadUserConfig(allocator)) |user_config| {
        mergeConfig(&config, user_config, allocator);
        var user_copy = user_config;
        user_copy.deinit(allocator);
    } else |_| {
        // User config doesn't exist or failed to parse, that's okay
    }

    // Try to load project config (overrides user config)
    if (loadProjectConfig(allocator)) |project_config| {
        mergeConfig(&config, project_config, allocator);
        var proj_copy = project_config;
        proj_copy.deinit(allocator);
    } else |_| {
        // Project config doesn't exist or failed to parse, that's okay
    }

    return config;
}

/// Load user-level config from ~/.config/git-wt/config
fn loadUserConfig(allocator: std.mem.Allocator) !Config {
    const home = std.posix.getenv("HOME") orelse return error.InvalidConfig;
    const config_path = try std.fmt.allocPrint(allocator, "{s}/.config/git-wt/config", .{home});
    defer allocator.free(config_path);

    return parseConfigFile(allocator, config_path);
}

/// Load project-level config from .git-wt.toml in repo root
fn loadProjectConfig(allocator: std.mem.Allocator) !Config {
    const repo_info = git.getRepoInfo(allocator) catch return error.InvalidConfig;
    defer allocator.free(repo_info.root);
    defer if (repo_info.main_repo_root) |root| allocator.free(root);

    const config_path = try std.fmt.allocPrint(allocator, "{s}/.git-wt.toml", .{repo_info.root});
    defer allocator.free(config_path);

    return parseConfigFile(allocator, config_path);
}

/// Simple TOML parser (supports basic key=value format)
fn parseConfigFile(allocator: std.mem.Allocator, path: []const u8) !Config {
    const file = fs.cwd().openFile(path, .{}) catch return error.InvalidConfig;
    defer file.close();

    const content = try file.readToEndAlloc(allocator, 1024 * 1024); // 1MB max
    defer allocator.free(content);

    return parseConfigContent(allocator, content);
}

/// Parse TOML content into Config struct
fn parseConfigContent(allocator: std.mem.Allocator, content: []const u8) !Config {
    var config = Config{};
    var extra_files = std.ArrayList([]const u8).empty;
    defer extra_files.deinit(allocator);
    var exclude_files = std.ArrayList([]const u8).empty;
    defer exclude_files.deinit(allocator);

    var current_section: []const u8 = "";
    var lines = std.mem.splitSequence(u8, content, "\n");
    var in_array = false;
    var array_key: []const u8 = "";

    while (lines.next()) |line| {
        const trimmed = std.mem.trim(u8, line, " \t\r");

        // Skip empty lines and comments
        if (trimmed.len == 0 or trimmed[0] == '#') continue;

        // Section headers [section]
        if (trimmed[0] == '[' and trimmed[trimmed.len - 1] == ']') {
            current_section = trimmed[1 .. trimmed.len - 1];
            in_array = false;
            continue;
        }

        // Handle array continuation
        if (in_array) {
            if (std.mem.indexOf(u8, trimmed, "]") != null) {
                // End of array
                in_array = false;
                // Parse any values on this line before the ]
                if (std.mem.indexOf(u8, trimmed, "\"")) |_| {
                    try parseArrayValue(allocator, trimmed, array_key, &extra_files, &exclude_files);
                }
            } else {
                // Array value line
                try parseArrayValue(allocator, trimmed, array_key, &extra_files, &exclude_files);
            }
            continue;
        }

        // Key-value pairs
        if (std.mem.indexOf(u8, trimmed, "=")) |eq_pos| {
            const key = std.mem.trim(u8, trimmed[0..eq_pos], " \t");
            const value = std.mem.trim(u8, trimmed[eq_pos + 1 ..], " \t");

            // Check if this is the start of an array
            if (value.len > 0 and value[0] == '[') {
                in_array = true;
                array_key = key;
                // Check if array closes on same line
                if (std.mem.indexOf(u8, value, "]") != null) {
                    in_array = false;
                    // Parse inline array
                    try parseArrayValue(allocator, value, key, &extra_files, &exclude_files);
                } else {
                    // Parse any values on this line
                    try parseArrayValue(allocator, value, key, &extra_files, &exclude_files);
                }
                continue;
            }

            try parseKeyValue(allocator, &config, current_section, key, value);
        }
    }

    // Assign collected arrays
    if (extra_files.items.len > 0) {
        config.extra_files = try extra_files.toOwnedSlice(allocator);
    }
    if (exclude_files.items.len > 0) {
        config.exclude_files = try exclude_files.toOwnedSlice(allocator);
    }

    return config;
}

/// Parse a single key-value pair
fn parseKeyValue(allocator: std.mem.Allocator, config: *Config, section: []const u8, key: []const u8, value: []const u8) !void {
    if (std.mem.eql(u8, section, "worktree")) {
        if (std.mem.eql(u8, key, "parent_dir")) {
            config.parent_dir = try parseString(allocator, value);
        }
    } else if (std.mem.eql(u8, section, "behavior")) {
        if (std.mem.eql(u8, key, "auto_confirm")) {
            config.auto_confirm = try parseBool(value);
        } else if (std.mem.eql(u8, key, "non_interactive")) {
            config.non_interactive = try parseBool(value);
        } else if (std.mem.eql(u8, key, "plain_output")) {
            config.plain_output = try parseBool(value);
        } else if (std.mem.eql(u8, key, "json_output")) {
            config.json_output = try parseBool(value);
        }
    } else if (std.mem.eql(u8, section, "ui")) {
        if (std.mem.eql(u8, key, "no_color")) {
            config.no_color = try parseBool(value);
        } else if (std.mem.eql(u8, key, "no_tty")) {
            config.no_tty = try parseBool(value);
        }
    }
}

/// Parse array values from TOML
fn parseArrayValue(allocator: std.mem.Allocator, line: []const u8, key: []const u8, extra_files: *std.ArrayList([]const u8), exclude_files: *std.ArrayList([]const u8)) !void {
    // Extract strings from line (format: "value" or ["value", "value"])
    var start: ?usize = null;
    var i: usize = 0;
    while (i < line.len) : (i += 1) {
        if (line[i] == '"') {
            if (start) |s| {
                // End of string
                const str = try allocator.dupe(u8, line[s + 1 .. i]);
                if (std.mem.eql(u8, key, "extra_files")) {
                    try extra_files.append(allocator, str);
                } else if (std.mem.eql(u8, key, "exclude_files")) {
                    try exclude_files.append(allocator, str);
                }
                start = null;
            } else {
                // Start of string
                start = i;
            }
        }
    }
}

/// Parse string value (handles quoted strings)
fn parseString(allocator: std.mem.Allocator, value: []const u8) ![]const u8 {
    if (value.len >= 2 and value[0] == '"' and value[value.len - 1] == '"') {
        return try allocator.dupe(u8, value[1 .. value.len - 1]);
    }
    return try allocator.dupe(u8, value);
}

/// Parse boolean value
fn parseBool(value: []const u8) !bool {
    if (std.mem.eql(u8, value, "true")) return true;
    if (std.mem.eql(u8, value, "false")) return false;
    return error.ParseError;
}

/// Merge configs (overlay takes precedence)
fn mergeConfig(base: *Config, overlay: Config, allocator: std.mem.Allocator) void {
    if (overlay.parent_dir) |dir| {
        if (base.parent_dir) |old| allocator.free(old);
        base.parent_dir = allocator.dupe(u8, dir) catch null;
    }
    if (overlay.auto_confirm) base.auto_confirm = true;
    if (overlay.non_interactive) base.non_interactive = true;
    if (overlay.plain_output) base.plain_output = true;
    if (overlay.json_output) base.json_output = true;
    if (overlay.no_color) base.no_color = true;
    if (overlay.no_tty) base.no_tty = true;

    // Merge arrays (append)
    if (overlay.extra_files.len > 0) {
        var new_list = std.ArrayList([]const u8).empty;
        for (base.extra_files) |f| {
            new_list.append(allocator, allocator.dupe(u8, f) catch continue) catch continue;
        }
        for (overlay.extra_files) |f| {
            new_list.append(allocator, allocator.dupe(u8, f) catch continue) catch continue;
        }
        if (base.extra_files.len > 0) {
            for (base.extra_files) |f| allocator.free(f);
            allocator.free(base.extra_files);
        }
        base.extra_files = new_list.toOwnedSlice(allocator) catch &.{};
    }

    if (overlay.exclude_files.len > 0) {
        var new_list = std.ArrayList([]const u8).empty;
        for (base.exclude_files) |f| {
            new_list.append(allocator, allocator.dupe(u8, f) catch continue) catch continue;
        }
        for (overlay.exclude_files) |f| {
            new_list.append(allocator, allocator.dupe(u8, f) catch continue) catch continue;
        }
        if (base.exclude_files.len > 0) {
            for (base.exclude_files) |f| allocator.free(f);
            allocator.free(base.exclude_files);
        }
        base.exclude_files = new_list.toOwnedSlice(allocator) catch &.{};
    }
}

/// Resolve parent directory path with {repo} substitution
/// Relative paths are resolved from repo root
pub fn resolveParentDir(allocator: std.mem.Allocator, parent_dir: []const u8, repo_name: []const u8, repo_root: []const u8) ![]const u8 {
    // Substitute {repo} with repository name
    var result = std.ArrayList(u8).empty;
    defer result.deinit(allocator);

    var i: usize = 0;
    while (i < parent_dir.len) {
        if (i + 6 <= parent_dir.len and std.mem.eql(u8, parent_dir[i .. i + 6], "{repo}")) {
            try result.appendSlice(allocator, repo_name);
            i += 6;
        } else {
            try result.append(allocator, parent_dir[i]);
            i += 1;
        }
    }

    const substituted = try result.toOwnedSlice(allocator);
    defer allocator.free(substituted);

    // If absolute path, return as-is
    if (substituted.len > 0 and substituted[0] == '/') {
        return try allocator.dupe(u8, substituted);
    }

    // If starts with ~, expand home directory
    if (substituted.len > 0 and substituted[0] == '~') {
        const home = std.posix.getenv("HOME") orelse return error.InvalidConfig;
        if (substituted.len == 1) {
            return try allocator.dupe(u8, home);
        }
        // ~/path -> $HOME/path
        return try std.fmt.allocPrint(allocator, "{s}{s}", .{ home, substituted[1..] });
    }

    // Relative path - resolve from repo root
    return try fs.path.join(allocator, &.{ repo_root, substituted });
}

// Tests
test "parseConfigContent: basic values" {
    const allocator = std.testing.allocator;

    const content =
        \\[worktree]
        \\parent_dir = "../{repo}-trees"
        \\
        \\[behavior]
        \\auto_confirm = false
        \\non_interactive = true
    ;

    var config = try parseConfigContent(allocator, content);
    defer config.deinit(allocator);

    try std.testing.expect(config.parent_dir != null);
    try std.testing.expectEqualStrings("../{repo}-trees", config.parent_dir.?);
    try std.testing.expect(config.non_interactive == true);
    try std.testing.expect(config.auto_confirm == false);
}

test "parseConfigContent: arrays" {
    const allocator = std.testing.allocator;

    const content =
        \\[sync]
        \\extra_files = [
        \\    ".vscode/settings.json",
        \\    ".idea/"
        \\]
    ;

    var config = try parseConfigContent(allocator, content);
    defer config.deinit(allocator);

    try std.testing.expect(config.extra_files.len == 2);
    try std.testing.expectEqualStrings(".vscode/settings.json", config.extra_files[0]);
    try std.testing.expectEqualStrings(".idea/", config.extra_files[1]);
}

test "resolveParentDir: repo substitution" {
    const allocator = std.testing.allocator;

    const result = try resolveParentDir(allocator, "../{repo}-trees", "my-repo", "/home/user/my-repo");
    defer allocator.free(result);

    try std.testing.expectEqualStrings("/home/user/my-repo/../my-repo-trees", result);
}

test "resolveParentDir: absolute path" {
    const allocator = std.testing.allocator;

    const result = try resolveParentDir(allocator, "/tmp/worktrees", "my-repo", "/home/user/my-repo");
    defer allocator.free(result);

    try std.testing.expectEqualStrings("/tmp/worktrees", result);
}

test "resolveParentDir: relative path" {
    const allocator = std.testing.allocator;

    const result = try resolveParentDir(allocator, "worktrees", "my-repo", "/home/user/my-repo");
    defer allocator.free(result);

    try std.testing.expectEqualStrings("/home/user/my-repo/worktrees", result);
}
