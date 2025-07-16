pub const packages = struct {
    pub const @"ansi_term-0.1.0-_baAywpoAABEqsPmS5Jz_CddDCrG8qdIyRIESH8D2fzd" = struct {
        pub const build_root = "/Users/paul/.cache/zig/p/ansi_term-0.1.0-_baAywpoAABEqsPmS5Jz_CddDCrG8qdIyRIESH8D2fzd";
        pub const build_zig = @import("ansi_term-0.1.0-_baAywpoAABEqsPmS5Jz_CddDCrG8qdIyRIESH8D2fzd");
        pub const deps: []const struct { []const u8, []const u8 } = &.{
        };
    };
    pub const @"clap-0.10.0-oBajB7jkAQAZ4cKLlzkeV9mDu2yGZvtN2QuOyfAfjBij" = struct {
        pub const build_root = "/Users/paul/.cache/zig/p/clap-0.10.0-oBajB7jkAQAZ4cKLlzkeV9mDu2yGZvtN2QuOyfAfjBij";
        pub const build_zig = @import("clap-0.10.0-oBajB7jkAQAZ4cKLlzkeV9mDu2yGZvtN2QuOyfAfjBij");
        pub const deps: []const struct { []const u8, []const u8 } = &.{
        };
    };
};

pub const root_deps: []const struct { []const u8, []const u8 } = &.{
    .{ "clap", "clap-0.10.0-oBajB7jkAQAZ4cKLlzkeV9mDu2yGZvtN2QuOyfAfjBij" },
    .{ "ansi_term", "ansi_term-0.1.0-_baAywpoAABEqsPmS5Jz_CddDCrG8qdIyRIESH8D2fzd" },
};
