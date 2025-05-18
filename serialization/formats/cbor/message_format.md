# CBOR Format for HybridPipe Messages

This document describes the CBOR (Concise Binary Object Representation) format used for HybridPipe messages.

## Overview

CBOR is a binary data format inspired by JSON but designed for smaller message sizes and faster processing. HybridPipe uses CBOR as one of its serialization options for efficient message transmission.

## Message Structure

A HybridPipe message serialized with CBOR has the following structure:

```
+----------------+----------------+----------------+
| Magic (2 bytes) | Header (map)    | Payload (any)   |
+----------------+----------------+----------------+
```

- **Magic**: A 2-byte identifier ('HP') that marks this as a HybridPipe message
- **Header**: A CBOR map containing metadata about the message
- **Payload**: The actual message content, which can be any CBOR-supported type

## Header Fields

The header is a CBOR map with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `version` | unsigned int | Message format version (currently 1) |
| `format` | unsigned int | Serialization format (0=JSON, 1=Protobuf, 2=MessagePack, 3=CBOR, 4=BSON) |
| `compression` | boolean | Whether the payload is compressed |
| `timestamp` | unsigned int | Timestamp (milliseconds since epoch) |
| `messageId` | text string | Message ID |
| `correlationId` | text string | Correlation ID (optional) |
| `replyTo` | text string | Reply-to pipe name (optional) |
| `ttl` | unsigned int | Time-to-live in milliseconds (optional) |
| `priority` | unsigned int | Priority (0-9, default 4) (optional) |
| `contentType` | text string | Content type (optional) |

## Payload

The payload can be any valid CBOR value, including:

- Maps
- Arrays
- Text strings
- Byte strings
- Integers
- Floating-point numbers
- Booleans
- Null

## CBOR Tags

HybridPipe uses the following CBOR tags for special data types:

| Tag | Data Type |
|-----|-----------|
| 0 | Date/Time (RFC 3339 text string) |
| 1 | Timestamp (seconds since epoch) |
| 2 | Unsigned bignum |
| 3 | Negative bignum |
| 4 | Decimal fraction |
| 5 | Bigfloat |
| 21 | Expected conversion to base64url encoding |
| 22 | Expected conversion to base64 encoding |
| 23 | Expected conversion to base16 encoding |
| 24 | Encoded CBOR data item |
| 32 | URI |
| 33 | Base64url |
| 34 | Base64 |
| 35 | Regular expression |
| 36 | MIME message |

## Compression

If the `compression` field in the header is `true`, the payload is compressed using the ZLIB algorithm. The compressed payload must be decompressed before it can be interpreted as a CBOR value.

## Example

Here's an example of a HybridPipe message serialized with CBOR (shown in hex):

```
48 50                                      # Magic "HP"
AA                                         # Map with 10 elements
67 76 65 72 73 69 6F 6E 01                # "version": 1
66 66 6F 72 6D 61 74 03                   # "format": 3 (CBOR)
6B 63 6F 6D 70 72 65 73 73 69 6F 6E F5    # "compression": true
69 74 69 6D 65 73 74 61 6D 70 1A 60 8F 73 E0 # "timestamp": 1620000000000
69 6D 65 73 73 61 67 65 49 64 68 61 62 63 64 31 32 33 34 # "messageId": "abcd1234"
6D 63 6F 72 72 65 6C 61 74 69 6F 6E 49 64 60 # "correlationId": ""
67 72 65 70 6C 79 54 6F 60                # "replyTo": ""
63 74 74 6C 00                            # "ttl": 0
68 70 72 69 6F 72 69 74 79 04             # "priority": 4
6B 63 6F 6E 74 65 6E 74 54 79 70 65 60    # "contentType": ""
A3                                         # Map with 3 elements
64 6E 61 6D 65 64 4A 6F 68 6E             # "name": "John"
63 61 67 65 18 1E                         # "age": 30
65 65 6D 61 69 6C 60                      # "email": ""
```

## Interoperability

CBOR-serialized HybridPipe messages can be deserialized by any language implementation that supports CBOR. The HybridPipe libraries for Go, Rust, and Zig all include support for CBOR serialization and deserialization.
