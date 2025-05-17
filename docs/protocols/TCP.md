# TCP/IP Protocol

## Overview

TCP/IP (Transmission Control Protocol/Internet Protocol) is the basic communication protocol of the Internet. It provides reliable, ordered, and error-checked delivery of data between applications running on hosts communicating over an IP network.

## Key Features

- **Reliable**: Guarantees delivery of data
- **Ordered**: Data is delivered in the order it was sent
- **Error-Checked**: Includes checksums to verify data integrity
- **Flow Control**: Prevents overwhelming receivers with too much data
- **Congestion Control**: Adapts to network congestion
- **Connection-Oriented**: Establishes a connection before sending data
- **Full-Duplex**: Allows simultaneous bidirectional data transfer

## Implementation in HybridPipe

HybridPipe implements a direct TCP/IP communication protocol, providing a low-level messaging system without any additional protocol overhead.

### Go Implementation

The Go implementation uses the standard library's `net` package for TCP/IP communication.

### Rust Implementation

The Rust implementation uses the Tokio runtime's `tokio::net` module for asynchronous TCP/IP communication.

## Configuration

### Go Implementation

```go
type TCPConfig struct {
    // Address is the TCP address to listen on or connect to (e.g., "localhost:8080")
    Address string
    // ServerMode indicates whether to run in server mode (true) or client mode (false)
    ServerMode bool
    // BufferSize is the size of the read/write buffer
    BufferSize int
    // KeepAlive enables keep-alive for TCP connections
    KeepAlive bool
    // KeepAlivePeriod is the keep-alive period in seconds
    KeepAlivePeriod int
    // ReadTimeout is the read timeout in milliseconds
    ReadTimeout int
    // WriteTimeout is the write timeout in milliseconds
    WriteTimeout int
    // DialTimeout is the dial timeout in milliseconds
    DialTimeout int
    // ReconnectDelay is the delay between reconnection attempts in milliseconds
    ReconnectDelay int
    // MaxReconnectAttempts is the maximum number of reconnection attempts
    MaxReconnectAttempts int
}
```

### Rust Implementation

```rust
pub struct TCPConfig {
    // Address is the TCP address to listen on or connect to (e.g., "localhost:8080")
    pub address: String,
    // Server mode indicates whether to run in server mode (true) or client mode (false)
    pub server_mode: bool,
    // Buffer size is the size of the read/write buffer
    pub buffer_size: usize,
    // Keep alive enables keep-alive for TCP connections
    pub keep_alive: bool,
    // Keep alive period is the keep-alive period in seconds
    pub keep_alive_period: u64,
    // Read timeout is the read timeout in milliseconds
    pub read_timeout: u64,
    // Write timeout is the write timeout in milliseconds
    pub write_timeout: u64,
    // Dial timeout is the dial timeout in milliseconds
    pub dial_timeout: u64,
    // Reconnect delay is the delay between reconnection attempts in milliseconds
    pub reconnect_delay: u64,
    // Max reconnect attempts is the maximum number of reconnection attempts
    pub max_reconnect_attempts: u32,
}
```

## Usage

### Go Implementation

```go
// Create a TCP router with default configuration
router := tcp.New(nil)

// Connect to TCP
if err := router.Connect(); err != nil {
    log.Fatalf("Failed to connect to TCP: %v", err)
}
defer router.Close()

// Subscribe to a pipe
if err := router.Subscribe("greetings", func(data []byte) error {
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
if err := router.Dispatch("greetings", "Hello, TCP!"); err != nil {
    log.Fatalf("Failed to dispatch message: %v", err)
}
```

### Rust Implementation

```rust
// Deploy a TCP router
let router = deploy_router(BrokerType::TCP)?;

// Connect to TCP
router.connect().await?;

// Subscribe to a pipe
let callback: Process = Box::new(|data| {
    Box::pin(async move {
        info!("Received message: {:?}", data);
        Ok(())
    })
});
router.subscribe("greetings", callback).await?;

// Dispatch a message
let message = "Hello, TCP!".as_bytes().to_vec();
router.dispatch("greetings", Box::new(message)).await?;
```

## Message Format

The TCP/IP implementation in HybridPipe uses a simple message format:

1. **Header**: 8 bytes
   - Message length (4 bytes)
   - Pipe name length (2 bytes)
   - Flags (2 bytes)
2. **Pipe Name**: Variable length
3. **Payload**: Variable length

This format allows for efficient parsing and routing of messages.

## Server and Client Modes

The TCP/IP implementation in HybridPipe can operate in two modes:

1. **Server Mode**: Listens for incoming connections and accepts them
2. **Client Mode**: Connects to a server

In server mode, the router listens on the specified address and accepts incoming connections. In client mode, the router connects to the specified address.

## Performance Considerations

TCP/IP is a low-level protocol with minimal overhead. Here are some tips for optimizing TCP/IP performance:

1. **Buffer Size**: Set an appropriate buffer size for your use case
2. **Nagle's Algorithm**: Disable Nagle's algorithm for low-latency applications
3. **Keep-Alive**: Enable keep-alive to detect disconnections
4. **Timeouts**: Set appropriate timeouts for your use case

## Limitations

1. **No Built-in Security**: TCP/IP does not provide built-in security features like authentication and encryption. You need to implement these yourself or use TLS
2. **No Message Routing**: TCP/IP does not provide built-in message routing. HybridPipe implements this on top of TCP/IP
3. **Connection Management**: TCP/IP requires careful connection management to handle disconnections and reconnections

## Examples

See the [examples](../../golang/protocols/tcp/example) directory for more examples of using TCP/IP with HybridPipe.io.

## References

- [TCP/IP Protocol Suite](https://en.wikipedia.org/wiki/Internet_protocol_suite)
- [Go net Package](https://golang.org/pkg/net/)
- [Rust tokio::net Module](https://docs.rs/tokio/latest/tokio/net/)
