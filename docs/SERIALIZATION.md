# HybridPipe Serialization

## Overview

HybridPipe provides a flexible serialization system that supports multiple formats and options. This document describes the serialization system in detail.

## Supported Formats

HybridPipe supports the following serialization formats:

- **GOB**: Go's native binary format (default for backward compatibility)
- **JSON**: Standard JSON format (language agnostic but less efficient)
- **Protocol Buffers**: Efficient binary format (language agnostic)
- **MessagePack**: Efficient binary format similar to JSON (language agnostic)

## Serialization Options

The serialization system can be configured using the `SerializationOptions` struct:

```go
type SerializationOptions struct {
    // Format is the serialization format to use.
    Format SerializationFormat
    // Compression indicates whether to compress the data.
    Compression bool
    // CompressionLevel is the compression level to use (0-9, where 0 is no compression
    // and 9 is maximum compression).
    CompressionLevel int
    // ProtobufMessageType is the fully qualified name of the Protocol Buffer message type.
    // This is only used when Format is FormatProtobuf.
    ProtobufMessageType string
    // TypeRegistry is the registry of type mappings for serialization.
    TypeRegistry *TypeRegistry
    // ProtobufRegistry is the registry of Protocol Buffer message types.
    ProtobufRegistry *ProtobufRegistry
    // CrossLanguageCompatible indicates whether to use cross-language compatible serialization.
    // This may limit the types that can be serialized.
    CrossLanguageCompatible bool
}
```

## Basic Usage

### Default Serialization

By default, HybridPipe uses GOB serialization for backward compatibility:

```go
// Encode data
data := map[string]interface{}{
    "message": "Hello, Serialization!",
    "count":   42,
}
encoded, err := hybridpipe.Encode(data)
if err != nil {
    log.Fatalf("Failed to encode data: %v", err)
}

// Decode data
var decoded map[string]interface{}
err = hybridpipe.Decode(encoded, &decoded)
if err != nil {
    log.Fatalf("Failed to decode data: %v", err)
}
```

### Custom Serialization

You can specify a different serialization format using `SerializationOptions`:

```go
// Create serialization options
options := &hybridpipe.SerializationOptions{
    Format:      hybridpipe.FormatJSON,
    Compression: true,
}

// Encode data with options
encoded, err := hybridpipe.EncodeWithOptions(data, options)
if err != nil {
    log.Fatalf("Failed to encode data: %v", err)
}

// Decode data with options
var decoded map[string]interface{}
err = hybridpipe.DecodeWithOptions(encoded, &decoded, options)
if err != nil {
    log.Fatalf("Failed to decode data: %v", err)
}
```

## Type Registration

When using GOB serialization, you need to register custom types before using them:

```go
// Define a custom type
type Person struct {
    Name string
    Age  int
}

// Register the type
hybridpipe.Enable(Person{})

// Now you can use the type
person := Person{Name: "Alice", Age: 30}
encoded, err := hybridpipe.Encode(person)
if err != nil {
    log.Fatalf("Failed to encode person: %v", err)
}

var decoded Person
err = hybridpipe.Decode(encoded, &decoded)
if err != nil {
    log.Fatalf("Failed to decode person: %v", err)
}
```

## Protocol Buffers

HybridPipe supports Protocol Buffers for efficient, language-agnostic serialization:

```go
// Define a Protocol Buffer message
type Person struct {
    Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
    Age  int32  `protobuf:"varint,2,opt,name=age,proto3" json:"age,omitempty"`
}

// Register the Protocol Buffer message type
hybridpipe.RegisterProtobufType("example.Person", &Person{})

// Create serialization options
options := &hybridpipe.SerializationOptions{
    Format:              hybridpipe.FormatProtobuf,
    ProtobufMessageType: "example.Person",
}

// Encode data with options
person := Person{Name: "Alice", Age: 30}
encoded, err := hybridpipe.EncodeWithOptions(person, options)
if err != nil {
    log.Fatalf("Failed to encode person: %v", err)
}

// Decode data with options
var decoded Person
err = hybridpipe.DecodeWithOptions(encoded, &decoded, options)
if err != nil {
    log.Fatalf("Failed to decode person: %v", err)
}
```

## Cross-Language Compatibility

HybridPipe supports cross-language compatible serialization using JSON, Protocol Buffers, or MessagePack:

```go
// Create serialization options
options := &hybridpipe.SerializationOptions{
    Format:                  hybridpipe.FormatJSON,
    CrossLanguageCompatible: true,
}

// Encode data with options
data := map[string]interface{}{
    "message": "Hello, Cross-Language Serialization!",
    "count":   42,
}
encoded, err := hybridpipe.EncodeWithOptions(data, options)
if err != nil {
    log.Fatalf("Failed to encode data: %v", err)
}

// The encoded data can be decoded by other languages that support JSON
```

## Type Conversion

HybridPipe supports automatic type conversion during serialization and deserialization:

```go
// Register a type conversion
hybridpipe.RegisterTypeMapping(
    reflect.TypeOf(time.Time{}),
    reflect.TypeOf(int64(0)),
    hybridpipe.TimeToUnixMillis,
    hybridpipe.UnixMillisToTime,
)

// Now time.Time will be automatically converted to int64 during serialization
// and back to time.Time during deserialization
data := map[string]interface{}{
    "timestamp": time.Now(),
}
encoded, err := hybridpipe.Encode(data)
if err != nil {
    log.Fatalf("Failed to encode data: %v", err)
}

var decoded map[string]interface{}
err = hybridpipe.Decode(encoded, &decoded)
if err != nil {
    log.Fatalf("Failed to decode data: %v", err)
}
// decoded["timestamp"] will be a time.Time
```

## Compression

HybridPipe supports automatic compression for large messages:

```go
// Create serialization options
options := &hybridpipe.SerializationOptions{
    Format:           hybridpipe.FormatJSON,
    Compression:      true,
    CompressionLevel: 6, // Default compression level
}

// Encode data with options
largeData := make([]byte, 1000000) // 1MB of data
rand.Read(largeData)               // Fill with random data
encoded, err := hybridpipe.EncodeWithOptions(largeData, options)
if err != nil {
    log.Fatalf("Failed to encode data: %v", err)
}

// The encoded data will be compressed if it's beneficial
fmt.Printf("Original size: %d, Encoded size: %d\n", len(largeData), len(encoded))

// Decode data with options
var decoded []byte
err = hybridpipe.DecodeWithOptions(encoded, &decoded, options)
if err != nil {
    log.Fatalf("Failed to decode data: %v", err)
}
```

## Performance Considerations

- **GOB** is the fastest for Go-to-Go communication but is not cross-language compatible.
- **Protocol Buffers** is the most efficient for cross-language communication but requires schema definition.
- **JSON** is the most widely supported but is less efficient than binary formats.
- **MessagePack** is a good compromise between efficiency and compatibility.

## Best Practices

- Use GOB for Go-to-Go communication when performance is critical.
- Use Protocol Buffers for cross-language communication when performance is critical.
- Use JSON for cross-language communication when compatibility is more important than performance.
- Use compression for large messages to reduce network bandwidth.
- Register custom types before using them with GOB serialization.
- Use type conversion for types that are not directly serializable.

## Conclusion

HybridPipe's serialization system provides a flexible and efficient way to encode and decode data for transmission through various messaging systems. By choosing the appropriate serialization format and options, you can optimize for performance, compatibility, or a balance of both.

Last updated: April 19, 2025
