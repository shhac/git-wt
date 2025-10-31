const std = @import("std");

test {
    _ = @import("new_test.zig");
    _ = @import("remove_test.zig");
    _ = @import("go_test.zig");
    _ = @import("list_test.zig");
    _ = @import("alias_test.zig");
    _ = @import("clean_test.zig");
}