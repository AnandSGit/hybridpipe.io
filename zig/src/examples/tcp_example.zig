// TCP example for HybridPipe
// Demonstrates using the TCP protocol

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
    
    std.debug.print("TCP Example\n", .{});
    std.debug.print("This example demonstrates using the TCP protocol with HybridPipe.\n", .{});
    std.debug.print("Note: This example requires a TCP server running on localhost:8080.\n", .{});
    std.debug.print("You can use netcat to create a simple server: nc -l 8080\n\n", .{});
    
    // Deploy a TCP router
    var router = hybridpipe.deployRouter(hybridpipe.TCP) catch |err| {
        std.debug.print("Failed to deploy TCP router: {}\n", .{err});
        return;
    };
    
    std.debug.print("Connected to TCP server\n", .{});
    
    // Subscribe to a pipe
    router.subscribe("example", messageHandler) catch |err| {
        std.debug.print("Failed to subscribe: {}\n", .{err});
        return;
    };
    
    std.debug.print("Subscribed to 'example' pipe\n", .{});
    
    // Send a message
    const message = "Hello from Zig HybridPipe!";
    router.dispatch("example", message) catch |err| {
        std.debug.print("Failed to dispatch message: {}\n", .{err});
        return;
    };
    
    std.debug.print("Sent message: {s}\n", .{message});
    
    // Wait a bit to allow the message to be processed
    std.time.sleep(1 * std.time.ns_per_s);
    
    // Unsubscribe from the pipe
    router.unsubscribe("example") catch |err| {
        std.debug.print("Failed to unsubscribe: {}\n", .{err});
        return;
    };
    
    std.debug.print("Unsubscribed from 'example' pipe\n", .{});
    
    // Disconnect from the server
    router.disconnect() catch |err| {
        std.debug.print("Failed to disconnect: {}\n", .{err});
        return;
    };
    
    std.debug.print("Disconnected from TCP server\n", .{});
}

fn messageHandler(data: []const u8) hybridpipe.Error!void {
    std.debug.print("Received message: {s}\n", .{data});
}
