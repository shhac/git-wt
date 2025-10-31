
pub const LockError = error{
    LockAcquisitionFailed,
    LockTimeout,
    InvalidLockFile,
};

pub const Lock = struct {
    path: []const u8,
    allocator: std.mem.Allocator,
    file: ?fs.File = null,
    
    /// Create a new lock instance
    pub fn init(allocator: std.mem.Allocator, lock_path: []const u8) Lock {
        return .{
            .allocator = allocator,
            .path = lock_path,
        };
    }
    
    /// Acquire the lock (blocking with timeout)
    pub fn acquire(self: *Lock, timeout_ms: u64) !void {
        const start_time = std.time.milliTimestamp();
        
        while (true) {
            // Try to create lock file exclusively
            self.file = fs.cwd().createFile(self.path, .{ .exclusive = true }) catch |err| {
                switch (err) {
                    error.PathAlreadyExists => {
                        // Check if we've timed out
                        const elapsed = @as(u64, @intCast(std.time.milliTimestamp() - start_time));
                        if (elapsed > timeout_ms) {
                            return LockError.LockTimeout;
                        }
                        
                        // Sleep a bit and retry
                        std.Thread.sleep(100 * std.time.ns_per_ms);
                        continue;
                    },
                    else => return err,
                }
            };
            
            // Write PID and timestamp to lock file
            if (self.file) |file| {
                const pid = std.c.getpid();
                const timestamp = std.time.timestamp();
                var buffer: [256]u8 = undefined;
                const content = try std.fmt.bufPrint(&buffer, "{d}\n{d}\n", .{ pid, timestamp });
                try file.writeAll(content);
                break;
            } else {
                return LockError.LockAcquisitionFailed;
            }
        }
    }
    
    /// Try to acquire the lock without blocking
    pub fn tryAcquire(self: *Lock) !bool {
        self.file = fs.cwd().createFile(self.path, .{ .exclusive = true }) catch |err| {
            switch (err) {
                error.PathAlreadyExists => return false,
                else => return err,
            }
        };
        
        // Write PID and timestamp to lock file
        if (self.file) |file| {
            const writer = file.writer();
            const pid = std.c.getpid();
            try writer.print("{d}\n{d}\n", .{ pid, std.time.timestamp() });
            return true;
        } else {
            return LockError.LockAcquisitionFailed;
        }
    }
    
    /// Release the lock
    pub fn release(self: *Lock) void {
        if (self.file) |file| {
            file.close();
            self.file = null;
            
            // Delete the lock file
            fs.cwd().deleteFile(self.path) catch |err| {
                std.log.warn("Failed to delete lock file {s}: {}", .{ self.path, err });
            };
        }
    }
    
    /// Check if lock is stale (process died without releasing)
    pub fn isStale(self: *Lock) !bool {
        const file = fs.cwd().openFile(self.path, .{}) catch |err| {
            switch (err) {
                error.FileNotFound => return true, // No lock file means it's "stale"
                else => return err,
            }
        };
        defer file.close();
        
        const content = try file.readToEndAlloc(self.allocator, 1024);
        defer self.allocator.free(content);
        
        // Parse PID from first line
        var lines = std.mem.tokenizeScalar(u8, content, '\n');
        const pid_str = lines.next() orelse return LockError.InvalidLockFile;
        const pid = std.fmt.parseInt(u32, pid_str, 10) catch return LockError.InvalidLockFile;
        
        // Check if process is still running
        // On Unix systems and WSL2, we can use kill(pid, 0) to check if process exists
        if (@hasDecl(std.posix, "kill")) {
            // kill returns error if process doesn't exist
            std.posix.kill(@intCast(pid), 0) catch {
                return true; // Process doesn't exist, lock is stale
            };
            return false; // Process exists, lock is not stale
        }
        
        // On native Windows (non-WSL2), we'd need OpenProcess, but since we only support WSL2:
        // If we reach here on Windows, assume WSL2 environment is expected
        // and the kill check should have worked. Consider the lock potentially stale.
        const builtin = @import("builtin");
        if (builtin.target.os.tag == .windows) {
            std.log.warn("Native Windows detected - WSL2 expected. Assuming stale lock for safety.\n", .{});
            return true; // Assume stale for safety
        }
        
        // For other unsupported platforms, assume not stale
        return false;
    }
    
    /// Clean up stale lock if it exists (atomic version)
    pub fn cleanStale(self: *Lock) !void {
        _ = try self.tryCleanStaleAtomic();
    }
    
    /// Try to clean up stale lock atomically
    /// Returns true if we successfully removed a stale lock
    fn tryCleanStaleAtomic(self: *Lock) !bool {
        // Check if process is still running
        if (!(try self.isStale())) {
            return false;
        }
        
        // Lock appears stale, try to remove it atomically
        // by attempting to rename it first (which is atomic)
        const stale_path = try std.fmt.allocPrint(self.allocator, "{s}.stale.{d}", .{ self.path, std.time.nanoTimestamp() });
        defer self.allocator.free(stale_path);
        
        fs.cwd().rename(self.path, stale_path) catch |err| {
            switch (err) {
                error.FileNotFound => return false, // Someone else removed it
                else => return err,
            }
        };
        
        // We successfully renamed it, now we can safely delete it
        fs.cwd().deleteFile(stale_path) catch |err| {
            // Log error but don't fail - the lock is effectively removed
            std.debug.print("Warning: Failed to delete stale lock file: {}\n", .{err});
        };
        
        return true;
    }

    /// Acquire lock with user-friendly error feedback
    /// This consolidates the common pattern of acquiring a lock and printing
    /// helpful error messages when acquisition fails due to timeout.
    pub fn acquireWithUserFeedback(self: *Lock, timeout_ms: u64, stderr: anytype) !void {
        self.acquire(timeout_ms) catch |err| {
            if (err == LockError.LockTimeout) {
                try colors.printError(stderr, "Another git-wt operation is in progress", .{});
                try stderr.print("{s}Tip:{s} Wait for the other operation to complete or check for stale locks\n", .{
                    colors.info_prefix, colors.reset
                });
            }
            return err;
        };
    }

    /// Deinit (ensures lock is released)
    pub fn deinit(self: *Lock) void {
        self.release();
    }
};

/// Helper to run a function with a lock
const std = @import("std");
const fs = std.fs;
const colors = @import("colors.zig");
const io = @import("io.zig");
pub fn withLock(allocator: std.mem.Allocator, lock_path: []const u8, timeout_ms: u64, comptime func: anytype, args: anytype) !@typeInfo(@TypeOf(func)).Fn.return_type.? {
    var lock = Lock.init(allocator, lock_path);
    defer lock.deinit(allocator);
    
    // Clean up any stale locks first
    try lock.cleanStale();
    
    // Acquire the lock
    try lock.acquire(timeout_ms);
    
    // Run the function
    return @call(.auto, func, args);
}

test "Lock basic operations" {
    const allocator = std.testing.allocator;
    
    // Use a temporary directory for testing
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();
    
    const tmp_path = try tmp_dir.dir.realpathAlloc(allocator, ".");
    defer allocator.free(tmp_path);
    
    const lock_path = try fs.path.join(allocator, &.{ tmp_path, "test.lock" });
    defer allocator.free(lock_path);
    
    var lock = Lock.init(allocator, lock_path);
    defer lock.deinit(allocator);
    
    // Test tryAcquire
    try std.testing.expect(try lock.tryAcquire());
    
    // Second tryAcquire should fail
    var lock2 = Lock.init(allocator, lock_path);
    defer lock2.deinit(allocator);
    try std.testing.expect(!try lock2.tryAcquire());
    
    // Release first lock
    lock.release();
    
    // Now second lock should succeed
    try std.testing.expect(try lock2.tryAcquire());
}

test "Lock timeout" {
    const allocator = std.testing.allocator;
    
    // Use a temporary directory for testing
    var tmp_dir = std.testing.tmpDir(.{});
    defer tmp_dir.cleanup();
    
    const tmp_path = try tmp_dir.dir.realpathAlloc(allocator, ".");
    defer allocator.free(tmp_path);
    
    const lock_path = try fs.path.join(allocator, &.{ tmp_path, "test_timeout.lock" });
    defer allocator.free(lock_path);
    
    var lock1 = Lock.init(allocator, lock_path);
    defer lock1.deinit(allocator);
    
    // Acquire first lock
    try lock1.acquire(1000);
    
    // Second lock should timeout
    var lock2 = Lock.init(allocator, lock_path);
    defer lock2.deinit(allocator);
    
    const result = lock2.acquire(200);
    try std.testing.expectError(LockError.LockTimeout, result);
}
