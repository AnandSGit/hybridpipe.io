// TCP protocol implementation for HybridPipe
// Provides TCP/IP communication

const std = @import("std");
const net = std.net;
const Error = @import("../../core/errors.zig").Error;
const HybridPipe = @import("../../core/core.zig").HybridPipe;
const Process = @import("../../core/core.zig").Process;
const BrokerType = @import("../protocol_types.zig").BrokerType;
const TcpConfig = @import("../../core/config.zig").TcpConfig;
const registry = @import("../../core/registry.zig");
const serialization = @import("../../serialization/serialization.zig");

/// TcpRouter implements the HybridPipe interface for TCP/IP communication
pub const TcpRouter = struct {
    pipe: HybridPipe,
    config: TcpConfig,
    allocator: std.mem.Allocator,
    connected: bool = false,
    server: ?net.Server = null,
    client: ?net.Stream = null,
    subscriptions: std.StringHashMap(Process),

    pub fn init(allocator: std.mem.Allocator) !*TcpRouter {
        const self = try allocator.create(TcpRouter);
        self.* = TcpRouter{
            .pipe = HybridPipe{
                .vtable = &vtable,
            },
            .config = TcpConfig.init(),
            .allocator = allocator,
            .subscriptions = std.StringHashMap(Process).init(allocator),
        };
        return self;
    }

    pub fn deinit(self: *TcpRouter) void {
        self.subscriptions.deinit();
        if (self.server) |*server| {
            server.deinit();
        }
        if (self.client) |*client| {
            client.close();
        }
        self.allocator.destroy(self);
    }

    fn connectFn(self_pipe: *HybridPipe) Error!void {
        // In Zig 0.12.0, we need a different approach to get the parent struct
        // We'll use a pointer cast and offset calculation
        const self = @as(*TcpRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(TcpRouter, "pipe")));

        if (self.connected) {
            return Error.AlreadyConnected;
        }

        // For simplicity, we'll just create a client connection
        const address = try net.Address.parseIp(self.config.server, self.config.port);
        self.client = try net.tcpConnectToAddress(address);
        self.connected = true;
    }

    fn disconnectFn(self_pipe: *HybridPipe) Error!void {
        const self = @as(*TcpRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(TcpRouter, "pipe")));

        if (!self.connected) {
            return Error.NotConnected;
        }

        if (self.server) |*server| {
            server.deinit();
            self.server = null;
        }

        if (self.client) |*client| {
            client.close();
            self.client = null;
        }

        self.connected = false;
    }

    fn dispatchFn(self_pipe: *HybridPipe, pipe: []const u8, data: []const u8) Error!void {
        const self = @as(*TcpRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(TcpRouter, "pipe")));

        if (!self.connected) {
            return Error.NotConnected;
        }

        if (self.client) |*client| {
            // Create a message with pipe name and data
            var message = try self.allocator.alloc(u8, pipe.len + 1 + data.len);
            defer self.allocator.free(message);

            std.mem.copyForwards(u8, message[0..pipe.len], pipe);
            message[pipe.len] = ':';
            std.mem.copyForwards(u8, message[pipe.len + 1 ..], data);

            // Encode the message
            const encoded = try serialization.encode(message);
            defer std.heap.page_allocator.free(encoded);

            // Send the message
            _ = try client.write(encoded);
        } else {
            return Error.NotConnected;
        }
    }

    fn subscribeFn(self_pipe: *HybridPipe, pipe: []const u8, callback: Process) Error!void {
        const self = @as(*TcpRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(TcpRouter, "pipe")));

        if (!self.connected) {
            return Error.NotConnected;
        }

        // Store the subscription
        const pipe_copy = try self.allocator.dupe(u8, pipe);
        try self.subscriptions.put(pipe_copy, callback);

        // In a real implementation, we would start a listener thread here
        // For simplicity, we'll just simulate it
    }

    fn unsubscribeFn(self_pipe: *HybridPipe, pipe: []const u8) Error!void {
        const self = @as(*TcpRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(TcpRouter, "pipe")));

        if (!self.connected) {
            return Error.NotConnected;
        }

        // Remove the subscription
        _ = self.subscriptions.remove(pipe);
    }

    fn isConnectedFn(self_pipe: *HybridPipe) bool {
        const self = @as(*TcpRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(TcpRouter, "pipe")));
        return self.connected;
    }

    const vtable = HybridPipe.VTable{
        .connect = connectFn,
        .disconnect = disconnectFn,
        .dispatch = dispatchFn,
        .subscribe = subscribeFn,
        .unsubscribe = unsubscribeFn,
        .isConnected = isConnectedFn,
    };
};

/// Factory function for creating a TCP router
pub fn createTcpRouter(allocator: std.mem.Allocator) Error!*HybridPipe {
    const router = try TcpRouter.init(allocator);
    return &router.pipe;
}

/// Register the TCP protocol with the HybridPipe system
pub fn register() void {
    registry.registerRouterFactory(BrokerType.TCP, createTcpRouter) catch |err| {
        std.debug.print("Failed to register TCP protocol: {}\n", .{err});
    };
}

// Test the TCP router
test "TcpRouter" {
    const allocator = std.testing.allocator;

    var router = try TcpRouter.init(allocator);
    defer router.deinit();

    // This is just a compile-time test to ensure the router is well-defined
    try std.testing.expect(!router.pipe.isConnected());

    // We can't actually test the connection in a unit test
    // as it would require a real TCP server
}
