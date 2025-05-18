// Mock example for HybridPipe
// Demonstrates using the Mock protocol for testing

const std = @import("std");
const hybridpipe = @import("hybridpipe");

pub fn main() !void {
    // Initialize the allocator
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    // Initialize the registry
    hybridpipe.registry.initRegistry(allocator);

    // Initialize the configuration
    hybridpipe.config.initConfig(allocator);

    // Initialize the library
    hybridpipe.init();

    std.debug.print("Mock Example\n", .{});
    std.debug.print("This example demonstrates using the Mock protocol with HybridPipe.\n\n", .{});

    // Deploy a Mock router
    var router = hybridpipe.deployRouter(hybridpipe.MOCK) catch |err| {
        std.debug.print("Failed to deploy Mock router: {}\n", .{err});
        return;
    };

    std.debug.print("Connected to Mock system\n", .{});

    // Create a middleware for logging
    const logging_middleware = hybridpipe.middleware.LoggingMiddleware.init(.Info);

    // Subscribe to a pipe
    try logging_middleware.beforeSubscribe(router, "example", messageHandler);
    router.subscribe("example", messageHandler) catch |err| {
        std.debug.print("Failed to subscribe: {}\n", .{err});
        return;
    };
    try logging_middleware.afterSubscribe(router, "example", messageHandler);

    std.debug.print("Subscribed to 'example' pipe\n", .{});

    // Send a message
    const message = "Hello from Zig HybridPipe!";
    try logging_middleware.beforeDispatch(router, "example", message);
    router.dispatch("example", message) catch |err| {
        std.debug.print("Failed to dispatch message: {}\n", .{err});
        return;
    };
    try logging_middleware.afterDispatch(router, "example", message);

    // Send another message
    const message2 = "This is a test message";
    try logging_middleware.beforeDispatch(router, "example", message2);
    router.dispatch("example", message2) catch |err| {
        std.debug.print("Failed to dispatch message: {}\n", .{err});
        return;
    };
    try logging_middleware.afterDispatch(router, "example", message2);

    // Unsubscribe from the pipe
    try logging_middleware.beforeUnsubscribe(router, "example");
    router.unsubscribe("example") catch |err| {
        std.debug.print("Failed to unsubscribe: {}\n", .{err});
        return;
    };
    try logging_middleware.afterUnsubscribe(router, "example");

    std.debug.print("Unsubscribed from 'example' pipe\n", .{});

    // Disconnect from the server
    try logging_middleware.beforeDisconnect(router);
    router.disconnect() catch |err| {
        std.debug.print("Failed to disconnect: {}\n", .{err});
        return;
    };
    try logging_middleware.afterDisconnect(router);

    std.debug.print("Disconnected from Mock system\n", .{});
}

fn messageHandler(data: []const u8) hybridpipe.Error!void {
    std.debug.print("Received message: {s}\n", .{data});
}
