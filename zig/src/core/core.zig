// Core module for HybridPipe
// Defines the core interfaces and functionality

const std = @import("std");
const Error = @import("errors.zig").Error;
const BrokerType = @import("../protocols/protocol_types.zig").BrokerType;

/// Process defines the callback function that should be called when a client receives a message
pub const Process = *const fn (data: []const u8) Error!void;

/// HybridPipe defines the core interface for all messaging implementations
pub const HybridPipe = struct {
    // Interface vtable
    vtable: *const VTable,

    // Interface methods
    pub fn connect(self: *HybridPipe) Error!void {
        return self.vtable.connect(self);
    }

    pub fn disconnect(self: *HybridPipe) Error!void {
        return self.vtable.disconnect(self);
    }

    pub fn dispatch(self: *HybridPipe, pipe: []const u8, data: []const u8) Error!void {
        return self.vtable.dispatch(self, pipe, data);
    }

    pub fn subscribe(self: *HybridPipe, pipe: []const u8, callback: Process) Error!void {
        return self.vtable.subscribe(self, pipe, callback);
    }

    pub fn unsubscribe(self: *HybridPipe, pipe: []const u8) Error!void {
        return self.vtable.unsubscribe(self, pipe);
    }

    pub fn isConnected(self: *HybridPipe) bool {
        return self.vtable.isConnected(self);
    }

    // Aliases for backward compatibility
    pub fn accept(self: *HybridPipe, pipe: []const u8, callback: Process) Error!void {
        return self.subscribe(pipe, callback);
    }

    pub fn remove(self: *HybridPipe, pipe: []const u8) Error!void {
        return self.unsubscribe(pipe);
    }

    pub fn close(self: *HybridPipe) Error!void {
        return self.disconnect();
    }

    // Virtual method table
    pub const VTable = struct {
        connect: *const fn (self: *HybridPipe) Error!void,
        disconnect: *const fn (self: *HybridPipe) Error!void,
        dispatch: *const fn (self: *HybridPipe, pipe: []const u8, data: []const u8) Error!void,
        subscribe: *const fn (self: *HybridPipe, pipe: []const u8, callback: Process) Error!void,
        unsubscribe: *const fn (self: *HybridPipe, pipe: []const u8) Error!void,
        isConnected: *const fn (self: *HybridPipe) bool,
    };
};

/// RouterFactory is a function that creates a new instance of a HybridPipe implementation
pub const RouterFactory = *const fn (allocator: std.mem.Allocator) Error!*HybridPipe;

// Test the HybridPipe interface
test "HybridPipe interface" {
    // This is just a compile-time test to ensure the interface is well-defined
    // We'll test the actual implementation in the protocol-specific tests
}
