# HybridPipe Message Header Specification

This document defines the standard message header format used across all HybridPipe implementations.

## Overview

Every HybridPipe message includes a header that contains metadata about the message. This header is used by the HybridPipe system to route, process, and manage messages.

## Binary Format

When transmitted over the wire, the HybridPipe message header has the following binary format:

```
+----------------+----------------+----------------+----------------+
| Magic (2 bytes) | Version (1 byte) | Format (1 byte) | Flags (1 byte)  |
+----------------+----------------+----------------+----------------+
| Reserved (3 bytes)              | Payload Length (4 bytes)        |
+----------------+----------------+----------------+----------------+
```

### Field Descriptions

- **Magic**: A 2-byte identifier ('HP') that marks this as a HybridPipe message
- **Version**: The version of the message format (currently 1)
- **Format**: The serialization format used for the payload:
  - 0: JSON
  - 1: Protocol Buffers
  - 2: MessagePack
  - 3: CBOR
  - 4: BSON
- **Flags**: Bit flags for message properties:
  - Bit 0: Compression (1 = compressed, 0 = not compressed)
  - Bit 1-7: Reserved for future use
- **Reserved**: 3 bytes reserved for future use
- **Payload Length**: The length of the payload in bytes (4-byte unsigned integer, little-endian)

## Extended Header

In addition to the binary header, HybridPipe messages can include an extended header with additional metadata. The extended header is included in the serialized payload and has the following fields:

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `timestamp` | uint64 | Timestamp (milliseconds since epoch) | Yes |
| `messageId` | string | Unique identifier for the message | Yes |
| `correlationId` | string | Identifier for correlating related messages | No |
| `replyTo` | string | Pipe name to send replies to | No |
| `ttl` | uint32 | Time-to-live in milliseconds | No |
| `priority` | uint8 | Message priority (0-9, default 4) | No |
| `contentType` | string | MIME type of the payload content | No |

## Language-Specific Implementations

Each language implementation should provide a `MessageHeader` struct/class that includes all the fields defined above. The implementation should also provide methods for serializing and deserializing the header.

### Go Example

```go
type MessageHeader struct {
    Version       uint8
    Format        SerializationFormat
    Compression   bool
    Timestamp     int64
    MessageID     string
    CorrelationID string
    ReplyTo       string
    TTL           uint32
    Priority      uint8
    ContentType   string
}
```

### Rust Example

```rust
pub struct MessageHeader {
    pub version: u8,
    pub format: SerializationFormat,
    pub compression: bool,
    pub timestamp: u64,
    pub message_id: String,
    pub correlation_id: Option<String>,
    pub reply_to: Option<String>,
    pub ttl: Option<u32>,
    pub priority: u8,
    pub content_type: Option<String>,
}
```

### Zig Example

```zig
pub const MessageHeader = struct {
    version: u8,
    format: SerializationFormat,
    compression: bool,
    timestamp: u64,
    message_id: []const u8,
    correlation_id: ?[]const u8,
    reply_to: ?[]const u8,
    ttl: ?u32,
    priority: u8,
    content_type: ?[]const u8,
};
```

## Serialization

When serializing a message, the header should be included in the serialized payload according to the format-specific guidelines:

- **JSON**: Include the header as a JSON object with the key "header"
- **Protocol Buffers**: Include the header as a Protocol Buffers message with the field number 1
- **MessagePack**: Include the header as a MessagePack map with the key "header"
- **CBOR**: Include the header as a CBOR map with the key "header"
- **BSON**: Include the header as a BSON document with the key "header"

## Versioning

The message format version (the "Version" field in the binary header) should be incremented whenever there is a breaking change to the message format. All HybridPipe implementations should support at least version 1 of the message format.

## Compatibility

To ensure compatibility between different language implementations, all implementations must adhere to this specification. Messages created by one implementation should be correctly interpreted by all other implementations.
