# HybridPipe

<div align="center">

![HybridPipe Logo](https://user-images.githubusercontent.com/5903620/170846556-73feced0-73fc-4fb2-bb88-e04101954968.png)

**A unified messaging interface for microservices and distributed systems**

[![Go Reference](https://pkg.go.dev/badge/github.com/AnandSGit/hybridpipe.io.svg)](https://pkg.go.dev/github.com/AnandSGit/hybridpipe.io)
[![Go Report Card](https://goreportcard.com/badge/github.com/AnandSGit/hybridpipe.io)](https://goreportcard.com/report/github.com/AnandSGit/hybridpipe.io)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

*Version: 2.0.0 (April 19, 2025)*

</div>

## Overview

HybridPipe provides a unified messaging interface for various message brokers and queuing systems. It enables seamless communication between microservices or individual processes via different messaging systems using a consistent API. With HybridPipe, you can switch between messaging platforms without changing your application code.

## Multi-Language Support

HybridPipe is now available in multiple programming languages:

- **Go**: The original implementation, located in the [golang](./golang) directory
- **Rust**: A new implementation with full feature parity, located in the [rust](./rust) directory

Both implementations provide the same core functionality and API concepts, allowing you to use the most appropriate language for your project while maintaining compatibility with other HybridPipe-based systems.

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

| Platform | Go | Rust | Description |
|----------|-------|-------|-------------|
| [Apache Kafka](https://kafka.apache.org/) | ✅ | ✅ | Distributed streaming platform |
| [NATS](https://nats.io/) | ✅ | ✅ | Cloud-native messaging system |
| [RabbitMQ](https://www.rabbitmq.com/) | ✅ | ✅ | Message broker supporting multiple protocols |
| [AMQP 1.0](https://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-overview-v1.0-os.html) | ✅ | ✅ | Advanced Message Queuing Protocol |
| [Apache Qpid](https://qpid.apache.org/) | ✅ | ✅ | Enterprise messaging system |
| [MQTT](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html) | ✅ | ✅ | IoT connectivity protocol |
| [NSQ](https://nsq.io/) | ✅ | ✅ | Realtime distributed messaging platform |
| [Redis](https://redis.io/) | ✅ | ✅ | In-memory data structure store |
| [TCP/IP](https://en.wikipedia.org/wiki/Internet_protocol_suite) | ✅ | ✅ | Direct TCP socket communication |
| [NetChan](https://github.com/AnandSGit/hybridpipe.io/netchan) | ✅ | ✅ | Go channels over network |

## Getting Started

### Go Implementation

```go
// Connect to Kafka
kafkaConn, err := hp.DeployRouter(hp.KAFKA)
if err != nil {
    log.Fatalf("Failed to connect to Kafka: %v", err)
}
defer kafkaConn.Close()

// Send a message
err = kafkaConn.Dispatch("app.users.events", person)
if err != nil {
    log.Printf("Failed to send message: %v", err)
}
```

For more details, see the [Go Implementation README](./golang/README.md).

### Rust Implementation

```rust
// Connect to Kafka
let kafka_conn = deploy_router(BrokerType::KAFKA)?;

// Send a message
kafka_conn.dispatch("app.users.events", Box::new(person)).await?;
```

For more details, see the [Rust Implementation README](./rust/README.md).

## Documentation

- [HybridPipe Documentation](./docs/HybridPipe.md)
- [Serialization Documentation](./docs/SERIALIZATION.md)
- [Tracing Documentation](./docs/TRACING_DOCUMENTATION.md)
- [Testing Documentation](./docs/testing.md)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
