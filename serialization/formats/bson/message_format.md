# BSON Format for HybridPipe Messages

This document describes the BSON (Binary JSON) format used for HybridPipe messages.

## Overview

BSON is a binary-encoded serialization of JSON-like documents, originally designed for MongoDB. HybridPipe supports BSON as one of its serialization formats, particularly useful for applications that interact with MongoDB or require BSON's specific features.

## Message Structure

A HybridPipe message serialized with BSON has the following structure:

```
+----------------+----------------+----------------+
| Magic (2 bytes) | Header (document) | Payload (any)   |
+----------------+----------------+----------------+
```

- **Magic**: A 2-byte identifier ('HP') that marks this as a HybridPipe message
- **Header**: A BSON document containing metadata about the message
- **Payload**: The actual message content, which can be any BSON-supported type

## Header Fields

The header is a BSON document with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `version` | int32 | Message format version (currently 1) |
| `format` | int32 | Serialization format (0=JSON, 1=Protobuf, 2=MessagePack, 3=CBOR, 4=BSON) |
| `compression` | boolean | Whether the payload is compressed |
| `timestamp` | int64 | Timestamp (milliseconds since epoch) |
| `messageId` | string | Message ID |
| `correlationId` | string | Correlation ID (optional) |
| `replyTo` | string | Reply-to pipe name (optional) |
| `ttl` | int32 | Time-to-live in milliseconds (optional) |
| `priority` | int32 | Priority (0-9, default 4) (optional) |
| `contentType` | string | Content type (optional) |

## Payload

The payload can be any valid BSON value, including:

- Documents (objects)
- Arrays
- Strings
- Numbers (int32, int64, double)
- Booleans
- Null
- Binary data
- ObjectId
- Date
- Regular expressions
- JavaScript code
- Timestamps

## BSON Types

BSON supports the following data types, which can be used in HybridPipe messages:

| Type Code | Name | Description |
|-----------|------|-------------|
| 0x01 | Double | 64-bit binary floating point |
| 0x02 | String | UTF-8 string |
| 0x03 | Document | BSON document |
| 0x04 | Array | BSON array |
| 0x05 | Binary | Binary data |
| 0x07 | ObjectId | 12-byte ObjectId |
| 0x08 | Boolean | true or false |
| 0x09 | Date | 64-bit integer milliseconds since the Unix epoch |
| 0x0A | Null | Null value |
| 0x0B | RegExp | Regular expression |
| 0x0D | JavaScript | JavaScript code |
| 0x10 | Int32 | 32-bit integer |
| 0x12 | Int64 | 64-bit integer |
| 0x13 | Timestamp | BSON timestamp (different from Date) |

## Compression

If the `compression` field in the header is `true`, the payload is compressed using the ZLIB algorithm. The compressed payload must be decompressed before it can be interpreted as a BSON value.

## Example

Here's an example of a HybridPipe message serialized with BSON (shown in hex):

```
48 50                                      # Magic "HP"
# BSON Header Document
4D 00 00 00                                # Document size (77 bytes)
10 76 65 72 73 69 6F 6E 00 01 00 00 00     # "version": 1
10 66 6F 72 6D 61 74 00 04 00 00 00        # "format": 4 (BSON)
08 63 6F 6D 70 72 65 73 73 69 6F 6E 00 01  # "compression": true
12 74 69 6D 65 73 74 61 6D 70 00 80 3C 94 35 66 01 00 00 # "timestamp": 1620000000000
02 6D 65 73 73 61 67 65 49 64 00 09 00 00 00 61 62 63 64 31 32 33 34 00 # "messageId": "abcd1234"
00                                         # End of document

# BSON Payload Document
3A 00 00 00                                # Document size (58 bytes)
02 6E 61 6D 65 00 05 00 00 00 4A 6F 68 6E 00 # "name": "John"
10 61 67 65 00 1E 00 00 00                 # "age": 30
02 65 6D 61 69 6C 00 01 00 00 00 00        # "email": ""
00                                         # End of document
```

## Interoperability

BSON-serialized HybridPipe messages can be deserialized by any language implementation that supports BSON. The HybridPipe libraries for Go, Rust, and Zig all include support for BSON serialization and deserialization.

## MongoDB Compatibility

BSON is the native format used by MongoDB, making it particularly useful for applications that interact with MongoDB databases. When using HybridPipe with MongoDB, consider using BSON as the serialization format for maximum compatibility and performance.
