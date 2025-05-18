// Protocol types for HybridPipe
// Defines the broker types supported by HybridPipe

const std = @import("std");

/// BrokerType represents the type of messaging system to use
pub const BrokerType = enum {
    // TCP/IP direct communication
    TCP,
    // In-memory mock for testing
    MOCK,
    // Reserved for future implementations
    NATS,
    KAFKA,
    RABBITMQ,
    ZEROMQ,
    AMQP1,
    MQTT,
    QPID,
    NSQ,
    REDIS,
    NETCHAN,

    pub fn toString(self: BrokerType) []const u8 {
        return switch (self) {
            .TCP => "TCP",
            .MOCK => "MOCK",
            .NATS => "NATS",
            .KAFKA => "KAFKA",
            .RABBITMQ => "RABBITMQ",
            .ZEROMQ => "ZEROMQ",
            .AMQP1 => "AMQP1",
            .MQTT => "MQTT",
            .QPID => "QPID",
            .NSQ => "NSQ",
            .REDIS => "REDIS",
            .NETCHAN => "NETCHAN",
        };
    }
};

// Test the BrokerType enum
test "BrokerType" {
    const tcp = BrokerType.TCP;
    try std.testing.expectEqualStrings("TCP", tcp.toString());
    
    const mock = BrokerType.MOCK;
    try std.testing.expectEqualStrings("MOCK", mock.toString());
}
