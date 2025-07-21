const std = @import("std");
const posix = std.posix;
const colors = @import("colors.zig");

/// Global state for signal handling
var g_raw_mode: ?*RawMode = null;
var g_signal_mutex = std.Thread.Mutex{};

/// Signal handler for SIGINT
fn handleSignal(_: i32) callconv(.C) void {
    // Restore terminal state
    g_signal_mutex.lock();
    defer g_signal_mutex.unlock();
    
    if (g_raw_mode) |raw_mode| {
        raw_mode.exit();
        showCursor() catch {};
    }
    
    // Exit the program
    std.process.exit(130); // 128 + SIGINT(2)
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
    is_raw: bool = false,
    
    /// Enter raw mode for single character input
    pub fn enter(self: *RawMode) !void {
        if (self.is_raw) return;
        
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
        self.is_raw = true;
        
        // Register for signal handling
        g_signal_mutex.lock();
        defer g_signal_mutex.unlock();
        g_raw_mode = self;
    }
    
    /// Exit raw mode and restore original settings
    pub fn exit(self: *RawMode) void {
        if (!self.is_raw) return;
        
        // Unregister from signal handling
        g_signal_mutex.lock();
        defer g_signal_mutex.unlock();
        if (g_raw_mode == self) {
            g_raw_mode = null;
        }
        
        posix.tcsetattr(std.io.getStdIn().handle, .FLUSH, self.original_termios) catch {};
        self.is_raw = false;
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
};

/// Render a selection list item with enhanced formatting
fn renderItem(
    writer: anytype,
    item_text: []const u8, 
    is_selected: bool,
    use_colors: bool,
) !void {
    if (use_colors) {
        if (is_selected) {
            // Selected item: green brackets with bright green asterisk and bold text
            try writer.print("  {s}[{s}*{s}]{s} {s}{s}{s}\n", .{
                colors.green,          // green [
                "\x1b[1m\x1b[92m",    // bright green + bold for *
                colors.green,          // back to green for ]
                colors.reset,          // reset before text
                "\x1b[1m",            // bold for text
                item_text,
                colors.reset,          // final reset
            });
        } else {
            // Unselected item: dim [ ] with normal text
            try writer.print("  {s}[ ]{s} {s}\n", .{
                "\x1b[2m", // dim
                colors.reset,
                item_text,
            });
        }
    } else {
        try writer.print("  [{s}] {s}\n", .{
            if (is_selected) "*" else " ",
            item_text,
        });
    }
}

/// Display an interactive selection list
/// Returns the selected index or null if cancelled
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
        try renderItem(stdout, item, i == selected, options.use_colors);
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
                // Clear the selection display
                const total_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
                try moveCursorUp(total_lines);
                for (0..total_lines) |_| {
                    try clearLine();
                    try stdout.print("\n", .{});
                }
                try moveCursorUp(total_lines);
                
                return selected;
            },
            .escape => {
                // Clear the selection display  
                const total_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
                try moveCursorUp(total_lines);
                for (0..total_lines) |_| {
                    try clearLine();
                    try stdout.print("\n", .{});
                }
                try moveCursorUp(total_lines);
                
                return null;
            },
            .char => {
                // Handle 'q' for quit
                if (key_info.char == 'q' or key_info.char == 'Q') {
                    // Clear the selection display
                    const total_lines: usize = items.len + (if (options.show_instructions) @as(usize, 2) else @as(usize, 0));
                    try moveCursorUp(total_lines);
                    for (0..total_lines) |_| {
                        try clearLine();
                        try stdout.print("\n", .{});
                    }
                    try moveCursorUp(total_lines);
                    
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
                try renderItem(stdout, item, i == selected, options.use_colors);
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