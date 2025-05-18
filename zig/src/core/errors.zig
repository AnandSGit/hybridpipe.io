// Error types for HybridPipe
// Defines the error types used throughout the system

const std = @import("std");
const BrokerType = @import("../protocols/protocol_types.zig").BrokerType;

/// Error represents all possible errors that can occur in HybridPipe
pub const Error = error{
    // Connection errors
    ConnectionFailed,
    AlreadyConnected,
    NotConnected,

    // Dispatch errors
    DispatchFailed,
    InvalidPipe,

    // Subscription errors
    SubscriptionFailed,
    UnsubscribeFailed,

    // Serialization errors
    SerializationFailed,
    DeserializationFailed,

    // Registry errors
    BrokerTypeNotRegistered,

    // Configuration errors
    ConfigurationError,

    // I/O errors
    IoError,

    // Timeout errors
    Timeout,

    // General errors
    OutOfMemory,
    InvalidArgument,
    NotImplemented,
    InternalError,

    // Network errors
    InvalidIPAddressFormat,
    Unexpected,
    DiskQuota,
    FileTooBig,
    InputOutput,
    NoSpaceLeft,
    DeviceBusy,
    AccessDenied,
    BrokenPipe,
    SystemResources,
    OperationAborted,
    NotOpenForWriting,
    LockViolation,
    WouldBlock,
    ConnectionResetByPeer,
    FileNotFound,
    ProcessFdQuotaExceeded,
    SystemFdQuotaExceeded,
    PermissionDenied,
    ConnectionTimedOut,
    AddressFamilyNotSupported,
    ProtocolFamilyNotAvailable,
    ProtocolNotSupported,
    SocketTypeNotSupported,
    AddressInUse,
    AddressNotAvailable,
    ConnectionRefused,
    NetworkUnreachable,
    ConnectionPending,
};

/// ErrorDetail provides additional context for an error
pub const ErrorDetail = struct {
    error_type: Error,
    message: []const u8,
    broker_type: ?BrokerType = null,

    pub fn init(error_type: Error, message: []const u8) ErrorDetail {
        return ErrorDetail{
            .error_type = error_type,
            .message = message,
            .broker_type = null,
        };
    }

    pub fn initWithBroker(error_type: Error, message: []const u8, broker_type: BrokerType) ErrorDetail {
        return ErrorDetail{
            .error_type = error_type,
            .message = message,
            .broker_type = broker_type,
        };
    }

    pub fn format(self: ErrorDetail, comptime fmt: []const u8, options: std.fmt.FormatOptions, writer: anytype) !void {
        _ = fmt;
        _ = options;

        try writer.print("Error: {s} - {s}", .{ @errorName(self.error_type), self.message });

        if (self.broker_type) |broker| {
            try writer.print(" (Broker: {s})", .{broker.toString()});
        }
    }
};

// Test the Error type
test "Error" {
    const err = Error.ConnectionFailed;
    try std.testing.expectEqual(err, Error.ConnectionFailed);
}

// Test the ErrorDetail struct
test "ErrorDetail" {
    const detail = ErrorDetail.init(Error.ConnectionFailed, "Failed to connect to server");
    try std.testing.expectEqual(detail.error_type, Error.ConnectionFailed);
    try std.testing.expectEqualStrings(detail.message, "Failed to connect to server");
    try std.testing.expect(detail.broker_type == null);

    const detail_with_broker = ErrorDetail.initWithBroker(Error.ConnectionFailed, "Failed to connect to server", BrokerType.TCP);
    try std.testing.expectEqual(detail_with_broker.error_type, Error.ConnectionFailed);
    try std.testing.expectEqualStrings(detail_with_broker.message, "Failed to connect to server");
    try std.testing.expect(detail_with_broker.broker_type != null);
    try std.testing.expectEqual(detail_with_broker.broker_type.?, BrokerType.TCP);
}
