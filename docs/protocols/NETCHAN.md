# NetChan Protocol

## Overview

NetChan is a protocol that implements Go channel capability over a network between different processes. It provides a simple and intuitive API for sending and receiving messages between processes using Go's native channel operators (`<-` and `->`).

## Key Features

- **Go Channel Semantics**: Use familiar Go channel operators for network communication
- **Type Safety**: Send and receive strongly-typed messages
- **Automatic Serialization**: Messages are automatically serialized and deserialized
- **Buffered Channels**: Configure buffer sizes to control message flow
- **Reconnection**: Automatic reconnection on network failures
- **Timeout Control**: Configure timeouts for send and receive operations

## Implementation in HybridPipe

NetChan is implemented as a standalone library in the HybridPipe system, with integration into the HybridPipe interface.

### Go Implementation

The Go implementation provides a full implementation of the NetChan protocol, with support for Go's native channel operators.

### Rust Implementation

The Rust implementation provides a similar API to the Go implementation, with adaptations for Rust's async/await syntax.

## Configuration

### Go Implementation

```go
type Config struct {
    // Network is the network type (e.g., "tcp", "udp")
    Network string
    // Address is the address to listen on or connect to
    Address string
    // BufferSize is the size of the channel buffer
    BufferSize int
    // Timeout is the timeout for network operations in milliseconds
    Timeout int
    // KeepAlive enables keep-alive for TCP connections
    KeepAlive bool
    // KeepAlivePeriod is the keep-alive period in seconds
    KeepAlivePeriod int
    // ReconnectDelay is the delay between reconnection attempts in milliseconds
    ReconnectDelay int
    // MaxReconnectAttempts is the maximum number of reconnection attempts
    MaxReconnectAttempts int
}
```

### Rust Implementation

```rust
pub struct NetChanConfig {
    // Network address to listen on or connect to (e.g., "127.0.0.1:9000")
    pub address: String,
    // Channel buffer size
    pub buffer_size: usize,
    // Timeout for network operations in milliseconds
    pub timeout: u64,
    // Reconnect interval in milliseconds
    pub reconnect_interval: u64,
    // Maximum reconnect attempts (0 = infinite)
    pub max_reconnect_attempts: u32,
}
```

## Usage

### Go Implementation

#### Server

```go
// Create a server with default configuration
server, err := netchan.NewServer(nil)
if err != nil {
    log.Fatalf("Failed to create server: %v", err)
}
defer server.Close()

// Create a channel
ch := server.Chan("example")

// Receive a message
value := ch.RecvOp()
fmt.Printf("Received: %v\n", value)

// Send a response
ch.SendOp("Response")
```

#### Client

```go
// Create a client with default configuration
config := netchan.DefaultConfig()
client, err := netchan.NewClient(config)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
defer client.Close()

// Create a channel
ch := client.Chan("example")

// Send a message
ch.SendOp("Hello, NetChan!")

// Receive a response
response := ch.RecvOp()
fmt.Printf("Received response: %v\n", response)
```

### Rust Implementation

```rust
// Create a NetChan with default configuration
let netchan = NetChan::new_default().await?;

// Create a channel
let ch = netchan.chan("example");

// Send a message
ch.send(Box::new("Hello, NetChan!".to_string())).await?;

// Receive a response
let response = ch.receive().await?;
println!("Received response: {:?}", response);
```

## Integration with HybridPipe

NetChan is fully integrated with the HybridPipe system, allowing you to use it as a messaging protocol alongside other protocols like NATS, Kafka, and RabbitMQ.

```go
// Create a NetChan router
config := &core.NetChanConfig{
    ChannelBufferSize: 10,
    SendTimeout:       1000,
}
router := netchan.New(config)

// Connect to the NetChan system
if err := router.Connect(); err != nil {
    log.Fatalf("Failed to connect to NetChan system: %v", err)
}
defer router.Close()

// Subscribe to a pipe
if err := router.Subscribe("example", func(data []byte) error {
    var message string
    if err := core.Decode(data, &message); err != nil {
        return err
    }
    fmt.Printf("Received: %s\n", message)
    return nil
}); err != nil {
    log.Fatalf("Failed to subscribe: %v", err)
}

// Dispatch a message
if err := router.Dispatch("example", "Hello, NetChan!"); err != nil {
    log.Fatalf("Failed to dispatch message: %v", err)
}
```

## Performance Considerations

NetChan is designed for ease of use rather than maximum performance. For high-throughput scenarios, consider using a more specialized protocol like ZeroMQ or NATS.

## Limitations

1. **Type Safety**: NetChan provides type safety at compile time for the Go implementation, but not for the Rust implementation
2. **Performance**: NetChan may not be as performant as specialized messaging protocols
3. **Scalability**: NetChan is designed for communication between a small number of processes, not for large-scale distributed systems

## Examples

See the [examples](../../golang/netchan/example) directory for more examples of using NetChan with HybridPipe.io.

## References

- [Go Channels Documentation](https://tour.golang.org/concurrency/2)
- [Rust Channels Documentation](https://doc.rust-lang.org/std/sync/mpsc/)
