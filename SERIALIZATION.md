# HybridPipe Language-Agnostic Serialization

This document provides detailed information about HybridPipe's language-agnostic serialization system.

## Table of Contents

- [Overview](#overview)
- [Supported Formats](#supported-formats)
- [Wire Format](#wire-format)
- [Using Serialization in Go](#using-serialization-in-go)
- [Cross-Language Support](#cross-language-support)
  - [Python](#python)
  - [JavaScript/Node.js](#javascriptnodejs)
  - [Java](#java)
- [Performance Considerations](#performance-considerations)
- [Backward Compatibility](#backward-compatibility)
- [Examples](#examples)

## Overview

HybridPipe's serialization system is designed to enable seamless communication between applications written in different programming languages. It provides a consistent wire format that can be encoded and decoded by any supported language.

Key features:

- Multiple serialization formats (JSON, Protocol Buffers, MessagePack, GOB)
- Automatic format detection and negotiation
- Compression for large messages
- Backward compatibility with existing HybridPipe applications
- Client libraries for multiple languages

## Supported Formats

| Format | Description | Pros | Cons | Best For |
|--------|-------------|------|------|----------|
| **JSON** | Text-based, human-readable format | Human-readable, universal support | Larger size, slower | Debugging, web integration |
| **Protocol Buffers** | Binary format with schema | Compact, fast, schema validation | Requires schema definition | High-performance, strict typing |
| **MessagePack** | Binary JSON alternative | Compact, fast, schema-less | Less human-readable | Efficient JSON alternative |
| **GOB** | Go's native binary format | Efficient for Go types | Go-only | Go-to-Go communication |

## Wire Format

HybridPipe uses a standardized wire format for all messages:

```
[Header (9 bytes)][Payload (variable)]
```

The header consists of:
- Magic number (2 bytes): 0x4850 ("HP" in ASCII)
- Version (1 byte): Current version is 1
- Format (1 byte): Indicates the serialization format used
- Flags (1 byte): Additional information (e.g., compression)
- Payload length (4 bytes): Length of the payload in bytes

This format allows for:
- Format detection and negotiation
- Backward compatibility
- Future extensions

## Using Serialization in Go

### Basic Usage

```go
import hp "github.com/AnandSGit/hybridpipe.io"

// Register custom types (only needed for GOB encoding)
hp.Enable(MyCustomType{})

// Serialize data using the default format (GOB for backward compatibility)
data := MyCustomType{Field1: "value", Field2: 42}
bytes, err := hp.Encode(data)
if err != nil {
    log.Fatalf("Failed to serialize: %v", err)
}

// Deserialize data
var result MyCustomType
err = hp.Decode(bytes, &result)
if err != nil {
    log.Fatalf("Failed to deserialize: %v", err)
}
```

### Using Different Formats

```go
// Create serialization options for JSON
jsonOptions := hp.SerializationOptions{
    Format: hp.FormatJSON,
    Compression: true,
}

// Serialize data using JSON
data := map[string]interface{}{
    "name": "John Doe",
    "age": 30,
    "items": []string{"phone", "laptop"},
}

jsonBytes, err := hp.EncodeWithOptions(data, jsonOptions)
if err != nil {
    log.Fatalf("Failed to serialize: %v", err)
}

// Deserialize JSON data
var result map[string]interface{}
err = hp.DecodeWithOptions(jsonBytes, &result)
if err != nil {
    log.Fatalf("Failed to deserialize: %v", err)
}
```

### Using Protocol Buffers

```go
import (
    hp "github.com/AnandSGit/hybridpipe.io"
    "github.com/AnandSGit/hybridpipe.io/proto"
)

// Create a Protocol Buffer message
message := &proto.HybridMessage{
    Header: &proto.MessageHeader{
        MessageId: "msg-123",
        Source: "service-a",
        Destination: "service-b",
        Timestamp: time.Now().UnixNano() / 1000000,
    },
    Payload: &proto.HybridMessage_StringMessage{
        StringMessage: &proto.StringMessage{
            Value: "Hello, world!",
        },
    },
}

// Create serialization options for Protocol Buffers
pbOptions := hp.SerializationOptions{
    Format: hp.FormatProtobuf,
}

// Serialize the message
bytes, err := hp.EncodeWithOptions(message, pbOptions)
if err != nil {
    log.Fatalf("Failed to serialize: %v", err)
}

// Deserialize the message
var result proto.HybridMessage
err = hp.DecodeWithOptions(bytes, &result)
if err != nil {
    log.Fatalf("Failed to deserialize: %v", err)
}
```

## Cross-Language Support

HybridPipe provides client libraries for multiple languages that use the same wire format.

### Python

```python
from hybridpipe_client import HybridPipeClient, SerializationFormat

# Create a client
client = HybridPipeClient("kafka", {
    "bootstrap_servers": "localhost:9092"
})

# Set serialization format to JSON
client.set_serialization_format(SerializationFormat.JSON)

# Enable compression for large messages
client.enable_compression(True)

# Send a message
client.dispatch("example-topic", {
    "message": "Hello from Python!",
    "timestamp": time.time()
})

# Define a message handler
def handle_message(data):
    print(f"Received message: {data}")

# Subscribe to a topic
client.accept("example-topic", handle_message)
```

### JavaScript/Node.js

```javascript
const { HybridPipeClient, SerializationFormat } = require('hybridpipe-client');

// Create a client
const client = new HybridPipeClient('kafka', {
    brokers: ['localhost:9092'],
    clientId: 'hybridpipe-example'
});

// Set serialization format to JSON
client.setSerializationFormat(SerializationFormat.JSON);

// Enable compression for large messages
client.enableCompression(true);

// Send a message
await client.dispatch('example-topic', {
    message: 'Hello from Node.js!',
    timestamp: Date.now()
});

// Subscribe to a topic
await client.accept('example-topic', (data) => {
    console.log(`Received message: ${JSON.stringify(data, null, 2)}`);
});
```

### Java

```java
import io.hybridpipe.client.HybridPipeClient;
import io.hybridpipe.client.HybridPipeClient.BrokerType;
import io.hybridpipe.client.HybridPipeClient.SerializationFormat;

import java.util.HashMap;
import java.util.Map;

// Create options
Map<String, Object> options = new HashMap<>();
options.put("bootstrapServers", "localhost:9092");
options.put("clientId", "hybridpipe-example");

// Create a client
try (HybridPipeClient client = new HybridPipeClient(BrokerType.KAFKA, options)) {
    // Set serialization format to JSON
    client.setSerializationFormat(SerializationFormat.JSON);

    // Enable compression for large messages
    client.enableCompression(true, 6);

    // Define a message class
    class Message {
        private String text;
        private long timestamp;

        // Getters and setters...
    }

    // Subscribe to a topic
    client.accept("example-topic", (Message message) -> {
        System.out.println("Received message: " + message);
    }, Message.class);

    // Send a message
    client.dispatch("example-topic", new Message("Hello from Java!", System.currentTimeMillis()));
}
```

## Performance Considerations

Performance varies by serialization format:

| Format | Size | Serialization Speed | Deserialization Speed |
|--------|------|---------------------|------------------------|
| JSON | Largest | Slowest | Slow |
| Protocol Buffers | Smallest | Fast | Fastest |
| MessagePack | Small | Fast | Fast |
| GOB | Medium | Fast (Go only) | Fast (Go only) |

Recommendations:

- Use **Protocol Buffers** for high-performance, cross-language communication
- Use **JSON** for debugging or when human-readability is important
- Use **MessagePack** as a good balance between performance and flexibility
- Use **GOB** only for Go-to-Go communication where backward compatibility is needed

## Backward Compatibility

HybridPipe maintains backward compatibility with existing applications:

- The default serialization format is still GOB for backward compatibility
- Messages without the HybridPipe header are assumed to be in GOB format
- The `Encode` and `Decode` functions maintain the same API as before
- New applications can opt-in to the new serialization formats using `EncodeWithOptions` and `DecodeWithOptions`

## Examples

Complete examples for each language can be found in the `examples/multilang` directory:

- Go: `examples/multilang/go/`
- Python: `examples/multilang/python/`
- JavaScript/Node.js: `examples/multilang/nodejs/`
- Java: `examples/multilang/java/`

These examples demonstrate how to use HybridPipe's language-agnostic serialization to communicate between applications written in different languages.
