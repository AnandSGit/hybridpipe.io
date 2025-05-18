// Mock protocol implementation for HybridPipe
// Provides an in-memory mock for testing

const std = @import("std");
const Error = @import("../../core/errors.zig").Error;
const HybridPipe = @import("../../core/core.zig").HybridPipe;
const Process = @import("../../core/core.zig").Process;
const BrokerType = @import("../protocol_types.zig").BrokerType;
const MockConfig = @import("../../core/config.zig").MockConfig;
const registry = @import("../../core/registry.zig");

/// Message represents a message in the mock system
const Message = struct {
    pipe: []const u8,
    data: []const u8,
    timestamp: i64,

    pub fn init(allocator: std.mem.Allocator, pipe: []const u8, data: []const u8) !Message {
        return Message{
            .pipe = try allocator.dupe(u8, pipe),
            .data = try allocator.dupe(u8, data),
            .timestamp = std.time.milliTimestamp(),
        };
    }

    pub fn deinit(self: *Message, allocator: std.mem.Allocator) void {
        allocator.free(self.pipe);
        allocator.free(self.data);
    }
};

/// MockRouter implements the HybridPipe interface for in-memory testing
pub const MockRouter = struct {
    pipe: HybridPipe,
    config: MockConfig,
    allocator: std.mem.Allocator,
    connected: bool = false,
    subscriptions: std.StringHashMap(Process),
    messages: std.ArrayList(Message),

    pub fn init(allocator: std.mem.Allocator) !*MockRouter {
        const self = try allocator.create(MockRouter);
        self.* = MockRouter{
            .pipe = HybridPipe{
                .vtable = &vtable,
            },
            .config = MockConfig.init(),
            .allocator = allocator,
            .subscriptions = std.StringHashMap(Process).init(allocator),
            .messages = std.ArrayList(Message).init(allocator),
        };
        return self;
    }

    pub fn deinit(self: *MockRouter) void {
        // Free all messages
        for (self.messages.items) |*message| {
            message.deinit(self.allocator);
        }
        self.messages.deinit();
        self.subscriptions.deinit();
        self.allocator.destroy(self);
    }

    fn connectFn(self_pipe: *HybridPipe) Error!void {
        const self = @as(*MockRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(MockRouter, "pipe")));

        if (self.connected) {
            return Error.AlreadyConnected;
        }

        // Simulate connection failures if configured
        if (self.config.simulate_failures) {
            var random = std.rand.DefaultPrng.init(@intCast(std.time.milliTimestamp()));
            const random_value = random.random().float(f32);

            if (random_value < self.config.failure_rate) {
                return Error.ConnectionFailed;
            }
        }

        // Simulate latency if configured
        if (self.config.latency_ms > 0) {
            std.time.sleep(self.config.latency_ms * std.time.ns_per_ms);
        }

        self.connected = true;
    }

    fn disconnectFn(self_pipe: *HybridPipe) Error!void {
        const self = @as(*MockRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(MockRouter, "pipe")));

        if (!self.connected) {
            return Error.NotConnected;
        }

        // Simulate latency if configured
        if (self.config.latency_ms > 0) {
            std.time.sleep(self.config.latency_ms * std.time.ns_per_ms);
        }

        self.connected = false;
    }

    fn dispatchFn(self_pipe: *HybridPipe, pipe: []const u8, data: []const u8) Error!void {
        const self = @as(*MockRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(MockRouter, "pipe")));

        if (!self.connected) {
            return Error.NotConnected;
        }

        // Simulate failures if configured
        if (self.config.simulate_failures) {
            var random = std.rand.DefaultPrng.init(@intCast(std.time.milliTimestamp()));
            const random_value = random.random().float(f32);

            if (random_value < self.config.failure_rate) {
                return Error.DispatchFailed;
            }
        }

        // Simulate latency if configured
        if (self.config.latency_ms > 0) {
            std.time.sleep(self.config.latency_ms * std.time.ns_per_ms);
        }

        // Store the message
        const message = try Message.init(self.allocator, pipe, data);
        try self.messages.append(message);

        // Deliver the message to subscribers
        if (self.subscriptions.get(pipe)) |callback| {
            try callback(data);
        }
    }

    fn subscribeFn(self_pipe: *HybridPipe, pipe: []const u8, callback: Process) Error!void {
        const self = @as(*MockRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(MockRouter, "pipe")));

        if (!self.connected) {
            return Error.NotConnected;
        }

        // Simulate failures if configured
        if (self.config.simulate_failures) {
            var random = std.rand.DefaultPrng.init(@intCast(std.time.milliTimestamp()));
            const random_value = random.random().float(f32);

            if (random_value < self.config.failure_rate) {
                return Error.SubscriptionFailed;
            }
        }

        // Simulate latency if configured
        if (self.config.latency_ms > 0) {
            std.time.sleep(self.config.latency_ms * std.time.ns_per_ms);
        }

        // Store the subscription
        const pipe_copy = try self.allocator.dupe(u8, pipe);
        try self.subscriptions.put(pipe_copy, callback);
    }

    fn unsubscribeFn(self_pipe: *HybridPipe, pipe: []const u8) Error!void {
        const self = @as(*MockRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(MockRouter, "pipe")));

        if (!self.connected) {
            return Error.NotConnected;
        }

        // Simulate failures if configured
        if (self.config.simulate_failures) {
            var random = std.rand.DefaultPrng.init(@intCast(std.time.milliTimestamp()));
            const random_value = random.random().float(f32);

            if (random_value < self.config.failure_rate) {
                return Error.UnsubscribeFailed;
            }
        }

        // Simulate latency if configured
        if (self.config.latency_ms > 0) {
            std.time.sleep(self.config.latency_ms * std.time.ns_per_ms);
        }

        // Remove the subscription
        _ = self.subscriptions.remove(pipe);
    }

    fn isConnectedFn(self_pipe: *HybridPipe) bool {
        const self = @as(*MockRouter, @ptrFromInt(@intFromPtr(self_pipe) - @offsetOf(MockRouter, "pipe")));
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

/// Factory function for creating a Mock router
pub fn createMockRouter(allocator: std.mem.Allocator) Error!*HybridPipe {
    const router = try MockRouter.init(allocator);
    return &router.pipe;
}

/// Register the Mock protocol with the HybridPipe system
pub fn register() void {
    registry.registerRouterFactory(BrokerType.MOCK, createMockRouter) catch |err| {
        std.debug.print("Failed to register Mock protocol: {}\n", .{err});
    };
}

// Test the Mock router
test "MockRouter" {
    const allocator = std.testing.allocator;

    var router = try MockRouter.init(allocator);
    defer router.deinit();

    try router.pipe.connect();
    try std.testing.expect(router.pipe.isConnected());

    // Use a static variable to store received data
    const TestData = struct {
        var received: ?[]const u8 = null;

        fn callback(data: []const u8) Error!void {
            received = data;
        }
    };

    try router.pipe.subscribe("test", TestData.callback);
    try router.pipe.dispatch("test", "Hello, Mock!");

    try std.testing.expectEqualStrings("Hello, Mock!", TestData.received.?);

    try router.pipe.unsubscribe("test");
    try router.pipe.disconnect();
    try std.testing.expect(!router.pipe.isConnected());
}
