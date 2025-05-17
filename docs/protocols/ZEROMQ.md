# ZeroMQ Protocol

## Overview

ZeroMQ (also known as ØMQ, 0MQ, or zmq) is a high-performance asynchronous messaging library designed to be used in distributed or concurrent applications. It provides a message-oriented middleware that is lightweight and fast.

## Key Features

- **High Performance**: ZeroMQ is designed for high throughput and low latency
- **Asynchronous I/O**: Non-blocking operations for better resource utilization
- **Multiple Messaging Patterns**: Support for request-reply, publish-subscribe, push-pull, and exclusive pair patterns
- **Transport Agnostic**: Can use TCP, IPC, inproc, and other transports
- **No Central Broker**: Peer-to-peer architecture with no single point of failure
- **Language Agnostic**: Bindings available for many programming languages

## Implementation in HybridPipe

HybridPipe implements the Publish-Subscribe pattern for ZeroMQ, which is well-suited for broadcasting messages to multiple receivers.

### Go Implementation

The Go implementation uses the [github.com/pebbe/zmq4](https://github.com/pebbe/zmq4) package, which provides Go bindings for ZeroMQ.

### Rust Implementation

The Rust implementation uses the [zmq](https://crates.io/crates/zmq) crate, which provides Rust bindings for ZeroMQ.

## Configuration

### Go Implementation

```go
type ZeroMQConfig struct {
    // Publisher endpoint (e.g., "tcp://127.0.0.1:5555")
    PublisherEndpoint string
    // Subscriber endpoint (e.g., "tcp://127.0.0.1:5556")
    SubscriberEndpoint string
    // Publisher bind flag (true = bind, false = connect)
    PublisherBind bool
    // Subscriber bind flag (true = bind, false = connect)
    SubscriberBind bool
    // Publisher high water mark
    PublisherHWM int
    // Subscriber high water mark
    SubscriberHWM int
    // Publisher linger period in milliseconds
    PublisherLinger int
    // Subscriber linger period in milliseconds
    SubscriberLinger int
    // Reconnect interval in milliseconds
    ReconnectInterval int
    // Maximum reconnect attempts (0 = infinite)
    MaxReconnectAttempts int
}
```

### Rust Implementation

```rust
pub struct ZeroMQConfig {
    // Publisher endpoint (e.g., "tcp://127.0.0.1:5555")
    pub publisher_endpoint: String,
    // Subscriber endpoint (e.g., "tcp://127.0.0.1:5556")
    pub subscriber_endpoint: String,
    // Publisher bind flag (true = bind, false = connect)
    pub publisher_bind: bool,
    // Subscriber bind flag (true = bind, false = connect)
    pub subscriber_bind: bool,
    // Publisher high water mark
    pub publisher_hwm: i32,
    // Subscriber high water mark
    pub subscriber_hwm: i32,
    // Publisher linger period in milliseconds
    pub publisher_linger: i32,
    // Subscriber linger period in milliseconds
    pub subscriber_linger: i32,
    // Reconnect interval in milliseconds
    pub reconnect_interval: u64,
    // Maximum reconnect attempts (0 = infinite)
    pub max_reconnect_attempts: u32,
}
```

## Usage

### Go Implementation

```go
// Create a ZeroMQ router with default configuration
router := zeromq.New(nil)

// Connect to ZeroMQ
if err := router.Connect(); err != nil {
    log.Fatalf("Failed to connect to ZeroMQ: %v", err)
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
if err := router.Dispatch("greetings", "Hello, ZeroMQ!"); err != nil {
    log.Fatalf("Failed to dispatch message: %v", err)
}
```

### Rust Implementation

```rust
// Deploy a ZeroMQ router
let router = deploy_router(BrokerType::ZEROMQ)?;

// Connect to ZeroMQ
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
let message = "Hello, ZeroMQ!".as_bytes().to_vec();
router.dispatch("greetings", Box::new(message)).await?;
```

## Performance Considerations

ZeroMQ is designed for high performance and low latency. Here are some tips for optimizing ZeroMQ performance:

1. **Message Size**: Smaller messages generally perform better than larger ones
2. **High Water Mark**: Set appropriate HWM values to prevent memory issues
3. **Socket Types**: Choose the right socket type for your use case
4. **Transport Protocol**: TCP is reliable but has higher latency, IPC is faster but limited to the local machine, and inproc is the fastest but limited to the same process

## Limitations

1. **No Built-in Security**: ZeroMQ does not provide built-in security features like authentication and encryption. You need to implement these yourself or use additional libraries
2. **No Message Persistence**: ZeroMQ does not persist messages. If a subscriber is not connected when a message is published, it will miss the message
3. **No Message Acknowledgment**: In the Publish-Subscribe pattern, there is no built-in acknowledgment mechanism to confirm that a message has been received

## Examples

See the [examples](../../golang/protocols/zeromq/example) directory for more examples of using ZeroMQ with HybridPipe.io.

## References

- [ZeroMQ Official Website](https://zeromq.org/)
- [ZeroMQ Guide](https://zguide.zeromq.org/)
- [Go ZeroMQ Binding](https://github.com/pebbe/zmq4)
- [Rust ZeroMQ Binding](https://crates.io/crates/zmq)
