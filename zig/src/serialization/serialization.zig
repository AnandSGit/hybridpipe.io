// Serialization system for HybridPipe
// Handles encoding and decoding of messages

const std = @import("std");
const Error = @import("../core/errors.zig").Error;

/// SerializationFormat represents the format used for serialization
pub const SerializationFormat = enum {
    /// Binary format (default)
    Binary,
    /// JSON format
    Json,
};

/// SerializationOptions configures the serialization behavior
pub const SerializationOptions = struct {
    /// Format specifies the serialization format to use
    format: SerializationFormat = .Binary,
    /// Compression enables compression for large messages
    compression: bool = false,
    /// CompressionLevel specifies the compression level (0-9, where 0 is no compression and 9 is maximum compression)
    compression_level: u8 = 6,

    pub fn init() SerializationOptions {
        return SerializationOptions{};
    }
};

/// Header for serialized messages
const MessageHeader = struct {
    /// Magic number to identify HybridPipe messages
    magic: [2]u8 = .{ 'H', 'P' },
    /// Version of the message format
    version: u8 = 1,
    /// Format used for serialization
    format: u8,
    /// Flags for additional options (e.g., compression)
    flags: u8 = 0,
    /// Reserved for future use
    reserved: [3]u8 = .{ 0, 0, 0 },
    /// Length of the payload
    payload_length: u32,

    pub fn init(format: SerializationFormat, compression: bool, payload_length: u32) MessageHeader {
        const format_byte: u8 = switch (format) {
            .Binary => 0,
            .Json => 1,
        };

        var flags: u8 = 0;
        if (compression) {
            flags |= 0x01; // Set compression flag
        }

        return MessageHeader{
            .format = format_byte,
            .flags = flags,
            .payload_length = payload_length,
        };
    }

    pub fn toBytes(self: MessageHeader) [12]u8 {
        var bytes: [12]u8 = undefined;

        bytes[0] = self.magic[0];
        bytes[1] = self.magic[1];
        bytes[2] = self.version;
        bytes[3] = self.format;
        bytes[4] = self.flags;
        bytes[5] = self.reserved[0];
        bytes[6] = self.reserved[1];
        bytes[7] = self.reserved[2];

        @memcpy(bytes[8..12], std.mem.asBytes(&self.payload_length));

        return bytes;
    }

    pub fn fromBytes(bytes: [12]u8) MessageHeader {
        var header = MessageHeader{
            .format = bytes[3],
            .flags = bytes[4],
            .payload_length = @as(*align(1) const u32, @ptrCast(bytes[8..12].ptr)).*,
        };

        header.magic[0] = bytes[0];
        header.magic[1] = bytes[1];
        header.version = bytes[2];
        header.reserved[0] = bytes[5];
        header.reserved[1] = bytes[6];
        header.reserved[2] = bytes[7];

        return header;
    }
};

/// Encode data using the default serialization options
pub fn encode(data: []const u8) ![]u8 {
    return encodeWithOptions(data, SerializationOptions.init());
}

/// Encode data using the specified serialization options
pub fn encodeWithOptions(data: []const u8, options: SerializationOptions) ![]u8 {
    const allocator = std.heap.page_allocator;

    // For now, we just use the data as-is
    // In a real implementation, we would serialize complex objects
    const payload = data;

    // Create the header
    const header = MessageHeader.init(options.format, options.compression, @intCast(payload.len));
    const header_bytes = header.toBytes();

    // Combine header and payload
    var result = try allocator.alloc(u8, header_bytes.len + payload.len);
    std.mem.copyForwards(u8, result[0..header_bytes.len], &header_bytes);
    std.mem.copyForwards(u8, result[header_bytes.len..], payload);

    return result;
}

/// Decode data using the default serialization options
pub fn decode(data: []const u8) ![]u8 {
    return decodeWithOptions(data, SerializationOptions.init());
}

/// Decode data using the specified serialization options
pub fn decodeWithOptions(data: []const u8, options: SerializationOptions) ![]u8 {
    _ = options;
    const allocator = std.heap.page_allocator;

    if (data.len < 12) {
        return Error.DeserializationFailed;
    }

    // Extract the header
    var header_bytes: [12]u8 = undefined;
    std.mem.copyForwards(u8, &header_bytes, data[0..12]);
    const header = MessageHeader.fromBytes(header_bytes);

    // Verify the magic number
    if (header.magic[0] != 'H' or header.magic[1] != 'P') {
        return Error.DeserializationFailed;
    }

    // Extract the payload
    const payload = try allocator.alloc(u8, header.payload_length);
    std.mem.copyForwards(u8, payload, data[12..][0..header.payload_length]);

    return payload;
}

// Test the serialization system
test "Serialization" {
    const test_data = "Hello, HybridPipe!";

    // Test encoding
    const encoded = try encode(test_data);
    defer std.heap.page_allocator.free(encoded);

    // Test decoding
    const decoded = try decode(encoded);
    defer std.heap.page_allocator.free(decoded);

    try std.testing.expectEqualStrings(test_data, decoded);
}

// Test the MessageHeader
test "MessageHeader" {
    const header = MessageHeader.init(.Binary, false, 100);
    const bytes = header.toBytes();
    const decoded_header = MessageHeader.fromBytes(bytes);

    try std.testing.expectEqual(header.format, decoded_header.format);
    try std.testing.expectEqual(header.flags, decoded_header.flags);
    try std.testing.expectEqual(header.payload_length, decoded_header.payload_length);
}
