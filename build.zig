const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const exe = b.addExecutable(.{
        .name = "git-wt",
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });
    
    // Link libc for system calls (getpid, etc.)
    exe.linkLibC();
    
    // Add version as build option
    const version_option = b.option([]const u8, "version", "Version string") orelse "0.1.1";
    const build_options = b.addOptions();
    build_options.addOption([]const u8, "version", version_option);
    exe.root_module.addOptions("build_options", build_options);

    // Add dependencies
    // Note: We're not using clap anymore but keeping dependency for future use

    b.installArtifact(exe);

    const run_cmd = b.addRunArtifact(exe);
    run_cmd.step.dependOn(b.getInstallStep());

    if (b.args) |args| {
        run_cmd.addArgs(args);
    }

    const run_step = b.step("run", "Run the app");
    run_step.dependOn(&run_cmd.step);

    const exe_unit_tests = b.addTest(.{
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });
    
    // Add build options to unit tests too
    exe_unit_tests.root_module.addOptions("build_options", build_options);

    const run_exe_unit_tests = b.addRunArtifact(exe_unit_tests);

    // Integration tests
    const integration_tests = b.addTest(.{
        .root_source_file = b.path("src/integration_tests.zig"),
        .target = target,
        .optimize = optimize,
    });
    
    integration_tests.root_module.addOptions("build_options", build_options);
    
    const run_integration_tests = b.addRunArtifact(integration_tests);

    // Test steps
    const test_step = b.step("test", "Run unit tests");
    test_step.dependOn(&run_exe_unit_tests.step);
    
    const integration_test_step = b.step("test-integration", "Run integration tests");
    integration_test_step.dependOn(&run_integration_tests.step);
    
    const test_all_step = b.step("test-all", "Run all tests (unit + integration)");
    test_all_step.dependOn(&run_exe_unit_tests.step);
    test_all_step.dependOn(&run_integration_tests.step);
}