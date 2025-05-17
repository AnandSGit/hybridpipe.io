# Apache Qpid Protocol

## Overview

Apache Qpid is an open-source messaging system that implements the Advanced Message Queuing Protocol (AMQP). It provides a reliable and secure messaging infrastructure with enterprise features like transactions, routing, and security.

## Key Features

- **AMQP Compliance**: Implements the AMQP 1.0 standard
- **Cross-Platform**: Runs on major operating systems
- **Multiple Language Bindings**: Supports Java, C++, Python, and more
- **Enterprise Features**: Transactions, routing, security, and more
- **High Performance**: Designed for high throughput and low latency
- **Reliability**: Message persistence and acknowledgment

## Implementation in HybridPipe

HybridPipe implements the Apache Qpid protocol using the AMQP 1.0 standard, providing a seamless integration with Qpid brokers.

### Go Implementation

The Go implementation uses the [github.com/Azure/go-amqp](https://github.com/Azure/go-amqp) package, which provides Go bindings for AMQP 1.0.

### Rust Implementation

The Rust implementation uses the [amqp](https://crates.io/crates/amqp) crate, which provides Rust bindings for AMQP 1.0.

## Configuration

### Go Implementation

```go
type QpidConfig struct {
    // URL is the AMQP URL (e.g., "amqp://guest:guest@localhost:5672")
    URL string
    // ContainerID is the AMQP container ID
    ContainerID string
    // MaxFrameSize is the maximum frame size
    MaxFrameSize uint32
    // ChannelMax is the maximum number of channels
    ChannelMax uint16
    // IdleTimeout is the idle timeout in milliseconds
    IdleTimeout int
    // ReconnectDelay is the delay between reconnection attempts in milliseconds
    ReconnectDelay int
    // MaxReconnectAttempts is the maximum number of reconnection attempts
    MaxReconnectAttempts int
}
```

### Rust Implementation

```rust
pub struct QpidConfig {
    // URL is the AMQP URL (e.g., "amqp://guest:guest@localhost:5672")
    pub url: String,
    // Container ID is the AMQP container ID
    pub container_id: String,
    // Max frame size is the maximum frame size
    pub max_frame_size: u32,
    // Channel max is the maximum number of channels
    pub channel_max: u16,
    // Idle timeout is the idle timeout in milliseconds
    pub idle_timeout: u64,
    // Reconnect delay is the delay between reconnection attempts in milliseconds
    pub reconnect_delay: u64,
    // Max reconnect attempts is the maximum number of reconnection attempts
    pub max_reconnect_attempts: u32,
}
```

## Usage

### Go Implementation

```go
// Create a Qpid router with default configuration
router := qpid.New(nil)

// Connect to Qpid
if err := router.Connect(); err != nil {
    log.Fatalf("Failed to connect to Qpid: %v", err)
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
if err := router.Dispatch("greetings", "Hello, Qpid!"); err != nil {
    log.Fatalf("Failed to dispatch message: %v", err)
}
```

### Rust Implementation

```rust
// Deploy a Qpid router
let router = deploy_router(BrokerType::QPID)?;

// Connect to Qpid
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
let message = "Hello, Qpid!".as_bytes().to_vec();
router.dispatch("greetings", Box::new(message)).await?;
```

## Performance Considerations

Apache Qpid is designed for enterprise messaging with a focus on reliability and features. Here are some tips for optimizing Qpid performance:

1. **Message Size**: Smaller messages generally perform better than larger ones
2. **Acknowledgment Mode**: Choose the appropriate acknowledgment mode for your use case
3. **Prefetch Size**: Set an appropriate prefetch size to control the flow of messages
4. **Transactions**: Use transactions only when necessary, as they add overhead

## Limitations

1. **Complexity**: Qpid is a complex system with many features, which can make it harder to configure and use
2. **Resource Usage**: Qpid can use more resources than simpler messaging systems
3. **Performance**: Qpid may not be as performant as specialized messaging systems like ZeroMQ or NATS

## Examples

See the [examples](../../golang/protocols/qpid/example) directory for more examples of using Apache Qpid with HybridPipe.io.

## References

- [Apache Qpid Official Website](https://qpid.apache.org/)
- [AMQP 1.0 Specification](https://www.amqp.org/resources/specifications)
- [Go AMQP Binding](https://github.com/Azure/go-amqp)
- [Rust AMQP Binding](https://crates.io/crates/amqp)
