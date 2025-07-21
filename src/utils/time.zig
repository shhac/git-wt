const std = @import("std");

/// Calculate seconds elapsed since a given modification time
/// Takes mod_time in nanoseconds and returns seconds ago
pub fn secondsSince(mod_time: i128) u64 {
    const timestamp = @divFloor(mod_time, std.time.ns_per_s);
    return @as(u64, @intCast(std.time.timestamp() - timestamp));
}

/// Format a duration in seconds to a human-readable string
/// Returns strings like "5s", "2m 30s", "1h 5m", "1d 1h", "1y 1mo"
/// Special cases: "just now" for 0-2 seconds, "ancient" for > 10 years
pub fn formatDuration(allocator: std.mem.Allocator, seconds: u64) ![]u8 {
    // Special case for very recent times
    if (seconds <= 2) {
        return try allocator.dupe(u8, "just now");
    }
    
    const minute = 60;
    const hour = minute * 60;
    const day = hour * 24;
    const week = day * 7;
    const month = day * 30;
    const year = day * 365;
    const decade = year * 10;
    
    // Special case for very old times
    if (seconds > decade) {
        const decades = seconds / decade;
        if (decades == 1) {
            return try allocator.dupe(u8, "over a decade");
        } else {
            return try std.fmt.allocPrint(allocator, "over {d} decades", .{decades});
        }
    }
    
    // Calculate each unit
    const years = seconds / year;
    const months = (seconds % year) / month;
    const weeks = (seconds % year % month) / week;
    const days = (seconds % year % month % week) / day;
    const hours = (seconds % day) / hour;
    const minutes = (seconds % hour) / minute;
    const secs = seconds % minute;
    
    // Build array of non-zero units
    var units = std.ArrayList(struct { value: u64, unit: []const u8 }).init(allocator);
    defer units.deinit();
    
    if (years > 0) try units.append(.{ .value = years, .unit = "y" });
    if (months > 0) try units.append(.{ .value = months, .unit = "mo" });
    if (weeks > 0) try units.append(.{ .value = weeks, .unit = "w" });
    if (days > 0) try units.append(.{ .value = days, .unit = "d" });
    if (hours > 0) try units.append(.{ .value = hours, .unit = "h" });
    if (minutes > 0) try units.append(.{ .value = minutes, .unit = "m" });
    if (secs > 0) try units.append(.{ .value = secs, .unit = "s" });
    
    // This shouldn't happen due to special cases above, but just in case
    if (units.items.len == 0) {
        return try allocator.dupe(u8, "just now");
    }
    
    // Format the two most significant units
    if (units.items.len == 1) {
        return try std.fmt.allocPrint(allocator, "{d}{s}", .{ units.items[0].value, units.items[0].unit });
    } else {
        return try std.fmt.allocPrint(allocator, "{d}{s} {d}{s}", .{
            units.items[0].value,
            units.items[0].unit,
            units.items[1].value,
            units.items[1].unit,
        });
    }
}

test "formatDuration" {
    const allocator = std.testing.allocator;
    
    // Test seconds
    const s = try formatDuration(allocator, 45);
    defer allocator.free(s);
    try std.testing.expectEqualStrings("45s", s);
    
    // Test minutes
    const m = try formatDuration(allocator, 150);
    defer allocator.free(m);
    try std.testing.expectEqualStrings("2m 30s", m);
    
    // Test hours
    const h = try formatDuration(allocator, 3900);
    defer allocator.free(h);
    try std.testing.expectEqualStrings("1h 5m", h);
    
    // Test days
    const d = try formatDuration(allocator, 90000);
    defer allocator.free(d);
    try std.testing.expectEqualStrings("1d 1h", d);
    
    // Test years
    const y = try formatDuration(allocator, 31536000 + 2592000);
    defer allocator.free(y);
    try std.testing.expectEqualStrings("1y 1mo", y);
    
    // Test zero
    const zero = try formatDuration(allocator, 0);
    defer allocator.free(zero);
    try std.testing.expectEqualStrings("just now", zero);
    
    // Test just now (1-2 seconds)
    const just_now = try formatDuration(allocator, 2);
    defer allocator.free(just_now);
    try std.testing.expectEqualStrings("just now", just_now);
    
    // Test over a decade
    const decade = try formatDuration(allocator, 365 * 24 * 60 * 60 * 11); // 11 years
    defer allocator.free(decade);
    try std.testing.expectEqualStrings("over a decade", decade);
    
    // Test multiple decades
    const decades = try formatDuration(allocator, 365 * 24 * 60 * 60 * 25); // 25 years
    defer allocator.free(decades);
    try std.testing.expectEqualStrings("over 2 decades", decades);
}