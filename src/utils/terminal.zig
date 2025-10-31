const std = @import("std");

/// Check if terminal supports UTF-8 encoding
/// Checks LANG and LC_CTYPE environment variables for UTF-8 markers
pub fn supportsUtf8() bool {
    // Check LC_CTYPE first (more specific)
    if (std.process.getEnvVarOwned(std.heap.page_allocator, "LC_CTYPE")) |lc_ctype| {
        defer std.heap.page_allocator.free(lc_ctype);
        if (std.mem.indexOf(u8, lc_ctype, "UTF-8") != null or std.mem.indexOf(u8, lc_ctype, "utf8") != null) {
            return true;
        }
    } else |_| {}

    // Fall back to LANG
    if (std.process.getEnvVarOwned(std.heap.page_allocator, "LANG")) |lang| {
        defer std.heap.page_allocator.free(lang);
        if (std.mem.indexOf(u8, lang, "UTF-8") != null or std.mem.indexOf(u8, lang, "utf8") != null) {
            return true;
        }
    } else |_| {}

    // Default to false if no UTF-8 indicators found
    return false;
}

/// Check if terminal supports colors
/// Checks NO_COLOR environment variable and TERM variable
pub fn supportsColor() bool {
    // Check NO_COLOR first (explicit opt-out)
    // https://no-color.org/
    if (std.process.getEnvVarOwned(std.heap.page_allocator, "NO_COLOR")) |no_color| {
        defer std.heap.page_allocator.free(no_color);
        // Any value for NO_COLOR means colors are disabled
        if (no_color.len > 0) {
            return false;
        }
    } else |_| {}

    // Check TERM variable
    if (std.process.getEnvVarOwned(std.heap.page_allocator, "TERM")) |term| {
        defer std.heap.page_allocator.free(term);

        // Terminals known to NOT support colors
        if (std.mem.eql(u8, term, "dumb")) {
            return false;
        }

        // Most modern terminals support colors
        // Common values: xterm, xterm-256color, screen, tmux, etc.
        return true;
    } else |_| {}

    // If no TERM variable, assume no color support
    return false;
}

/// Check if terminal supports ANSI escape codes
/// This is closely related to color support
pub fn supportsAnsi() bool {
    // ANSI support is typically tied to color support
    // Most terminals that support colors also support ANSI codes
    return supportsColor();
}

test "UTF-8 detection" {
    // Basic test that the function runs without crashing
    _ = supportsUtf8();
}

test "color detection" {
    // Basic test that the function runs without crashing
    _ = supportsColor();
}

test "ANSI detection" {
    // Basic test that the function runs without crashing
    _ = supportsAnsi();
}
