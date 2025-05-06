# HybridPipe Serialization Package

The serialization package provides functionality for encoding and decoding data in various formats for transmission through the HybridPipe messaging system.

## Features

- **Multiple Formats**: Support for Gob, JSON, Protocol Buffers, and MessagePack
- **Compression**: Optional compression for large messages
- **Type Conversion**: Automatic conversion between Go types and wire formats
- **Cross-Language Compatibility**: Options for ensuring compatibility with non-Go clients
- **Protocol Buffers Integration**: Enhanced support for Protocol Buffers

## Serialization Formats

| Format | Description | Pros | Cons | Best For |
|--------|-------------|------|------|----------|
| **Gob** | Go's native binary format | Efficient for Go types | Go-only | Go-to-Go communication |
| **JSON** | Text-based, human-readable format | Human-readable, universal support | Larger size, slower | Debugging, web integration |
| **Protocol Buffers** | Binary format with schema | Compact, fast, schema validation | Requires schema definition | High-performance, strict typing |
| **MessagePack** | Binary JSON alternative | Compact, fast, schema-less | Less human-readable | Efficient JSON alternative |

## Usage

### Basic Usage

```go
// Encode data
bytes, err := serialization.Encode(data)
if err != nil {
    // Handle error
}

// Decode data
var result MyType
err = serialization.Decode(bytes, &result)
if err != nil {
    // Handle error
}
```

### Advanced Usage

```go
// Create serialization options
options := serialization.SerializationOptions{
    Format:            serialization.FormatJSON,
    Compression:       true,
    CompressionLevel:  6,
    CrossLanguageCompatible: true,
}

// Encode data with options
bytes, err := serialization.EncodeWithOptions(data, options)
if err != nil {
    // Handle error
}

// Decode data with options
var result MyType
err = serialization.DecodeWithOptions(bytes, &result, options)
if err != nil {
    // Handle error
}
```

### Type Registration

For Gob encoding, you need to register custom types:

```go
// Register a custom type
serialization.Enable(MyType{})
```

### Protocol Buffers

For Protocol Buffers, you can register message types:

```go
// Register a Protocol Buffer message type
serialization.RegisterProtobufType(MyMessage{})
```
