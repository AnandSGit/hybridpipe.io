// Configuration system for HybridPipe
// Handles loading and processing configuration for all protocols

const std = @import("std");
const Error = @import("errors.zig").Error;
const BrokerType = @import("../protocols/protocol_types.zig").BrokerType;

/// IoMode represents the I/O mode for the system
pub const IoMode = enum {
    /// Blocking I/O mode
    blocking,
    /// Evented (async) I/O mode
    evented,
};

/// GlobalConfig contains global configuration for the HybridPipe system
pub const GlobalConfig = struct {
    /// I/O mode for the system
    io_mode: IoMode = .blocking,
    /// Default allocator to use
    allocator: std.mem.Allocator,
    /// Debug mode
    debug: bool = false,

    pub fn init(allocator: std.mem.Allocator) GlobalConfig {
        return GlobalConfig{
            .allocator = allocator,
        };
    }
};

/// TcpConfig contains configuration for TCP protocol
pub const TcpConfig = struct {
    /// Server address
    server: []const u8 = "localhost",
    /// Server port
    port: u16 = 8080,
    /// Connection timeout in milliseconds
    timeout_ms: u32 = 5000,
    /// Keep alive
    keep_alive: bool = true,

    pub fn init() TcpConfig {
        return TcpConfig{};
    }
};

/// MockConfig contains configuration for the Mock protocol
pub const MockConfig = struct {
    /// Simulated latency in milliseconds
    latency_ms: u32 = 0,
    /// Simulate random failures
    simulate_failures: bool = false,
    /// Failure rate (0.0 - 1.0)
    failure_rate: f32 = 0.0,

    pub fn init() MockConfig {
        return MockConfig{};
    }
};

/// ProtocolConfig is a union of all protocol-specific configurations
pub const ProtocolConfig = union(BrokerType) {
    TCP: TcpConfig,
    MOCK: MockConfig,
    NATS: void,
    KAFKA: void,
    RABBITMQ: void,
    ZEROMQ: void,
    AMQP1: void,
    MQTT: void,
    QPID: void,
    NSQ: void,
    REDIS: void,
    NETCHAN: void,

    pub fn init(broker_type: BrokerType) ProtocolConfig {
        return switch (broker_type) {
            .TCP => ProtocolConfig{ .TCP = TcpConfig.init() },
            .MOCK => ProtocolConfig{ .MOCK = MockConfig.init() },
            else => @panic("Protocol not implemented"),
        };
    }
};

/// ConfigManager handles loading and storing configuration
pub const ConfigManager = struct {
    global_config: GlobalConfig,
    protocol_configs: std.AutoHashMap(BrokerType, ProtocolConfig),

    pub fn init(allocator: std.mem.Allocator) ConfigManager {
        return ConfigManager{
            .global_config = GlobalConfig.init(allocator),
            .protocol_configs = std.AutoHashMap(BrokerType, ProtocolConfig).init(allocator),
        };
    }

    pub fn deinit(self: *ConfigManager) void {
        self.protocol_configs.deinit();
    }

    pub fn setProtocolConfig(self: *ConfigManager, config: ProtocolConfig) !void {
        const broker_type = switch (config) {
            .TCP => BrokerType.TCP,
            .MOCK => BrokerType.MOCK,
            else => return Error.NotImplemented,
        };

        try self.protocol_configs.put(broker_type, config);
    }

    pub fn getProtocolConfig(self: *ConfigManager, broker_type: BrokerType) !ProtocolConfig {
        return self.protocol_configs.get(broker_type) orelse ProtocolConfig.init(broker_type);
    }
};

// Global configuration manager
var global_config_manager: ?ConfigManager = null;

/// Initialize the configuration system
pub fn initConfig(allocator: std.mem.Allocator) void {
    if (global_config_manager == null) {
        global_config_manager = ConfigManager.init(allocator);
    }
}

/// Get the global configuration manager
pub fn getConfigManager() !*ConfigManager {
    return &global_config_manager orelse return Error.ConfigurationError;
}

// Test the configuration system
test "ConfigManager" {
    const allocator = std.testing.allocator;
    var config_manager = ConfigManager.init(allocator);
    defer config_manager.deinit();

    // Test TCP config
    const tcp_config = TcpConfig{
        .server = "example.com",
        .port = 9090,
    };
    const protocol_config = ProtocolConfig{ .TCP = tcp_config };
    try config_manager.setProtocolConfig(protocol_config);

    const retrieved_config = try config_manager.getProtocolConfig(BrokerType.TCP);
    try std.testing.expectEqualStrings("example.com", retrieved_config.TCP.server);
    try std.testing.expectEqual(@as(u16, 9090), retrieved_config.TCP.port);
}
