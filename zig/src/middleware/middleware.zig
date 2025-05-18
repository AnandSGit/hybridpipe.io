// Middleware system for HybridPipe
// Provides middleware components for extending functionality

const std = @import("std");
const Error = @import("../core/errors.zig").Error;
const HybridPipe = @import("../core/core.zig").HybridPipe;
const Process = @import("../core/core.zig").Process;

/// LoggingMiddleware logs all HybridPipe operations
pub const LoggingMiddleware = struct {
    log_level: LogLevel,

    pub const LogLevel = enum {
        Debug,
        Info,
        Warning,
        Error,
    };

    pub fn init(log_level: LogLevel) LoggingMiddleware {
        return LoggingMiddleware{
            .log_level = log_level,
        };
    }

    fn log(self: *const LoggingMiddleware, comptime format: []const u8, args: anytype) void {
        switch (self.log_level) {
            .Debug => std.debug.print("[DEBUG] " ++ format ++ "\n", args),
            .Info => std.debug.print("[INFO] " ++ format ++ "\n", args),
            .Warning => std.debug.print("[WARNING] " ++ format ++ "\n", args),
            .Error => std.debug.print("[ERROR] " ++ format ++ "\n", args),
        }
    }

    pub fn beforeConnect(self: *const LoggingMiddleware, router: *HybridPipe) Error!void {
        self.log("Connecting to router", .{});
        _ = router;
    }

    pub fn afterConnect(self: *const LoggingMiddleware, router: *HybridPipe) Error!void {
        self.log("Connected to router", .{});
        _ = router;
    }

    pub fn beforeDisconnect(self: *const LoggingMiddleware, router: *HybridPipe) Error!void {
        self.log("Disconnecting from router", .{});
        _ = router;
    }

    pub fn afterDisconnect(self: *const LoggingMiddleware, router: *HybridPipe) Error!void {
        self.log("Disconnected from router", .{});
        _ = router;
    }

    pub fn beforeDispatch(self: *const LoggingMiddleware, router: *HybridPipe, pipe: []const u8, data: []const u8) Error!void {
        self.log("Dispatching message to pipe: {s} (size: {d})", .{ pipe, data.len });
        _ = router;
    }

    pub fn afterDispatch(self: *const LoggingMiddleware, router: *HybridPipe, pipe: []const u8, data: []const u8) Error!void {
        self.log("Dispatched message to pipe: {s} (size: {d})", .{ pipe, data.len });
        _ = router;
    }

    pub fn beforeSubscribe(self: *const LoggingMiddleware, router: *HybridPipe, pipe: []const u8, callback: Process) Error!void {
        self.log("Subscribing to pipe: {s}", .{pipe});
        _ = router;
        _ = callback;
    }

    pub fn afterSubscribe(self: *const LoggingMiddleware, router: *HybridPipe, pipe: []const u8, callback: Process) Error!void {
        self.log("Subscribed to pipe: {s}", .{pipe});
        _ = router;
        _ = callback;
    }

    pub fn beforeUnsubscribe(self: *const LoggingMiddleware, router: *HybridPipe, pipe: []const u8) Error!void {
        self.log("Unsubscribing from pipe: {s}", .{pipe});
        _ = router;
    }

    pub fn afterUnsubscribe(self: *const LoggingMiddleware, router: *HybridPipe, pipe: []const u8) Error!void {
        self.log("Unsubscribed from pipe: {s}", .{pipe});
        _ = router;
    }

    pub fn beforeProcess(self: *const LoggingMiddleware, pipe: []const u8, data: []const u8) Error!void {
        self.log("Processing message from pipe: {s} (size: {d})", .{ pipe, data.len });
    }

    pub fn afterProcess(self: *const LoggingMiddleware, pipe: []const u8, data: []const u8) Error!void {
        self.log("Processed message from pipe: {s} (size: {d})", .{ pipe, data.len });
    }
};

// Test the middleware system
test "LoggingMiddleware" {
    const logging_middleware = LoggingMiddleware.init(.Debug);

    // This is just a compile-time test to ensure the middleware is well-defined
    try logging_middleware.beforeConnect(undefined);
    try logging_middleware.afterConnect(undefined);
    try logging_middleware.beforeDispatch(undefined, "test", "data");
    try logging_middleware.afterDispatch(undefined, "test", "data");
}
