# MessagePack Format for HybridPipe Messages

This document describes the MessagePack format used for HybridPipe messages.

## Overview

MessagePack is a binary serialization format that is more compact than JSON. HybridPipe uses MessagePack to serialize messages for efficient transmission over the network.

## Message Structure

A HybridPipe message serialized with MessagePack has the following structure:

```
+----------------+----------------+----------------+
| Magic (2 bytes) | Header (map)    | Payload (any)   |
+----------------+----------------+----------------+
```

- **Magic**: A 2-byte identifier ('HP') that marks this as a HybridPipe message
- **Header**: A MessagePack map containing metadata about the message
- **Payload**: The actual message content, which can be any MessagePack-supported type

## Header Fields

The header is a MessagePack map with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `v` | uint | Message format version (currently 1) |
| `f` | uint | Serialization format (0=JSON, 1=Protobuf, 2=MessagePack, 3=CBOR, 4=BSON) |
| `c` | bool | Whether the payload is compressed |
| `t` | uint | Timestamp (milliseconds since epoch) |
| `id` | str | Message ID |
| `cid` | str | Correlation ID (optional) |
| `r` | str | Reply-to pipe name (optional) |
| `ttl` | uint | Time-to-live in milliseconds (optional) |
| `p` | uint | Priority (0-9, default 4) (optional) |
| `ct` | str | Content type (optional) |

## Payload

The payload can be any valid MessagePack value, including:

- Maps (objects)
- Arrays
- Strings
- Numbers
- Booleans
- Null

## Example

Here's an example of a HybridPipe message serialized with MessagePack (shown in hex):

```
48 50                                      # Magic "HP"
8A                                         # Map with 10 elements
A1 76 01                                   # "v": 1
A1 66 02                                   # "f": 2 (MessagePack)
A1 63 C3                                   # "c": true
A1 74 CF 00 00 01 7D 78 6E 12 00          # "t": 1620000000000
A2 69 64 B4 61 62 63 64 31 32 33 34       # "id": "abcd1234"
A3 63 69 64 A0                            # "cid": ""
A1 72 A0                                  # "r": ""
A3 74 74 6C 00                            # "ttl": 0
A1 70 04                                  # "p": 4
A2 63 74 A0                               # "ct": ""
83                                         # Map with 3 elements
A4 6E 61 6D 65 A4 4A 6F 68 6E             # "name": "John"
A3 61 67 65 1E                            # "age": 30
A5 65 6D 61 69 6C A0                      # "email": ""
```

## Compression

If the `c` field in the header is `true`, the payload is compressed using the zlib algorithm. The compressed payload must be decompressed before it can be interpreted as a MessagePack value.

## Interoperability

MessagePack-serialized HybridPipe messages can be deserialized by any language implementation that supports MessagePack. The HybridPipe libraries for Go, Rust, and Zig all include support for MessagePack serialization and deserialization.
