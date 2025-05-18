# HybridPipe Serialization

This directory contains shared serialization schemas, format definitions, and language-agnostic components for the HybridPipe system.

## Purpose

The serialization system in HybridPipe serves as a bridge between different messaging protocols and language implementations. It provides:

1. **Consistent Message Format**: Standardized message structure across all protocols
2. **Cross-Language Compatibility**: Ensures messages can be exchanged between Go, Rust, and Zig implementations
3. **Protocol Interoperability**: Enables communication between different messaging systems
4. **Extensible Format Support**: Allows for multiple serialization formats (JSON, Protocol Buffers, MessagePack, etc.)

## Directory Structure

- `/formats/` - Contains schema definitions for each supported serialization format
  - `/formats/json/` - JSON schema definitions
  - `/formats/protobuf/` - Protocol Buffers schema definitions
  - `/formats/msgpack/` - MessagePack format specifications
  - `/formats/cbor/` - CBOR format specifications
  - `/formats/bson/` - BSON format specifications
- `/common/` - Language-agnostic utilities and helpers for serialization

## Relationship to Language-Specific Implementations

Each language implementation (Go, Rust, Zig) has its own serialization module that implements the formats defined in this directory:

- Go: `golang/serialization/`
- Rust: `rust/src/serialization/`
- Zig: `zig/src/serialization/`

The language-specific implementations should:

1. Adhere to the schemas and format definitions in this directory
2. Implement the serialization and deserialization functions for each supported format
3. Maintain compatibility with the other language implementations

## Message Format

All HybridPipe messages follow a standard envelope format:

```
+----------------+----------------+----------------+
| Magic (2 bytes) | Header (10 bytes) | Payload (variable) |
+----------------+----------------+----------------+
```

- **Magic**: A 2-byte identifier ('HP') that marks this as a HybridPipe message
- **Header**: Contains metadata about the message (format, compression, etc.)
- **Payload**: The actual message content, serialized according to the specified format

## Adding New Formats

To add support for a new serialization format:

1. Create a new directory under `/formats/` with the format name
2. Add schema definitions and documentation for the format
3. Update each language-specific implementation to support the new format
4. Update the serialization documentation in `docs/SERIALIZATION.md`

## Testing Cross-Language Compatibility

When making changes to the serialization system, ensure that messages serialized in one language implementation can be correctly deserialized in the others. The test suite includes cross-language compatibility tests for this purpose.

## Documentation

For more detailed information about the serialization system, see the [Serialization Documentation](../docs/SERIALIZATION.md).
