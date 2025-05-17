# NSQ Protocol

## Overview

NSQ is a realtime distributed messaging platform designed to operate at scale, handling billions of messages per day. It is a high-performance, distributed, and fault-tolerant messaging system that decouples producers and consumers.

## Key Features

- **Distributed**: No single point of failure
- **Scalable**: Horizontally scalable without any centralized brokers
- **Fault-Tolerant**: Messages are guaranteed to be delivered at least once
- **High-Performance**: Designed for high throughput
- **Discovery Service**: Automatic discovery of producers and consumers
- **Simple Topology**: Easy to set up and operate
- **Message Delivery Guarantees**: At-least-once delivery semantics
- **Backpressure**: Built-in backpressure mechanisms

## Implementation in HybridPipe

HybridPipe implements the NSQ protocol using the official NSQ client libraries, providing a seamless integration with NSQ.

### Go Implementation

The Go implementation uses the [github.com/nsqio/go-nsq](https://github.com/nsqio/go-nsq) package, which is the official Go client for NSQ.

### Rust Implementation

The Rust implementation uses a custom NSQ client built on top of the Tokio runtime.

## Configuration

### Go Implementation

```go
type NSQConfig struct {
    // NsqdAddress is the address of the nsqd instance (e.g., "localhost:4150")
    NsqdAddress string
    // LookupdAddresses is a list of nsqlookupd addresses (e.g., ["localhost:4161"])
    LookupdAddresses []string
    // Topic is the NSQ topic
    Topic string
    // Channel is the NSQ channel
    Channel string
    // MaxInFlight is the maximum number of messages to process at once
    MaxInFlight int
    // MaxAttempts is the maximum number of attempts to process a message
    MaxAttempts int
    // RequeueDelay is the delay before requeuing a failed message in milliseconds
    RequeueDelay int
    // HeartbeatInterval is the heartbeat interval in milliseconds
    HeartbeatInterval int
    // ReadTimeout is the read timeout in milliseconds
    ReadTimeout int
    // WriteTimeout is the write timeout in milliseconds
    WriteTimeout int
    // DialTimeout is the dial timeout in milliseconds
    DialTimeout int
}
```

### Rust Implementation

```rust
pub struct NSQConfig {
    // Nsqd address is the address of the nsqd instance (e.g., "localhost:4150")
    pub nsqd_address: String,
    // Lookupd addresses is a list of nsqlookupd addresses (e.g., ["localhost:4161"])
    pub lookupd_addresses: Vec<String>,
    // Topic is the NSQ topic
    pub topic: String,
    // Channel is the NSQ channel
    pub channel: String,
    // Max in flight is the maximum number of messages to process at once
    pub max_in_flight: i32,
    // Max attempts is the maximum number of attempts to process a message
    pub max_attempts: i32,
    // Requeue delay is the delay before requeuing a failed message in milliseconds
    pub requeue_delay: i64,
    // Heartbeat interval is the heartbeat interval in milliseconds
    pub heartbeat_interval: u64,
    // Read timeout is the read timeout in milliseconds
    pub read_timeout: u64,
    // Write timeout is the write timeout in milliseconds
    pub write_timeout: u64,
    // Dial timeout is the dial timeout in milliseconds
    pub dial_timeout: u64,
}
```

## Usage

### Go Implementation

```go
// Create an NSQ router with default configuration
router := nsq.New(nil)

// Connect to NSQ
if err := router.Connect(); err != nil {
    log.Fatalf("Failed to connect to NSQ: %v", err)
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
if err := router.Dispatch("greetings", "Hello, NSQ!"); err != nil {
    log.Fatalf("Failed to dispatch message: %v", err)
}
```

### Rust Implementation

```rust
// Deploy an NSQ router
let router = deploy_router(BrokerType::NSQ)?;

// Connect to NSQ
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
let message = "Hello, NSQ!".as_bytes().to_vec();
router.dispatch("greetings", Box::new(message)).await?;
```

## Topics and Channels

NSQ uses topics and channels to route messages:

- **Topics**: A topic is a distinct stream of data. Producers publish messages to topics.
- **Channels**: A channel is a logical grouping of consumers subscribed to a topic. Each channel receives a copy of all the messages for the topic, and the messages are distributed among the consumers of that channel.

In HybridPipe, the pipe name is used as the topic, and a default channel is used for all subscriptions.

## Performance Considerations

NSQ is designed for high throughput and low latency. Here are some tips for optimizing NSQ performance:

1. **Message Size**: Keep messages small to minimize bandwidth usage
2. **Max In Flight**: Set an appropriate max in flight value to control the flow of messages
3. **Requeue Delay**: Set an appropriate requeue delay to avoid overwhelming the system with requeued messages
4. **Heartbeat Interval**: Set an appropriate heartbeat interval to detect disconnections quickly

## Limitations

1. **At-Least-Once Delivery**: NSQ guarantees at-least-once delivery, which means messages may be delivered multiple times
2. **No Message Ordering**: NSQ does not guarantee message ordering
3. **No Message Persistence**: NSQ does not persist messages by default, although it can be configured to do so

## Examples

See the [examples](../../golang/protocols/nsq/example) directory for more examples of using NSQ with HybridPipe.io.

## References

- [NSQ Official Website](https://nsq.io/)
- [NSQ Documentation](https://nsq.io/overview/design.html)
- [Go NSQ Client](https://github.com/nsqio/go-nsq)
