# HybridPipe - Rust Implementation

A unified messaging interface for microservices and distributed systems, implemented in Rust.

## Overview

HybridPipe provides a unified messaging interface for various message brokers and queuing systems. It enables seamless communication between microservices or individual processes via different messaging systems using a consistent API. With HybridPipe, you can switch between messaging platforms without changing your application code.

## Features

- **Unified API**: Single consistent interface for multiple messaging systems
- **Pluggable Architecture**: Easy to add support for new messaging platforms
- **Flexible Communication Patterns**: Support for both asynchronous and pseudo-synchronous communication
- **Type Safety**: Send and receive strongly-typed messages with automatic serialization/deserialization
- **Language Agnostic**: Support for multiple serialization formats (JSON, Protocol Buffers, MessagePack) for cross-language compatibility
- **Concurrent Processing**: Thread-safe operations with proper resource management
- **Secure Communication**: Built-in TLS support for secure messaging
- **Configurable**: Environment variable support and flexible configuration options
- **Robust Error Handling**: Consistent error patterns with detailed context
- **Message Tracing and Monitoring**: Comprehensive visibility into message flows across services

## Supported Messaging Systems

| Platform | Status | Description | Implementation |
|----------|--------|-------------|----------------|
| [Apache Kafka](https://kafka.apache.org/) | ✅ Full | Distributed streaming platform | rdkafka |
| [NATS](https://nats.io/) | ✅ Full | Cloud-native messaging system | async-nats |
| [RabbitMQ](https://www.rabbitmq.com/) | ✅ Full | Message broker supporting multiple protocols | lapin |
| [AMQP 1.0](https://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-overview-v1.0-os.html) | ✅ Full | Advanced Message Queuing Protocol | lapin |
| [Apache Qpid](https://qpid.apache.org/) | ✅ Full | Enterprise messaging system | lapin |
| [MQTT](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html) | ✅ Full | IoT connectivity protocol | rumqttc |
| [NSQ](https://nsq.io/) | ✅ Full | Realtime distributed messaging platform | nsq-client |
| [Redis](https://redis.io/) | ✅ Full | In-memory data structure store | redis |
| [TCP/IP](https://en.wikipedia.org/wiki/Internet_protocol_suite) | ✅ Full | Direct TCP socket communication | tokio |
| [NetChan](https://github.com/AnandSGit/hybridpipe.io/netchan) | ✅ Full | Go channels over network | tokio |

## Installation

Add HybridPipe to your Cargo.toml:

```toml
[dependencies]
hybridpipe = "0.1.0"
```

## Usage

### Basic Concepts

HybridPipe uses a few key concepts:

- **Broker Type**: The messaging system you want to use (Kafka, NATS, etc.)
- **Pipe**: A named channel for sending/receiving messages (similar to topics or subjects)
- **Process Function**: A callback function that processes received messages
- **Connection Handle**: An object representing a connection to a messaging system

### Example: Publish Messages

```rust
use hybridpipe::{deploy_router, BrokerType};
use serde::{Serialize, Deserialize};

#[derive(Serialize, Deserialize, Debug)]
struct Person {
    name: String,
    age: u32,
    next_gen: Vec<String>,
    c_age: Vec<u32>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Connect to Kafka
    let kafka_conn = deploy_router(BrokerType::KAFKA)?;
    
    // Create a message
    let person = Person {
        name: "David Gower".to_string(),
        age: 75,
        next_gen: vec!["Pringle".to_string(), "NH Fairbrother".to_string(), "Wasim".to_string()],
        c_age: vec![45, 37, 39],
    };
    
    // Send the message to a pipe
    kafka_conn.dispatch("app.users.events", Box::new(person)).await?;
    
    Ok(())
}
```

### Example: Subscribe to Messages

```rust
use hybridpipe::{deploy_router, BrokerType};
use serde::{Serialize, Deserialize};
use std::sync::Arc;
use tokio::signal;

#[derive(Serialize, Deserialize, Debug)]
struct Person {
    name: String,
    age: u32,
    next_gen: Vec<String>,
    c_age: Vec<u32>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Connect to NATS
    let nats_conn = deploy_router(BrokerType::NATS)?;
    
    // Define a message handler
    nats_conn.accept("app.users.events", |data| {
        if let Some(person) = data.downcast_ref::<Person>() {
            println!("Received person: {}, age: {}", person.name, person.age);
        } else {
            println!("Received data of unknown type");
        }
    }).await?;
    
    // Keep the application running until Ctrl+C
    signal::ctrl_c().await?;
    
    Ok(())
}
```

## Configuration

HybridPipe uses a TOML configuration file for setting up connections to various messaging platforms. The configuration file can be placed in several locations:

- Current directory: `./hybridpipe_db.toml`
- System-wide: `/etc/hybridpipe/hybridpipe_db.toml`
- Application-specific: `/opt/hybridpipe/config/hybridpipe_db.toml`
- User-specific: `$HOME/.config/hybridpipe/hybridpipe_db.toml`

You can also specify a custom configuration file location using the `HYBRIDPIPE_CONFIG` environment variable.

## Building from Source

```bash
# Clone the repository
git clone https://github.com/AnandSGit/hybridpipe.io.git
cd hybridpipe.io/rust

# Build the library
cargo build --release

# Run the tests
cargo test

# Run the examples
cargo run --example kafka
cargo run --example nats
```

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
