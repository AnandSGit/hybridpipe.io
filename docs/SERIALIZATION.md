# Serialization Documentation

HybridPipe.io uses language and OS agnostic data serialization formats to ensure interoperability between different systems. This document describes the serialization formats supported by HybridPipe.io and how to use them.

## Overview

Serialization is the process of converting data structures or objects into a format that can be stored or transmitted and reconstructed later. HybridPipe.io supports multiple serialization formats to accommodate different use cases and performance requirements.

## Supported Formats

HybridPipe.io supports the following serialization formats:

1. **JSON**: A lightweight data-interchange format that is easy to read and write
2. **Protocol Buffers**: A language-neutral, platform-neutral, extensible mechanism for serializing structured data
3. **MessagePack**: A binary serialization format that is more compact than JSON
4. **CBOR**: Concise Binary Object Representation, a binary data format inspired by JSON
5. **BSON**: Binary JSON, a binary-encoded serialization of JSON-like documents

## Default Format

By default, HybridPipe.io uses JSON for serialization. This provides good compatibility and readability, but may not be the most efficient format for all use cases.

## Changing the Serialization Format

You can change the serialization format by setting the `SerializationFormat` field in the configuration:

```go
config := &core.Config{
    SerializationFormat: core.SerializationFormatProtobuf,
}
```

The following serialization formats are available:

```go
const (
    SerializationFormatJSON       = "json"
    SerializationFormatProtobuf   = "protobuf"
    SerializationFormatMessagePack = "msgpack"
    SerializationFormatCBOR       = "cbor"
    SerializationFormatBSON       = "bson"
)
```

## Protocol-Specific Serialization

Some protocols have their own serialization formats or requirements. HybridPipe.io handles these automatically, but you should be aware of the following:

- **NATS**: Supports JSON, Protocol Buffers, and MessagePack
- **Kafka**: Supports all formats
- **RabbitMQ**: Supports all formats
- **AMQP**: Supports all formats
- **MQTT**: Supports all formats, but JSON is recommended for interoperability
- **Qpid**: Supports all formats
- **NSQ**: Supports all formats
- **TCP**: Supports all formats
- **Redis**: Supports all formats
- **ZeroMQ**: Supports all formats
- **NetChan**: Supports all formats, with special handling for Go types

## Custom Serialization

You can implement custom serialization by providing your own serialization and deserialization functions:

```go
// Register a custom serialization format
core.RegisterSerializationFormat("custom", customSerialize, customDeserialize)

// Use the custom serialization format
config := &core.Config{
    SerializationFormat: "custom",
}
```

The serialization and deserialization functions must have the following signatures:

```go
func customSerialize(v interface{}) ([]byte, error)
func customDeserialize(data []byte, v interface{}) error
```

## Performance Considerations

The choice of serialization format can have a significant impact on performance. Here are some general guidelines:

- **JSON**: Good for readability and debugging, but less efficient than binary formats
- **Protocol Buffers**: Efficient for both size and speed, but requires schema definitions
- **MessagePack**: More compact than JSON and faster to serialize/deserialize
- **CBOR**: Similar to MessagePack, but with some additional features
- **BSON**: Good for MongoDB compatibility, but less efficient than other binary formats

## Size Comparison

Here's a rough comparison of the size of serialized data for different formats:

| Format | Relative Size |
|--------|---------------|
| JSON | 100% |
| Protocol Buffers | 60-80% |
| MessagePack | 70-90% |
| CBOR | 70-90% |
| BSON | 80-100% |

## Speed Comparison

Here's a rough comparison of the serialization/deserialization speed for different formats:

| Format | Relative Speed |
|--------|---------------|
| JSON | 100% |
| Protocol Buffers | 200-300% |
| MessagePack | 150-250% |
| CBOR | 150-250% |
| BSON | 120-200% |

## Examples

### JSON Serialization

```go
// Use JSON serialization (default)
config := &core.Config{
    SerializationFormat: core.SerializationFormatJSON,
}

// Serialize data
data := map[string]interface{}{
    "name": "John",
    "age": 30,
}
serialized, err := core.Encode(data)
if err != nil {
    log.Fatalf("Failed to serialize data: %v", err)
}

// Deserialize data
var result map[string]interface{}
if err := core.Decode(serialized, &result); err != nil {
    log.Fatalf("Failed to deserialize data: %v", err)
}
```

### Protocol Buffers Serialization

```go
// Use Protocol Buffers serialization
config := &core.Config{
    SerializationFormat: core.SerializationFormatProtobuf,
}

// Define a Protocol Buffers message
type Person struct {
    Name string `protobuf:"bytes,1,opt,name=name,proto3"`
    Age  int32  `protobuf:"varint,2,opt,name=age,proto3"`
}

// Serialize data
person := &Person{
    Name: "John",
    Age:  30,
}
serialized, err := core.Encode(person)
if err != nil {
    log.Fatalf("Failed to serialize data: %v", err)
}

// Deserialize data
var result Person
if err := core.Decode(serialized, &result); err != nil {
    log.Fatalf("Failed to deserialize data: %v", err)
}
```

### MessagePack Serialization

```go
// Use MessagePack serialization
config := &core.Config{
    SerializationFormat: core.SerializationFormatMessagePack,
}

// Serialize data
data := map[string]interface{}{
    "name": "John",
    "age": 30,
}
serialized, err := core.Encode(data)
if err != nil {
    log.Fatalf("Failed to serialize data: %v", err)
}

// Deserialize data
var result map[string]interface{}
if err := core.Decode(serialized, &result); err != nil {
    log.Fatalf("Failed to deserialize data: %v", err)
}
```

## Interoperability

When using HybridPipe.io to communicate with systems that are not using HybridPipe.io, you should consider the serialization format carefully. Here are some recommendations:

- **JSON**: Best for interoperability with web services and JavaScript applications
- **Protocol Buffers**: Good for interoperability with Google services and systems that support Protocol Buffers
- **MessagePack**: Good for interoperability with systems that support MessagePack
- **CBOR**: Good for interoperability with systems that support CBOR
- **BSON**: Good for interoperability with MongoDB

## Troubleshooting

If you're having issues with serialization, here are some common problems and solutions:

- **Type mismatch**: Make sure the type you're deserializing to matches the serialized data
- **Missing fields**: Some serialization formats (like Protocol Buffers) handle missing fields differently than others
- **Circular references**: JSON and some other formats don't handle circular references well
- **Performance issues**: If serialization is a bottleneck, consider using a more efficient format like Protocol Buffers or MessagePack
