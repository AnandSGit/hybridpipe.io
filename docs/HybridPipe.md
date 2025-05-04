# HybridPipe Documentation

## Overview

HybridPipe is a unified messaging interface for various message brokers. It enables communication between microservices or individual processes via different messaging systems like Kafka, NATS, RabbitMQ, ZeroMQ, AMQP 1.0, MQTT, and more.

## Features

- **Unified Interface**: A single, consistent API for multiple messaging systems
- **Protocol Agnostic**: Support for multiple messaging protocols
- **Pluggable Architecture**: Easy to extend with new protocols
- **Middleware Support**: Tracing, monitoring, and more
- **Context-Aware**: Support for context-based operations
- **Serialization Options**: Multiple serialization formats
- **Cross-Language Compatible**: Support for language-agnostic serialization

## Supported Protocols

- **NATS**: A lightweight, high-performance messaging system
- **Kafka**: A distributed streaming platform
- **RabbitMQ**: A robust messaging system for applications
- **AMQP 1.0**: Advanced Message Queuing Protocol
- **MQTT**: A lightweight messaging protocol for small sensors and mobile devices
- **Qpid**: Apache Qpid implementation of AMQP
- **NSQ**: A realtime distributed messaging platform
- **TCP/IP**: Direct TCP/IP communication
- **Redis**: In-memory data structure store used as a message broker
- **NetChan**: Go channels over network
- **Mock**: In-memory mock implementation for testing

## Installation

```bash
go get hybridpipe.io
```

## Basic Usage

```go
package main

import (
    "log"
    "time"

    "hybridpipe.io"
)

func main() {
    // Deploy a router for the desired messaging system
    router, err := hybridpipe.DeployRouter(hybridpipe.NATS)
    if err != nil {
        log.Fatalf("Failed to deploy router: %v", err)
    }
    defer router.Close()

    // Subscribe to a pipe
    err = router.Subscribe("my-pipe", func(data []byte) error {
        var message string
        if err := hybridpipe.Decode(data, &message); err != nil {
            return err
        }
        log.Printf("Received message: %s", message)
        return nil
    })
    if err != nil {
        log.Fatalf("Failed to subscribe: %v", err)
    }

    // Dispatch a message
    err = router.Dispatch("my-pipe", "Hello, HybridPipe!")
    if err != nil {
        log.Fatalf("Failed to dispatch message: %v", err)
    }

    // Wait for the message to be processed
    time.Sleep(1 * time.Second)
}
```

## Advanced Usage

### Context-Aware Operations

```go
package main

import (
    "context"
    "log"
    "time"

    "hybridpipe.io"
    "hybridpipe.io/core"
)

func main() {
    // Deploy a router for the desired messaging system
    router, err := hybridpipe.DeployRouter(hybridpipe.NATS)
    if err != nil {
        log.Fatalf("Failed to deploy router: %v", err)
    }
    defer router.Close()

    // Check if the router supports context-aware operations
    ctxRouter, ok := router.(core.ContextAwareHybridPipe)
    if !ok {
        log.Fatalf("Router does not support context-aware operations")
    }

    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Subscribe to a pipe with context
    err = ctxRouter.SubscribeWithContext(ctx, "my-pipe", func(data []byte) error {
        var message string
        if err := hybridpipe.Decode(data, &message); err != nil {
            return err
        }
        log.Printf("Received message: %s", message)
        return nil
    })
    if err != nil {
        log.Fatalf("Failed to subscribe: %v", err)
    }

    // Dispatch a message with context
    err = ctxRouter.DispatchWithContext(ctx, "my-pipe", "Hello, Context-Aware HybridPipe!")
    if err != nil {
        log.Fatalf("Failed to dispatch message: %v", err)
    }

    // Wait for the message to be processed
    time.Sleep(1 * time.Second)
}
```

### Middleware

```go
package main

import (
    "log"
    "time"

    "hybridpipe.io"
    "hybridpipe.io/middleware"
    "hybridpipe.io/monitoring"
    "hybridpipe.io/tracing"
)

func main() {
    // Deploy a router for the desired messaging system
    baseRouter, err := hybridpipe.DeployRouter(hybridpipe.NATS)
    if err != nil {
        log.Fatalf("Failed to deploy router: %v", err)
    }
    defer baseRouter.Close()

    // Add tracing middleware
    tracedRouter := tracing.NewTracedRouter(baseRouter, "my-service")

    // Add monitoring middleware
    monitoredRouter := monitoring.NewMonitoredRouter(tracedRouter, "my-router")

    // Subscribe to a pipe
    err = monitoredRouter.Subscribe("my-pipe", func(data []byte) error {
        var message string
        if err := hybridpipe.Decode(data, &message); err != nil {
            return err
        }
        log.Printf("Received message: %s", message)
        return nil
    })
    if err != nil {
        log.Fatalf("Failed to subscribe: %v", err)
    }

    // Dispatch a message
    err = monitoredRouter.Dispatch("my-pipe", "Hello, Middleware-Enabled HybridPipe!")
    if err != nil {
        log.Fatalf("Failed to dispatch message: %v", err)
    }

    // Wait for the message to be processed
    time.Sleep(1 * time.Second)

    // Get monitoring statistics
    stats := monitoredRouter.GetStats()
    log.Printf("Messages sent: %d", stats.GetMessagesSent())
    log.Printf("Messages received: %d", stats.GetMessagesReceived())
}
```

### Serialization Options

```go
package main

import (
    "log"

    "hybridpipe.io"
    "hybridpipe.io/core"
)

func main() {
    // Create serialization options
    options := &core.SerializationOptions{
        Format:      core.FormatJSON,
        Compression: true,
    }

    // Encode data with options
    data := map[string]interface{}{
        "message": "Hello, Serialization!",
        "count":   42,
    }
    encoded, err := hybridpipe.EncodeWithOptions(data, options)
    if err != nil {
        log.Fatalf("Failed to encode data: %v", err)
    }

    // Decode data with options
    var decoded map[string]interface{}
    err = hybridpipe.DecodeWithOptions(encoded, &decoded, options)
    if err != nil {
        log.Fatalf("Failed to decode data: %v", err)
    }

    log.Printf("Decoded message: %s", decoded["message"])
    log.Printf("Decoded count: %d", int(decoded["count"].(float64)))
}
```

## Configuration

HybridPipe can be configured using a TOML configuration file. By default, it looks for a file named `hybridpipe.toml` in the current directory.

```toml
# HybridPipe Configuration

[NATS]
NATSServer = "localhost"
NATSPort = 4222
NATSClusterName = "test-cluster"
NATSClientName = "hybridpipe-client"
NATSConnectTimeout = 5

[Kafka]
KafkaServer = "localhost"
KafkaPort = 9092
KafkaClientID = "hybridpipe-client"
KafkaConnectTimeout = 5

[RabbitMQ]
RabbitMQServer = "localhost"
RabbitMQPort = 5672
RabbitMQUsername = "guest"
RabbitMQPassword = "guest"
RabbitMQVHost = "/"
RabbitMQConnectTimeout = 5

[AMQP]
AMQPServer = "localhost"
AMQPPort = 5672
AMQPUsername = "guest"
AMQPPassword = "guest"
AMQPConnectTimeout = 5

[MQTT]
MQTTServer = "localhost"
MQTTPort = 1883
MQTTClientID = "hybridpipe-client"
MQTTConnectTimeout = 5
MQTTQoS = 1

[Qpid]
QpidServer = "localhost"
QpidPort = 5672
QpidUsername = "guest"
QpidPassword = "guest"
QpidConnectTimeout = 5

[NSQ]
NSQDAddress = "localhost"
NSQDPort = 4150
NSQLookupdAddresses = "localhost:4161"
NSQClientID = "hybridpipe-client"
NSQConnectTimeout = 5

[TCP]
TCPServer = "localhost"
TCPPort = 9000
TCPConnectTimeout = 5

[Redis]
RedisServer = "localhost"
RedisPort = 6379
RedisPassword = ""
RedisDB = 0
RedisConnectTimeout = 5

[NetChan]
NetChanBufferSize = 10
NetChanConnectTimeout = 5
```

## Error Handling

HybridPipe provides detailed error information for all operations. Errors are returned as standard Go errors and can be handled using normal Go error handling patterns.

```go
router, err := hybridpipe.DeployRouter(hybridpipe.NATS)
if err != nil {
    switch {
    case errors.Is(err, core.ErrNotConnected):
        log.Fatalf("Not connected to messaging system: %v", err)
    case errors.Is(err, core.ErrPipeNotFound):
        log.Fatalf("Pipe not found: %v", err)
    default:
        log.Fatalf("Unknown error: %v", err)
    }
}
```

## Testing

HybridPipe includes a mock implementation for testing purposes. This allows you to test your code without connecting to a real messaging system.

```go
package main

import (
    "testing"

    "hybridpipe.io"
    "hybridpipe.io/protocols/mock"
)

func TestMessaging(t *testing.T) {
    // Create a mock router
    router := mock.New("test-router")
    err := router.Connect()
    if err != nil {
        t.Fatalf("Failed to connect to mock router: %v", err)
    }
    defer router.Close()

    // Test pipe name
    pipeName := "test-pipe"

    // Test data
    testData := "Hello, Mock HybridPipe!"

    // Wait channel for synchronization
    received := make(chan bool)

    // Subscribe to the pipe
    err = router.Subscribe(pipeName, func(data []byte) error {
        var message string
        if err := hybridpipe.Decode(data, &message); err != nil {
            t.Errorf("Failed to decode message: %v", err)
            return err
        }
        if message != testData {
            t.Errorf("Expected %q, got %q", testData, message)
        }
        received <- true
        return nil
    })
    if err != nil {
        t.Fatalf("Failed to subscribe: %v", err)
    }

    // Dispatch a message
    err = router.Dispatch(pipeName, testData)
    if err != nil {
        t.Fatalf("Failed to dispatch message: %v", err)
    }

    // Wait for the message to be received
    select {
    case <-received:
        // Message received successfully
    case <-time.After(1 * time.Second):
        t.Fatal("Timed out waiting for message")
    }
}
```

## Contributing

Contributions to HybridPipe are welcome! Please see the [CONTRIBUTING.md](CONTRIBUTING.md) file for details.

## License

HybridPipe is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contact

For questions, issues, or feature requests, please open an issue on the [GitHub repository](https://github.com/hybridpipe/hybridpipe).

Last updated: April 19, 2025
