const std = @import("std");

// Main build function
pub fn build(b: *std.Build) void {
    // Standard target options
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    // Create the HybridPipe module
    const hybridpipe_module = b.addModule("hybridpipe", .{
        .root_source_file = .{ .path = "src/main.zig" },
    });

    // Create the HybridPipe library
    const lib = b.addStaticLibrary(.{
        .name = "hybridpipe",
        .root_source_file = .{ .path = "src/main.zig" },
        .target = target,
        .optimize = optimize,
    });

    // Install the library
    b.installArtifact(lib);

    // Create tests
    const lib_tests = b.addTest(.{
        .root_source_file = .{ .path = "src/main.zig" },
        .target = target,
        .optimize = optimize,
    });

    const test_step = b.step("test", "Run library tests");
    test_step.dependOn(&lib_tests.step);

    // Create examples
    const tcp_example = b.addExecutable(.{
        .name = "tcp_example",
        .root_source_file = .{ .path = "src/examples/tcp_example.zig" },
        .target = target,
        .optimize = optimize,
    });
    tcp_example.root_module.addImport("hybridpipe", hybridpipe_module);
    b.installArtifact(tcp_example);

    const mock_example = b.addExecutable(.{
        .name = "mock_example",
        .root_source_file = .{ .path = "src/examples/mock_example.zig" },
        .target = target,
        .optimize = optimize,
    });
    mock_example.root_module.addImport("hybridpipe", hybridpipe_module);
    b.installArtifact(mock_example);

    // Run examples
    const run_tcp_example = b.addRunArtifact(tcp_example);
    run_tcp_example.step.dependOn(b.getInstallStep());
    if (b.args) |args| {
        run_tcp_example.addArgs(args);
    }

    const run_tcp_step = b.step("run-tcp", "Run the TCP example");
    run_tcp_step.dependOn(&run_tcp_example.step);

    const run_mock_example = b.addRunArtifact(mock_example);
    run_mock_example.step.dependOn(b.getInstallStep());
    if (b.args) |args| {
        run_mock_example.addArgs(args);
    }

    const run_mock_step = b.step("run-mock", "Run the Mock example");
    run_mock_step.dependOn(&run_mock_example.step);
}
