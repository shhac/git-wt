const std = @import("std");

/// Parsed command arguments
pub const ParsedArgs = struct {
    /// Positional arguments (non-flag arguments)
    positional: std.ArrayList([]const u8),
    /// Flag arguments mapped to their values (or null for boolean flags)
    flags: std.StringHashMap(?[]const u8),
    allocator: std.mem.Allocator,

    pub fn deinit(self: *ParsedArgs) void {
        self.positional.deinit();
        self.flags.deinit();
    }

    /// Check if a flag exists (supports multiple aliases)
    pub fn hasFlag(self: *const ParsedArgs, names: []const []const u8) bool {
        for (names) |name| {
            if (self.flags.contains(name)) return true;
        }
        return false;
    }

    /// Get flag value (supports multiple aliases)
    pub fn getFlag(self: *const ParsedArgs, names: []const []const u8) ?[]const u8 {
        for (names) |name| {
            if (self.flags.get(name)) |value| return value;
        }
        return null;
    }

    /// Get first positional argument
    pub fn getPositional(self: *const ParsedArgs, index: usize) ?[]const u8 {
        if (index >= self.positional.items.len) return null;
        return self.positional.items[index];
    }
};

/// Parse command-line arguments into flags and positional arguments
pub fn parseArgs(allocator: std.mem.Allocator, args: []const []const u8) !ParsedArgs {
    var result = ParsedArgs{
        .positional = std.ArrayList([]const u8).init(allocator),
        .flags = std.StringHashMap(?[]const u8).init(allocator),
        .allocator = allocator,
    };
    errdefer result.deinit();

    var i: usize = 0;
    while (i < args.len) : (i += 1) {
        const arg = args[i];
        
        if (arg.len > 0 and arg[0] == '-') {
            // It's a flag
            const flag_name = arg;
            
            // Check if next arg is the value (not another flag)
            const has_value = (i + 1 < args.len) and 
                             (args[i + 1].len == 0 or args[i + 1][0] != '-');
            
            if (has_value) {
                i += 1;
                try result.flags.put(flag_name, args[i]);
            } else {
                try result.flags.put(flag_name, null);
            }
        } else {
            // It's a positional argument
            try result.positional.append(arg);
        }
    }

    return result;
}

test "parseArgs basic" {
    const allocator = std.testing.allocator;
    
    // Test basic parsing
    const args = [_][]const u8{ "branch-name", "--force", "--parent-dir", "/tmp/test" };
    var parsed = try parseArgs(allocator, &args);
    defer parsed.deinit();
    
    // Check positional args
    try std.testing.expectEqualStrings("branch-name", parsed.getPositional(0).?);
    try std.testing.expect(parsed.getPositional(1) == null);
    
    // Check flags
    try std.testing.expect(parsed.hasFlag(&.{"--force"}));
    try std.testing.expect(parsed.hasFlag(&.{ "--force", "-f" }));
    try std.testing.expectEqualStrings("/tmp/test", parsed.getFlag(&.{"--parent-dir"}).?);
}

test "parseArgs with aliases" {
    const allocator = std.testing.allocator;
    
    const args = [_][]const u8{ "-n", "-p", "/custom", "feature" };
    var parsed = try parseArgs(allocator, &args);
    defer parsed.deinit();
    
    // Check using aliases
    try std.testing.expect(parsed.hasFlag(&.{ "--non-interactive", "-n" }));
    try std.testing.expectEqualStrings("/custom", parsed.getFlag(&.{ "--parent-dir", "-p" }).?);
    try std.testing.expectEqualStrings("feature", parsed.getPositional(0).?);
}

test "parseArgs empty and edge cases" {
    const allocator = std.testing.allocator;
    
    // Empty args
    const empty_args = [_][]const u8{};
    var parsed1 = try parseArgs(allocator, &empty_args);
    defer parsed1.deinit();
    try std.testing.expect(parsed1.positional.items.len == 0);
    
    // Only flags
    const flag_args = [_][]const u8{ "--force", "--debug" };
    var parsed2 = try parseArgs(allocator, &flag_args);
    defer parsed2.deinit();
    try std.testing.expect(parsed2.positional.items.len == 0);
    try std.testing.expect(parsed2.hasFlag(&.{"--force"}));
    try std.testing.expect(parsed2.hasFlag(&.{"--debug"}));
}