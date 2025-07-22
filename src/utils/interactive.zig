const std = @import("std");
const posix = std.posix;
const colors = @import("colors.zig");
const git = @import("git.zig");
const time = @import("time.zig");

/// Global state for signal handling
var g_original_termios: ?posix.termios = null;
var g_is_raw_mode = std.atomic.Value(bool).init(false);
var g_signal_mutex = std.Thread.Mutex{};
var g_needs_redraw = std.atomic.Value(bool).init(false);

/// Signal handler for SIGINT
fn handleSignal(_: i32) callconv(.C) void {
    // Check if we're in raw mode and get termios atomically
    var termios_copy: posix.termios = undefined;
    var should_restore = false;
    
    {
        g_signal_mutex.lock();
        defer g_signal_mutex.unlock();
        
        if (g_is_raw_mode.load(.acquire) and g_original_termios != null) {
            termios_copy = g_original_termios.?;
            should_restore = true;
        }
    }
    
    // Restore terminal state outside of mutex lock
    if (should_restore) {
        posix.tcsetattr(std.io.getStdIn().handle, .FLUSH, termios_copy) catch {};
        showCursor() catch {};
    }
    
    // Exit the program
    std.process.exit(130); // 128 + SIGINT(2)
}

/// Signal handler for SIGWINCH (window size change)
fn handleWinch(_: i32) callconv(.C) void {
    // Set flag to trigger redraw on next iteration
    g_needs_redraw.store(true, .release);
}

/// Check if stdin is a TTY (terminal)
pub fn isStdinTty() bool {
    return posix.isatty(std.io.getStdIn().handle);
}

/// Check if stdout is a TTY (terminal)
pub fn isStdoutTty() bool {
    return posix.isatty(std.io.getStdOut().handle);
}

/// Terminal control for raw mode
pub const RawMode = struct {
    original_termios: posix.termios,
    is_raw: std.atomic.Value(bool) = std.atomic.Value(bool).init(false),
    
    /// Enter raw mode for single character input
    pub fn enter(self: *RawMode) !void {
        if (self.is_raw.load(.acquire)) return;
        
        // Get current terminal settings
        self.original_termios = try posix.tcgetattr(std.io.getStdIn().handle);
        
        // Create raw mode settings
        var raw = self.original_termios;
        
        // Disable canonical mode and echo
        raw.lflag.ICANON = false;
        raw.lflag.ECHO = false;
        raw.lflag.ISIG = true;   // Keep signals enabled for Ctrl+C
        raw.lflag.IEXTEN = false;
        
        // Disable input processing
        raw.iflag.IXON = false;
        raw.iflag.ICRNL = false;
        raw.iflag.BRKINT = false;
        raw.iflag.INPCK = false;
        raw.iflag.ISTRIP = false;
        
        // Set character-at-a-time input with timeout
        raw.cc[@intFromEnum(posix.V.TIME)] = 1; // 0.1 second timeout
        raw.cc[@intFromEnum(posix.V.MIN)] = 0;   // Don't block
        
        try posix.tcsetattr(std.io.getStdIn().handle, .FLUSH, raw);
        self.is_raw.store(true, .release);
        
        // Register for signal handling
        g_signal_mutex.lock();
        defer g_signal_mutex.unlock();
        g_original_termios = self.original_termios;
        g_is_raw_mode.store(true, .release);
    }
    
    /// Exit raw mode and restore original settings
    pub fn exit(self: *RawMode) void {
        if (!self.is_raw.load(.acquire)) return;
        
        // Unregister from signal handling atomically
        {
            g_signal_mutex.lock();
            defer g_signal_mutex.unlock();
            g_is_raw_mode.store(false, .release);
            g_original_termios = null;
        }
        
        // Restore terminal settings outside of mutex lock
        posix.tcsetattr(std.io.getStdIn().handle, .FLUSH, self.original_termios) catch {};
        self.is_raw.store(false, .release);
    }
};

/// Key types for input handling
pub const Key = enum {
    up,
    down,
    enter,
    space,
    escape,
    char,
    none, // timeout or no input
};

/// Read a key press in raw mode
pub fn readKey() !struct { key: Key, char: u8 } {
    const stdin = std.io.getStdIn().reader();
    var buf: [3]u8 = undefined;
    
    // Try to read first byte
    const n = stdin.read(buf[0..1]) catch |err| {
        if (err == error.WouldBlock) {
            return .{ .key = .none, .char = 0 };
        }
        return err;
    };
    
    if (n == 0) return .{ .key = .none, .char = 0 };
    
    const first = buf[0];
    
    // Check for special keys
    switch (first) {
        27 => { // ESC sequence
            // Try to read more bytes for arrow keys
            const n2 = stdin.read(buf[1..2]) catch 0;
            if (n2 == 0) return .{ .key = .escape, .char = 27 };
            
            if (buf[1] == '[') {
                const n3 = stdin.read(buf[2..3]) catch 0;
                if (n3 > 0) {
                    switch (buf[2]) {
                        'A' => return .{ .key = .up, .char = 0 },
                        'B' => return .{ .key = .down, .char = 0 },
                        else => {},
                    }
                }
            }
            return .{ .key = .escape, .char = 27 };
        },
        '\r', '\n' => return .{ .key = .enter, .char = first },
        ' ' => return .{ .key = .space, .char = first },
        else => return .{ .key = .char, .char = first },
    }
}

/// Clear current line and move cursor to start
pub fn clearLine() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("\r\x1b[2K", .{});
}

/// Move cursor up n lines
pub fn moveCursorUp(n: usize) !void {
    if (n == 0) return;
    const stdout = std.io.getStdOut().writer();
    try stdout.print("\x1b[{d}A", .{n});
}

/// Move cursor down n lines  
pub fn moveCursorDown(n: usize) !void {
    if (n == 0) return;
    const stdout = std.io.getStdOut().writer();
    try stdout.print("\x1b[{d}B", .{n});
}

/// Hide cursor
pub fn hideCursor() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("\x1b[?25l", .{});
}

/// Show cursor
pub fn showCursor() !void {
    const stdout = std.io.getStdOut().writer();
    try stdout.print("\x1b[?25h", .{});
}

/// Interactive selection result
pub const SelectionResult = enum {
    selected,
    cancelled,
    failed,
};

/// Options for interactive selection
pub const SelectOptions = struct {
    prompt: []const u8 = "Select an option:",
    show_instructions: bool = true,
    use_colors: bool = true,
    multi_select: bool = false,
};

/// Render a selection list item with enhanced formatting
fn renderItem(
    writer: anytype,
    item_text: []const u8, 
    is_current: bool,
    is_selected: bool,
    use_colors: bool,
    multi_select_mode: bool,
) !void {
    if (use_colors) {
        if (multi_select_mode) {
            // Multi-select mode: show checkbox and highlight current item
            const checkbox = if (is_selected) "☑" else "☐";
            if (is_current) {
                // Current item: highlighted background
                try writer.print("  {s}{s} {s}{s}{s}\n", .{
                    "\x1b[7m", // reverse video (highlight)
                    checkbox,
                    item_text,
                    colors.reset,
                    colors.reset,
                });
            } else if (is_selected) {
                // Selected but not current: green checkbox
                try writer.print("  {s}{s}{s} {s}\n", .{
                    colors.green,
                    checkbox,
                    colors.reset,
                    item_text,
                });
            } else {
                // Unselected: dim checkbox
                try writer.print("  {s}{s}{s} {s}\n", .{
                    "\x1b[2m", // dim
                    checkbox,
                    colors.reset,
                    item_text,
                });
            }
        } else {
            // Single-select mode: original behavior
            if (is_current) {
                // Current item: green brackets with bright green asterisk and bold text
                try writer.print("  {s}[{s}{s}*{s}{s}]{s} {s}{s}{s}\n", .{
                    colors.green,          // green [
                    colors.reset,          // reset to remove any inherited formatting
                    "\x1b[1m\x1b[92m",    // bright green + bold for *
                    colors.reset,          // reset to clear bold
                    colors.green,          // back to green for ]
                    colors.reset,          // reset before text
                    "\x1b[1m",            // bold for text
                    item_text,
                    colors.reset,          // final reset
                });
            } else {
                // Non-current item: dim [ ] with normal text
                try writer.print("  {s}[ ]{s} {s}\n", .{
                    "\x1b[2m", // dim
                    colors.reset,
                    item_text,
                });
            }
        }
    } else {
        if (multi_select_mode) {
            const checkbox = if (is_selected) "[X]" else "[ ]";
            const indicator = if (is_current) " <--" else "";
            try writer.print("  {s} {s}{s}\n", .{ checkbox, item_text, indicator });
        } else {
            try writer.print("  [{s}] {s}\n", .{
                if (is_current) "*" else " ",
                item_text,
            });
        }
    }
}

/// Display an interactive selection list
/// Returns the selected index or null if cancelled (single-select)
/// For multi-select, use selectMultipleFromList
pub fn selectFromList(
    allocator: std.mem.Allocator,
    items: []const []const u8,
    options: SelectOptions,
) !?usize {
    _ = allocator; // Reserved for future use
    if (items.len == 0) return null;
    
    // Check if we're in a TTY
    if (!isStdinTty() or !isStdoutTty()) {
        // Fall back to non-interactive mode
        return null;
    }
    
    const stdout = std.io.getStdOut().writer();
    var selected: usize = 0;
    
    // Install signal handler for SIGINT
    var sigaction = posix.Sigaction{
        .handler = .{ .handler = handleSignal },
        .mask = posix.empty_sigset,
        .flags = 0,
    };
    var old_sigaction: posix.Sigaction = undefined;
    posix.sigaction(posix.SIG.INT, &sigaction, &old_sigaction);
    defer {
        // Restore original signal handler
        posix.sigaction(posix.SIG.INT, &old_sigaction, null);
    }
    
    // Install signal handler for SIGWINCH
    var winch_action = posix.Sigaction{
        .handler = .{ .handler = handleWinch },
        .mask = posix.empty_sigset,
        .flags = 0,
    };
    var old_winch_action: posix.Sigaction = undefined;
    posix.sigaction(posix.SIG.WINCH, &winch_action, &old_winch_action);
    defer {
        // Restore original signal handler
        posix.sigaction(posix.SIG.WINCH, &old_winch_action, null);
    }
    
    // Set up raw mode
    var raw_mode = RawMode{ .original_termios = undefined };
    try raw_mode.enter();
    defer raw_mode.exit();
    
    // Hide cursor
    try hideCursor();
    defer showCursor() catch {};
    
    // Initial render with newline for spacing
    try stdout.print("\n", .{});
    for (items, 0..) |item, i| {
        try renderItem(stdout, item, i == selected, false, options.use_colors, false);
    }
    
    if (options.show_instructions) {
        try stdout.print("\n{s}↑/↓{s} Navigate  {s}Enter/Space{s} Select  {s}ESC{s} Cancel\n", .{
            if (options.use_colors) colors.yellow else "",
            if (options.use_colors) colors.reset else "",
            if (options.use_colors) colors.yellow else "",
            if (options.use_colors) colors.reset else "",
            if (options.use_colors) colors.yellow else "",
            if (options.use_colors) colors.reset else "",
        });
    }
    
    // Input loop
    while (true) {
        // Check if terminal was resized
        if (g_needs_redraw.swap(false, .acq_rel)) {
            // Clear entire screen and redraw from scratch
            try stdout.print("\x1b[2J\x1b[H", .{}); // Clear screen and move to home
            
            // Redraw everything
            try stdout.print("\n", .{});
            for (items, 0..) |item, i| {
                try renderItem(stdout, item, i == selected, false, options.use_colors, false);
            }
            
            if (options.show_instructions) {
                try stdout.print("\n{s}↑/↓{s} Navigate  {s}Enter/Space{s} Select  {s}ESC{s} Cancel\n", .{
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                });
            }
        }
        
        const key_info = try readKey();
        
        var needs_redraw = false;
        
        switch (key_info.key) {
            .up => {
                if (selected > 0) {
                    selected -= 1;
                    needs_redraw = true;
                }
            },
            .down => {
                if (selected < items.len - 1) {
                    selected += 1;
                    needs_redraw = true;
                }
            },
            .enter, .space => {
                // Clear the selection display including initial newline
                const total_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
                // Move up one extra line to account for initial newline
                try moveCursorUp(total_lines + 1);
                try clearLine(); // Clear the initial empty line
                try stdout.print("\n", .{});
                for (0..total_lines) |_| {
                    try clearLine();
                    try stdout.print("\n", .{});
                }
                try moveCursorUp(total_lines + 1);
                
                return selected;
            },
            .escape => {
                // Clear the selection display including initial newline
                const total_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
                // Move up one extra line to account for initial newline
                try moveCursorUp(total_lines + 1);
                try clearLine(); // Clear the initial empty line
                try stdout.print("\n", .{});
                for (0..total_lines) |_| {
                    try clearLine();
                    try stdout.print("\n", .{});
                }
                try moveCursorUp(total_lines + 1);
                
                return null;
            },
            .char => {
                // Handle 'q' for quit
                if (key_info.char == 'q' or key_info.char == 'Q') {
                    // Clear the selection display including initial newline
                    const total_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
                    // Move up one extra line to account for initial newline
                    try moveCursorUp(total_lines + 1);
                    try clearLine(); // Clear the initial empty line
                    try stdout.print("\n", .{});
                    for (0..total_lines) |_| {
                        try clearLine();
                        try stdout.print("\n", .{});
                    }
                    try moveCursorUp(total_lines + 1);
                    
                    return null;
                }
            },
            .none => continue,
        }
        
        // Redraw if selection changed
        if (needs_redraw) {
            // Move back to start of list
            const redraw_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
            try moveCursorUp(redraw_lines);
            
            // Redraw all items
            for (items, 0..) |item, i| {
                try clearLine();
                try renderItem(stdout, item, i == selected, false, options.use_colors, false);
            }
            
            // Redraw instructions
            if (options.show_instructions) {
                try clearLine();
                try stdout.print("\n{s}↑/↓{s} Navigate  {s}Enter/Space{s} Select  {s}ESC{s} Cancel\n", .{
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                });
            }
        }
    }
}

/// Display an interactive multi-selection list
/// Returns array of selected indices or null if cancelled
pub fn selectMultipleFromList(
    allocator: std.mem.Allocator,
    items: []const []const u8,
    options: SelectOptions,
) !?[]usize {
    if (items.len == 0) return null;
    
    // Check if we're in a TTY
    if (!isStdinTty() or !isStdoutTty()) {
        // Fall back to non-interactive mode
        return null;
    }
    
    const stdout = std.io.getStdOut().writer();
    var current: usize = 0;
    var selected = std.ArrayList(bool).init(allocator);
    defer selected.deinit();
    
    // Initialize selection state - all unselected
    try selected.appendNTimes(false, items.len);
    
    // Install signal handlers
    var sigaction = posix.Sigaction{
        .handler = .{ .handler = handleSignal },
        .mask = posix.empty_sigset,
        .flags = 0,
    };
    var old_sigaction: posix.Sigaction = undefined;
    posix.sigaction(posix.SIG.INT, &sigaction, &old_sigaction);
    defer {
        posix.sigaction(posix.SIG.INT, &old_sigaction, null);
    }
    
    var winch_action = posix.Sigaction{
        .handler = .{ .handler = handleWinch },
        .mask = posix.empty_sigset,
        .flags = 0,
    };
    var old_winch_action: posix.Sigaction = undefined;
    posix.sigaction(posix.SIG.WINCH, &winch_action, &old_winch_action);
    defer {
        posix.sigaction(posix.SIG.WINCH, &old_winch_action, null);
    }
    
    // Set up raw mode
    var raw_mode = RawMode{ .original_termios = undefined };
    try raw_mode.enter();
    defer raw_mode.exit();
    
    // Hide cursor
    try hideCursor();
    defer showCursor() catch {};
    
    // Initial render
    try stdout.print("\n", .{});
    for (items, 0..) |item, i| {
        try renderItem(stdout, item, i == current, selected.items[i], options.use_colors, true);
    }
    
    if (options.show_instructions) {
        try stdout.print("\n{s}↑/↓{s} Navigate  {s}Space{s} Toggle  {s}Enter{s} Confirm  {s}ESC{s} Cancel\n", .{
            if (options.use_colors) colors.yellow else "",
            if (options.use_colors) colors.reset else "",
            if (options.use_colors) colors.yellow else "",
            if (options.use_colors) colors.reset else "",
            if (options.use_colors) colors.yellow else "",
            if (options.use_colors) colors.reset else "",
            if (options.use_colors) colors.yellow else "",
            if (options.use_colors) colors.reset else "",
        });
    }
    
    // Input loop
    while (true) {
        // Check if terminal was resized
        if (g_needs_redraw.swap(false, .acq_rel)) {
            try stdout.print("\x1b[2J\x1b[H", .{}); // Clear screen and move to home
            
            // Redraw everything
            try stdout.print("\n", .{});
            for (items, 0..) |item, i| {
                try renderItem(stdout, item, i == current, selected.items[i], options.use_colors, true);
            }
            
            if (options.show_instructions) {
                try stdout.print("\n{s}↑/↓{s} Navigate  {s}Space{s} Toggle  {s}Enter{s} Confirm  {s}ESC{s} Cancel\n", .{
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                });
            }
        }
        
        const key_info = try readKey();
        var needs_redraw = false;
        
        switch (key_info.key) {
            .up => {
                if (current > 0) {
                    current -= 1;
                    needs_redraw = true;
                }
            },
            .down => {
                if (current < items.len - 1) {
                    current += 1;
                    needs_redraw = true;
                }
            },
            .space => {
                // Toggle selection
                selected.items[current] = !selected.items[current];
                needs_redraw = true;
            },
            .enter => {
                // Confirm selection or select current if nothing selected
                var has_selection = false;
                for (selected.items) |is_selected| {
                    if (is_selected) {
                        has_selection = true;
                        break;
                    }
                }
                
                if (!has_selection) {
                    // Nothing selected, select current item
                    selected.items[current] = true;
                }
                
                // Clear display
                const total_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
                try moveCursorUp(total_lines + 1);
                try clearLine();
                try stdout.print("\n", .{});
                for (0..total_lines) |_| {
                    try clearLine();
                    try stdout.print("\n", .{});
                }
                try moveCursorUp(total_lines + 1);
                
                // Build result array
                var result = std.ArrayList(usize).init(allocator);
                for (selected.items, 0..) |is_selected, i| {
                    if (is_selected) {
                        try result.append(i);
                    }
                }
                
                return try result.toOwnedSlice();
            },
            .escape => {
                // Clear display
                const total_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
                try moveCursorUp(total_lines + 1);
                try clearLine();
                try stdout.print("\n", .{});
                for (0..total_lines) |_| {
                    try clearLine();
                    try stdout.print("\n", .{});
                }
                try moveCursorUp(total_lines + 1);
                
                return null;
            },
            .char => {
                if (key_info.char == 'q' or key_info.char == 'Q') {
                    // Clear display
                    const total_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
                    try moveCursorUp(total_lines + 1);
                    try clearLine();
                    try stdout.print("\n", .{});
                    for (0..total_lines) |_| {
                        try clearLine();
                        try stdout.print("\n", .{});
                    }
                    try moveCursorUp(total_lines + 1);
                    
                    return null;
                }
            },
            .none => continue,
        }
        
        // Redraw if needed
        if (needs_redraw) {
            const redraw_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
            try moveCursorUp(redraw_lines);
            
            // Redraw all items
            for (items, 0..) |item, i| {
                try clearLine();
                try renderItem(stdout, item, i == current, selected.items[i], options.use_colors, true);
            }
            
            // Redraw instructions
            if (options.show_instructions) {
                try clearLine();
                try stdout.print("\n{s}↑/↓{s} Navigate  {s}Space{s} Toggle  {s}Enter{s} Confirm  {s}ESC{s} Cancel\n", .{
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                    if (options.use_colors) colors.yellow else "",
                    if (options.use_colors) colors.reset else "",
                });
            }
        }
    }
}

/// Format worktree option for display
pub fn formatWorktreeOption(
    allocator: std.mem.Allocator,
    wt_info: git.WorktreeWithTime, 
    use_colors: bool,
) ![]u8 {
    const timestamp = @divFloor(wt_info.mod_time, std.time.ns_per_s);
    const time_ago_seconds = @as(u64, @intCast(std.time.timestamp() - timestamp));
    const duration_str = try time.formatDuration(allocator, time_ago_seconds);
    defer allocator.free(duration_str);
    
    if (use_colors) {
        return try std.fmt.allocPrint(allocator, "{s}{s}{s} @ {s}{s}{s} - {s}{s} ago{s}", .{
            colors.path_color,
            wt_info.display_name,
            colors.reset,
            colors.magenta,
            wt_info.worktree.branch,
            colors.reset,
            colors.yellow,
            duration_str,
            colors.reset,
        });
    } else {
        return try std.fmt.allocPrint(allocator, "{s} @ {s} - {s} ago", .{
            wt_info.display_name,
            wt_info.worktree.branch,
            duration_str,
        });
    }
}

/// Read number selection from user
pub fn readNumberSelection(
    allocator: std.mem.Allocator,
    prompt: []const u8,
    max_value: usize,
) !?usize {
    const input_util = @import("input.zig");
    const response = try input_util.readLine(allocator, prompt);
    if (response) |resp| {
        defer allocator.free(resp);
        const trimmed = std.mem.trim(u8, resp, " \t\r\n");
        
        if (trimmed.len > 0 and (trimmed[0] == 'q' or trimmed[0] == 'Q')) {
            return null;
        }
        
        const selection = std.fmt.parseInt(usize, trimmed, 10) catch {
            return error.InvalidSelection;
        };
        
        if (selection < 1 or selection > max_value) {
            return error.InvalidSelection;
        }
        
        return selection - 1;
    }
    return null;
}