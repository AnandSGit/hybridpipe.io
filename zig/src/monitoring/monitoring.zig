// Monitoring system for HybridPipe
// Provides monitoring and metrics collection

const std = @import("std");
const Error = @import("../core/errors.zig").Error;
const HybridPipe = @import("../core/core.zig").HybridPipe;
const Process = @import("../core/core.zig").Process;

/// MessageStats tracks message statistics for a pipe
pub const MessageStats = struct {
    /// Number of messages sent
    messages_sent: u64 = 0,
    /// Number of messages received
    messages_received: u64 = 0,
    /// Total bytes sent
    bytes_sent: u64 = 0,
    /// Total bytes received
    bytes_received: u64 = 0,
    /// Last message sent timestamp
    last_sent_timestamp: i64 = 0,
    /// Last message received timestamp
    last_received_timestamp: i64 = 0,
    /// Average message size sent
    average_message_size_sent: f64 = 0,
    /// Average message size received
    average_message_size_received: f64 = 0,

    pub fn init() MessageStats {
        return MessageStats{};
    }

    pub fn recordMessageSent(self: *MessageStats, size: usize) void {
        self.messages_sent += 1;
        self.bytes_sent += size;
        self.last_sent_timestamp = std.time.milliTimestamp();
        self.average_message_size_sent = @as(f64, @floatFromInt(self.bytes_sent)) / @as(f64, @floatFromInt(self.messages_sent));
    }

    pub fn recordMessageReceived(self: *MessageStats, size: usize) void {
        self.messages_received += 1;
        self.bytes_received += size;
        self.last_received_timestamp = std.time.milliTimestamp();
        self.average_message_size_received = @as(f64, @floatFromInt(self.bytes_received)) / @as(f64, @floatFromInt(self.messages_received));
    }

    pub fn reset(self: *MessageStats) void {
        self.messages_sent = 0;
        self.messages_received = 0;
        self.bytes_sent = 0;
        self.bytes_received = 0;
        self.last_sent_timestamp = 0;
        self.last_received_timestamp = 0;
        self.average_message_size_sent = 0;
        self.average_message_size_received = 0;
    }
};

/// MonitoringMiddleware collects metrics for HybridPipe operations
pub const MonitoringMiddleware = struct {
    stats: std.StringHashMap(MessageStats),
    allocator: std.mem.Allocator,

    pub fn init(allocator: std.mem.Allocator) MonitoringMiddleware {
        return MonitoringMiddleware{
            .stats = std.StringHashMap(MessageStats).init(allocator),
            .allocator = allocator,
        };
    }

    pub fn deinit(self: *MonitoringMiddleware) void {
        self.stats.deinit();
    }

    pub fn getStats(self: *MonitoringMiddleware, pipe: []const u8) !MessageStats {
        return self.stats.get(pipe) orelse MessageStats.init();
    }

    pub fn getAllStats(self: *MonitoringMiddleware) std.StringHashMap(MessageStats) {
        return self.stats;
    }

    pub fn resetStats(self: *MonitoringMiddleware) void {
        var it = self.stats.iterator();
        while (it.next()) |entry| {
            var stats = entry.value_ptr;
            stats.reset();
        }
    }

    fn getOrCreateStats(self: *MonitoringMiddleware, pipe: []const u8) !*MessageStats {
        const pipe_copy = try self.allocator.dupe(u8, pipe);

        if (!self.stats.contains(pipe)) {
            try self.stats.put(pipe_copy, MessageStats.init());
        }

        return self.stats.getPtr(pipe).?;
    }

    pub fn beforeConnect(self: *MonitoringMiddleware, router: *HybridPipe) Error!void {
        _ = self;
        _ = router;
    }

    pub fn afterConnect(self: *MonitoringMiddleware, router: *HybridPipe) Error!void {
        _ = self;
        _ = router;
    }

    pub fn beforeDisconnect(self: *MonitoringMiddleware, router: *HybridPipe) Error!void {
        _ = self;
        _ = router;
    }

    pub fn afterDisconnect(self: *MonitoringMiddleware, router: *HybridPipe) Error!void {
        _ = self;
        _ = router;
    }

    pub fn beforeDispatch(self: *MonitoringMiddleware, router: *HybridPipe, pipe: []const u8, data: []const u8) Error!void {
        _ = router;
        var stats = try self.getOrCreateStats(pipe);
        stats.recordMessageSent(data.len);
    }

    pub fn afterDispatch(self: *MonitoringMiddleware, router: *HybridPipe, pipe: []const u8, data: []const u8) Error!void {
        _ = self;
        _ = router;
        _ = pipe;
        _ = data;
    }

    pub fn beforeSubscribe(self: *MonitoringMiddleware, router: *HybridPipe, pipe: []const u8, callback: Process) Error!void {
        _ = self;
        _ = router;
        _ = pipe;
        _ = callback;
    }

    pub fn afterSubscribe(self: *MonitoringMiddleware, router: *HybridPipe, pipe: []const u8, callback: Process) Error!void {
        _ = self;
        _ = router;
        _ = pipe;
        _ = callback;
    }

    pub fn beforeUnsubscribe(self: *MonitoringMiddleware, router: *HybridPipe, pipe: []const u8) Error!void {
        _ = self;
        _ = router;
        _ = pipe;
    }

    pub fn afterUnsubscribe(self: *MonitoringMiddleware, router: *HybridPipe, pipe: []const u8) Error!void {
        _ = self;
        _ = router;
        _ = pipe;
    }

    pub fn beforeProcess(self: *MonitoringMiddleware, pipe: []const u8, data: []const u8) Error!void {
        var stats = try self.getOrCreateStats(pipe);
        stats.recordMessageReceived(data.len);
    }

    pub fn afterProcess(self: *MonitoringMiddleware, pipe: []const u8, data: []const u8) Error!void {
        _ = self;
        _ = pipe;
        _ = data;
    }
};

// Test the monitoring system
test "MessageStats" {
    var stats = MessageStats.init();

    stats.recordMessageSent(100);
    try std.testing.expectEqual(@as(u64, 1), stats.messages_sent);
    try std.testing.expectEqual(@as(u64, 100), stats.bytes_sent);
    try std.testing.expectEqual(@as(f64, 100.0), stats.average_message_size_sent);

    stats.recordMessageReceived(200);
    try std.testing.expectEqual(@as(u64, 1), stats.messages_received);
    try std.testing.expectEqual(@as(u64, 200), stats.bytes_received);
    try std.testing.expectEqual(@as(f64, 200.0), stats.average_message_size_received);

    stats.reset();
    try std.testing.expectEqual(@as(u64, 0), stats.messages_sent);
    try std.testing.expectEqual(@as(u64, 0), stats.messages_received);
}

test "MonitoringMiddleware" {
    const allocator = std.testing.allocator;
    var monitoring = MonitoringMiddleware.init(allocator);
    defer monitoring.deinit();

    try monitoring.beforeDispatch(undefined, "test", "data");

    var stats = try monitoring.getStats("test");
    try std.testing.expectEqual(@as(u64, 1), stats.messages_sent);
    try std.testing.expectEqual(@as(u64, 4), stats.bytes_sent); // "data" is 4 bytes

    monitoring.resetStats();
    stats = try monitoring.getStats("test");
    try std.testing.expectEqual(@as(u64, 0), stats.messages_sent);
}
