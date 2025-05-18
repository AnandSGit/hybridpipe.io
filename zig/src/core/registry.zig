// Registry system for HybridPipe
// Manages the registration and instantiation of protocol implementations

const std = @import("std");
const Error = @import("errors.zig").Error;
const BrokerType = @import("../protocols/protocol_types.zig").BrokerType;
const HybridPipe = @import("core.zig").HybridPipe;
const RouterFactory = @import("core.zig").RouterFactory;

/// Registry manages the registration of protocol implementations
pub const Registry = struct {
    factories: std.AutoHashMap(BrokerType, RouterFactory),
    allocator: std.mem.Allocator,

    pub fn init(allocator: std.mem.Allocator) Registry {
        return Registry{
            .factories = std.AutoHashMap(BrokerType, RouterFactory).init(allocator),
            .allocator = allocator,
        };
    }

    pub fn deinit(self: *Registry) void {
        self.factories.deinit();
    }

    pub fn registerRouterFactory(self: *Registry, broker_type: BrokerType, factory: RouterFactory) !void {
        try self.factories.put(broker_type, factory);
    }

    pub fn deployRouter(self: *Registry, broker_type: BrokerType) !*HybridPipe {
        const factory = self.factories.get(broker_type) orelse {
            return Error.BrokerTypeNotRegistered;
        };

        var router = try factory(self.allocator);
        try router.connect();
        return router;
    }
};

// Global registry instance
var global_registry_instance: Registry = undefined;
var global_registry_initialized: bool = false;

/// Initialize the registry with the given allocator
pub fn initRegistry(allocator: std.mem.Allocator) void {
    if (!global_registry_initialized) {
        global_registry_instance = Registry.init(allocator);
        global_registry_initialized = true;
    }
}

/// Get the global registry instance
pub fn getRegistry() !*Registry {
    if (!global_registry_initialized) {
        return Error.InternalError;
    }
    return &global_registry_instance;
}

/// Register a router factory for a specific broker type
pub fn registerRouterFactory(broker_type: BrokerType, factory: RouterFactory) !void {
    var registry = try getRegistry();
    try registry.registerRouterFactory(broker_type, factory);
}

/// Deploy a router for a specific broker type
pub fn deployRouter(broker_type: BrokerType) !*HybridPipe {
    var registry = try getRegistry();
    return registry.deployRouter(broker_type);
}

// Test the registry system
test "Registry" {
    // We'll test the registry with actual protocol implementations
    // in the protocol-specific tests
}
