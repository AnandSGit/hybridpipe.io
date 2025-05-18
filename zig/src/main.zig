// HybridPipe - A messaging middleware system
// Main entry point for the library

// Export core components
pub const core = @import("core/core.zig");
pub const errors = @import("core/errors.zig");
pub const config = @import("core/config.zig");
pub const registry = @import("core/registry.zig");

// Export protocols
pub const protocols = struct {
    pub const tcp = @import("protocols/tcp/tcp.zig");
    pub const mock = @import("protocols/mock/mock.zig");
    pub const protocol_types = @import("protocols/protocol_types.zig");
};

// Export serialization
pub const serialization = @import("serialization/serialization.zig");

// Export middleware
pub const middleware = @import("middleware/middleware.zig");

// Export monitoring
pub const monitoring = @import("monitoring/monitoring.zig");

// Re-export common types and functions for convenience
pub const HybridPipe = core.HybridPipe;
pub const BrokerType = protocols.protocol_types.BrokerType;
pub const Error = errors.Error;

// Constants for broker types
pub const TCP = protocols.protocol_types.BrokerType.TCP;
pub const MOCK = protocols.protocol_types.BrokerType.MOCK;

// Initialize the library
pub fn init() void {
    // Register all protocol implementations
    protocols.tcp.register();
    protocols.mock.register();
}

// Deploy a router for a specific broker type
pub fn deployRouter(broker_type: BrokerType) !*HybridPipe {
    return registry.deployRouter(broker_type);
}

// Test function
test "main" {
    // Run all tests
    _ = @import("core/core.zig");
    _ = @import("core/errors.zig");
    _ = @import("core/config.zig");
    _ = @import("core/registry.zig");
    _ = @import("protocols/tcp/tcp.zig");
    _ = @import("protocols/mock/mock.zig");
    _ = @import("serialization/serialization.zig");
    _ = @import("middleware/middleware.zig");
    _ = @import("monitoring/monitoring.zig");
}
