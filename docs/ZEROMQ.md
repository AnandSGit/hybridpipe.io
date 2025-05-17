# ZeroMQ Protocol Documentation

ZeroMQ (also known as ØMQ, 0MQ, or zmq) is a high-performance asynchronous messaging library designed to be used in distributed or concurrent applications. HybridPipe.io provides a seamless integration with ZeroMQ, allowing you to use it as a messaging protocol alongside other protocols.

## Overview

ZeroMQ provides a message-oriented middleware that is lightweight and fast. It supports various messaging patterns, including:

- **Request-Reply**: Connects a set of clients to a set of services
- **Publish-Subscribe**: Connects a set of publishers to a set of subscribers
- **Push-Pull (Pipeline)**: Connects nodes in a fan-out/fan-in pattern
- **Exclusive Pair**: Connects two sockets exclusively

HybridPipe.io implements the Publish-Subscribe pattern for ZeroMQ, which is well-suited for broadcasting messages to multiple receivers.

## Installation

### Go Implementation

ZeroMQ requires the installation of the ZeroMQ library on your system:

#### Ubuntu/Debian
```bash
sudo apt-get install libzmq3-dev
```

#### macOS
```bash
brew install zeromq
```

#### Windows
Download and install the ZeroMQ library from the [official website](https://zeromq.org/download/).

Then, install the Go binding:
```bash
go get github.com/pebbe/zmq4
```

### Rust Implementation

Add the following dependencies to your Cargo.toml:
```toml
[dependencies]
zmq = "0.10"
```

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
package main

import (
    "fmt"
    "log"

    "hybridpipe.io/core"
    "hybridpipe.io/protocols/zeromq"
)

func main() {
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
}
```

### Rust Implementation

```rust
use hybridpipe::{deploy_router, BrokerType, Process, Result};
use log::{error, info};
use std::sync::Arc;
use std::time::Duration;
use tokio::time;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging
    env_logger::init();

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

    // Wait for the message to be processed
    time::sleep(Duration::from_millis(200)).await;

    // Disconnect
    router.disconnect().await?;

    Ok(())
}
```

## Advanced Usage

### Publisher-Subscriber Pattern

ZeroMQ's Publish-Subscribe pattern is a one-way data distribution pattern. Publishers produce messages, and subscribers consume them. The key features of this pattern are:

1. **One-to-Many**: A single publisher can send messages to multiple subscribers
2. **Topic-Based Filtering**: Subscribers can filter messages based on topics
3. **Asynchronous**: Publishers don't wait for subscribers to receive messages

In HybridPipe.io, the pipe name is used as the topic for ZeroMQ messages.

### Binding vs. Connecting

ZeroMQ sockets can either bind to an endpoint or connect to an endpoint:

- **Bind**: The socket listens for incoming connections
- **Connect**: The socket initiates a connection to a bound socket

The choice between binding and connecting depends on your network architecture:

- If you have a fixed number of publishers and a dynamic number of subscribers, publishers should bind and subscribers should connect
- If you have a dynamic number of publishers and a fixed number of subscribers, publishers should connect and subscribers should bind

### High Water Mark

The high water mark (HWM) is a hard limit on the maximum number of messages that ZeroMQ will queue in memory for a socket. If this limit is reached, ZeroMQ will take action depending on the socket type:

- For PUB sockets, new messages will be dropped
- For SUB sockets, the socket will block or drop messages depending on the socket option

Setting appropriate HWM values is important for preventing memory issues in high-throughput scenarios.

## Examples

See the [examples](../golang/protocols/zeromq/example) directory for more examples of using ZeroMQ with HybridPipe.io.

## Integration with HybridPipe

ZeroMQ is fully integrated with the HybridPipe system, allowing you to use it as a messaging protocol alongside other protocols like NATS, Kafka, and RabbitMQ.

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

## Troubleshooting

1. **Connection Issues**: Check that the endpoints are correct and that there are no firewall issues
2. **Message Loss**: Check the high water mark settings and ensure that subscribers are connected before publishers start sending messages
3. **Performance Issues**: Check the message size, socket type, and transport protocol
