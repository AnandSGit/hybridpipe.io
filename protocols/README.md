# HybridPipe Protocol Implementations

This directory contains implementations of the HybridPipe interface for various messaging protocols.

## Available Protocols

- **AMQP**: Advanced Message Queuing Protocol 1.0 implementation
- **Kafka**: Apache Kafka implementation
- **MQTT**: Message Queuing Telemetry Transport implementation
- **NATS**: NATS messaging system implementation
- **NetChan**: Go channels over network implementation
- **NSQ**: NSQ messaging platform implementation
- **Qpid**: Apache Qpid implementation
- **RabbitMQ**: RabbitMQ implementation (using AMQP 0.9.1)
- **Redis**: Redis pub/sub implementation
- **TCP**: Direct TCP/IP communication implementation
- **Mock**: In-memory mock implementation for testing

## Architecture

Each protocol implementation follows a consistent pattern:

1. **Interface Implementation**: Each protocol implements the `core.HybridPipe` interface
2. **Registration**: Each protocol registers itself with the core registry
3. **Configuration**: Each protocol reads its configuration from the global config
4. **Error Handling**: Each protocol provides detailed error information

## Adding a New Protocol

To add a new protocol implementation:

1. Create a new directory under `protocols/` for your protocol
2. Implement the `core.HybridPipe` interface
3. Create a `register.go` file that registers your implementation
4. Add appropriate configuration to the `core.HybridDB` struct
5. Update the main package to import your protocol

Example registration:

```go
package myprotocol

import (
    "reflect"
    "hybridpipe.io/core"
)

// Register registers the protocol with the HybridPipe system.
func Register() {
    core.RegisterRouter(core.MY_PROTOCOL, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the protocol when the package is imported.
func init() {
    Register()
}
```
