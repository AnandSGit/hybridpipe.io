# NetChan: Network Channels for Go

*Version: 1.0.0 (April 19, 2025)*

## Overview

NetChan is a Go library that extends the concept of Go channels across network boundaries, allowing different processes to communicate using the familiar channel semantics. It provides a simple, type-safe API that closely mirrors Go's native channels while handling all the complexities of network communication under the hood.

NetChan is fully integrated with the HybridPipe messaging system, providing a consistent interface for inter-process communication alongside other messaging protocols like Kafka, NATS, RabbitMQ, and more.

## Table of Contents

- [Concepts](#concepts)
- [Architecture](#architecture)
- [API Reference](#api-reference)
- [HybridPipe Integration](#hybridpipe-integration)
- [Protocol Specification](#protocol-specification)
- [Security Considerations](#security-considerations)
- [Performance Characteristics](#performance-characteristics)
- [Error Handling](#error-handling)
- [Examples](#examples)
- [Testing](#testing)
- [Future Enhancements](#future-enhancements)

## Concepts

### Network Channels

NetChan implements the concept of "network channels" - channels that can be used to send and receive data between different processes over a network. These channels maintain the same semantics as Go's native channels:

- Send and receive operations
- Buffered and unbuffered channels
- Channel closing
- Range over channels
- Select statements

### Endpoints

NetChan uses the concept of endpoints to represent the two sides of a channel connection:

- **Sender Endpoint**: Used to send data over the network channel
- **Receiver Endpoint**: Used to receive data from the network channel

Each endpoint has a unique identifier that allows it to be discovered and connected to by other endpoints.

### Serialization

NetChan automatically handles serialization and deserialization of data sent over the network. It supports:

- All basic Go types
- Structs with exported fields
- Maps and slices
- Custom types that implement the `encoding.BinaryMarshaler` and `encoding.BinaryUnmarshaler` interfaces

## Architecture

NetChan is built on a layered architecture:

1. **API Layer**: Provides the user-facing API that mimics Go's channel operations
2. **Protocol Layer**: Handles the NetChan protocol, including message framing, type information, and control messages
3. **Transport Layer**: Manages the underlying network connections (TCP/IP by default)
4. **Serialization Layer**: Handles encoding and decoding of Go values

### Component Diagram

```
┌─────────────────────────────────────────┐
│               Application               │
└───────────────────┬─────────────────────┘
                    │
┌───────────────────▼─────────────────────┐
│              NetChan API                │
│  (Chan, Send, Recv, Close, Select)      │
└───────────────────┬─────────────────────┘
                    │
┌───────────────────▼─────────────────────┐
│           Protocol Handler               │
│  (Message framing, control messages)     │
└───────────────────┬─────────────────────┘
                    │
┌───────────────────▼─────────────────────┐
│           Transport Layer                │
│  (TCP connections, reconnection logic)   │
└───────────────────┬─────────────────────┘
                    │
┌───────────────────▼─────────────────────┐
│         Serialization Layer              │
│  (Encoding/decoding of Go values)        │
└─────────────────────────────────────────┘
```

### Connection Establishment

1. The receiver endpoint binds to a TCP port and listens for incoming connections
2. The sender endpoint connects to the receiver's address
3. The endpoints exchange handshake messages to verify compatibility
4. Once the handshake is complete, the channel is established and ready for use

### Message Flow

1. When a value is sent on a NetChan:
   - The value is serialized
   - A message header is prepended (includes type information, sequence number, etc.)
   - The message is sent over the network connection

2. When a value is received from a NetChan:
   - The message header is parsed
   - The message body is deserialized into the appropriate Go type
   - The value is returned to the receiver

## API Reference

### Creating Channels

```go
// Create a new unbuffered channel
ch := netchan.Make(T{}) // T is the type of values to be sent

// Create a new buffered channel with capacity
ch := netchan.MakeBuffered(T{}, capacity)

// Listen for incoming channel connections
receiver := netchan.Listen[T](address)

// Connect to a channel
sender := netchan.Dial[T](address)
```

### Channel Operations

```go
// Send a value (blocks if the channel is full)
err := ch.Send(value)

// Try to send a value (non-blocking)
ok, err := ch.TrySend(value)

// Receive a value (blocks if the channel is empty)
value, ok, err := ch.Recv()

// Try to receive a value (non-blocking)
value, ok, err := ch.TryRecv()

// Close a channel
err := ch.Close()

// Check if a channel is closed
closed := ch.IsClosed()
```

### Select Operations

```go
// Create a new select case for sending
sendCase := netchan.SendCase(ch, value)

// Create a new select case for receiving
recvCase := netchan.RecvCase(ch)

// Create a new select case for a timeout
timeoutCase := netchan.TimeoutCase(duration)

// Perform a select operation
chosen, recv, recvOK, err := netchan.Select([]netchan.Case{sendCase, recvCase, timeoutCase})
```

### Configuration

```go
// Configure a channel with options
ch := netchan.Make(T{}, netchan.WithBufferSize(10), netchan.WithTimeout(5*time.Second))

// Available options
netchan.WithBufferSize(n)           // Set the channel buffer size
netchan.WithTimeout(duration)       // Set operation timeout
netchan.WithReconnect(true)         // Enable automatic reconnection
netchan.WithReconnectBackoff(policy) // Set reconnection backoff policy
netchan.WithCompression(level)      // Enable compression
netchan.WithTLS(config)             // Enable TLS encryption
```

## HybridPipe Integration

NetChan is fully integrated with the HybridPipe messaging system, allowing you to use it alongside other messaging protocols with a consistent interface.

### Using NetChan through HybridPipe

```go
import (
    "fmt"
    "log"

    hp "github.com/AnandSGit/hybridpipe.io"
)

func main() {
    // Create a NetChan router through HybridPipe
    router, err := hp.DeployRouter(hp.NETCHAN)
    if err != nil {
        log.Fatalf("Failed to deploy NetChan router: %v", err)
    }
    defer router.Close()

    // Subscribe to a channel
    router.Accept("my-channel", func(data interface{}) {
        fmt.Printf("Received: %v\n", data)
    })

    // Send a message
    router.Dispatch("my-channel", "Hello, NetChan!")
}
```

### Configuration

NetChan can be configured through the HybridPipe configuration file:

```toml
[NETCHAN]
NetChanServer = "localhost"
NetChanPort = 8080
NetChanBufferSize = 10
NetChanTimeout = 5000
```

### Benefits of HybridPipe Integration

1. **Consistent API**: Use the same API for NetChan as for other messaging systems
2. **Easy Migration**: Switch between messaging systems without changing application code
3. **Centralized Configuration**: Configure all messaging systems in one place
4. **Combined Functionality**: Use NetChan alongside other messaging systems in the same application

## Protocol Specification

NetChan uses a binary protocol for efficiency. The protocol consists of:

### Message Types

- `HANDSHAKE`: Initial connection setup
- `DATA`: Contains a serialized value
- `CLOSE`: Indicates channel closure
- `ACK`: Acknowledges receipt of a message
- `PING`: Connection keepalive
- `PONG`: Response to a ping

### Message Format

Each message has the following format:

```
┌─────────────┬────────────┬────────────┬────────────┬────────────┐
│ Message Type│ Sequence # │ Flags      │ Length     │ Payload    │
│ (1 byte)    │ (4 bytes)  │ (1 byte)   │ (4 bytes)  │ (variable) │
└─────────────┴────────────┴────────────┴────────────┴────────────┘
```

- **Message Type**: Identifies the type of message
- **Sequence Number**: Used for ordering and acknowledgment
- **Flags**: Additional message metadata
- **Length**: Length of the payload in bytes
- **Payload**: The serialized data (for DATA messages) or control information

### Handshake Process

1. Sender connects to receiver
2. Sender sends `HANDSHAKE` message with protocol version and type information
3. Receiver validates the handshake and responds with its own `HANDSHAKE` message
4. If both handshakes are successful, the channel is established

## Security Considerations

### Authentication

NetChan supports TLS for authentication and encryption. Users can provide a TLS configuration when creating channels:

```go
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    RootCAs:      certPool,
}

ch := netchan.Make(T{}, netchan.WithTLS(tlsConfig))
```

### Authorization

NetChan does not implement application-level authorization. It is recommended to implement authorization at the application level or use network-level controls (firewalls, VPNs, etc.).

### Encryption

All data sent over NetChan can be encrypted using TLS. When TLS is enabled, all communication, including the handshake and data messages, is encrypted.

## Performance Characteristics

### Latency

NetChan adds some overhead compared to local channels due to:
- Serialization/deserialization
- Network latency
- Protocol processing

Typical latency overhead is in the range of 0.1-1ms on a local network, depending on the size and complexity of the data being sent.

### Throughput

NetChan is designed to be efficient for both small and large messages:
- Small messages (< 1KB): ~10,000-50,000 messages/second
- Large messages (> 1MB): Limited by network bandwidth

### Resource Usage

- Each NetChan connection uses one TCP connection
- Memory usage depends on the buffer size and the size of messages in transit
- CPU usage is primarily affected by serialization/deserialization operations

## Error Handling

NetChan provides detailed error information for all operations. Common error types include:

- `ConnectionError`: Problems with the underlying network connection
- `TimeoutError`: Operation timed out
- `SerializationError`: Failed to serialize or deserialize a value
- `ClosedError`: Attempted to use a closed channel
- `HandshakeError`: Failed to establish a channel connection

All errors implement the standard Go `error` interface and provide additional context through error wrapping.

## Examples

### Basic Usage

```go
// Receiver
func main() {
    // Create a receiver for int values
    receiver := netchan.Listen[int](":8080")
    defer receiver.Close()

    // Receive values
    for {
        value, ok, err := receiver.Recv()
        if err != nil {
            log.Printf("Error: %v", err)
            continue
        }
        if !ok {
            log.Println("Channel closed")
            break
        }
        log.Printf("Received: %d", value)
    }
}

// Sender
func main() {
    // Connect to the receiver
    sender := netchan.Dial[int]("localhost:8080")
    defer sender.Close()

    // Send values
    for i := 0; i < 10; i++ {
        if err := sender.Send(i); err != nil {
            log.Printf("Error: %v", err)
            continue
        }
        log.Printf("Sent: %d", i)
    }
}
```

### Using Select

```go
func main() {
    ch1 := netchan.Dial[int]("localhost:8080")
    ch2 := netchan.Dial[string]("localhost:8081")

    for {
        // Create select cases
        case1 := netchan.RecvCase(ch1)
        case2 := netchan.RecvCase(ch2)
        timeout := netchan.TimeoutCase(5 * time.Second)

        // Perform select
        chosen, recv, recvOK, err := netchan.Select([]netchan.Case{case1, case2, timeout})

        if err != nil {
            log.Printf("Select error: %v", err)
            continue
        }

        switch chosen {
        case 0: // ch1
            if recvOK {
                value := recv.(int)
                log.Printf("Received from ch1: %d", value)
            } else {
                log.Println("ch1 closed")
            }
        case 1: // ch2
            if recvOK {
                value := recv.(string)
                log.Printf("Received from ch2: %s", value)
            } else {
                log.Println("ch2 closed")
            }
        case 2: // timeout
            log.Println("Select timed out")
        }
    }
}
```

## Testing

NetChan includes comprehensive tests:

- Unit tests for all components
- Integration tests for end-to-end functionality
- Performance benchmarks
- Fault injection tests for error handling

The test suite includes a mock transport layer that can be used to test NetChan without actual network connections.

## Future Enhancements

Planned future enhancements include:

- **Multi-receiver support**: Allow multiple receivers for the same channel
- **Channel groups**: Group related channels for easier management
- **Metrics and monitoring**: Built-in support for collecting performance metrics
- **Alternative transports**: Support for WebSockets, QUIC, and other transport protocols
- **Cross-language support**: Implementations for other languages to enable cross-language communication
- **Distributed select**: Allow select operations across multiple machines
