# HybridPipe

<div align="center">

![HybridPipe Logo](https://user-images.githubusercontent.com/5903620/170846556-73feced0-73fc-4fb2-bb88-e04101954968.png)

**A unified messaging interface for microservices and distributed systems**

[![Go Reference](https://pkg.go.dev/badge/github.com/AnandSGit/hybridpipe.io.svg)](https://pkg.go.dev/github.com/AnandSGit/hybridpipe.io)
[![Go Report Card](https://goreportcard.com/badge/github.com/AnandSGit/hybridpipe.io)](https://goreportcard.com/report/github.com/AnandSGit/hybridpipe.io)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

*Version: 2.0.0 (April 19, 2025)*

</div>

## 📋 Table of Contents

- [HybridPipe](#hybridpipe)
  - [📋 Table of Contents](#-table-of-contents)
  - [🔍 Overview](#-overview)
  - [✨ Features](#-features)
  - [🚀 Supported Messaging Systems](#-supported-messaging-systems)
  - [📦 Installation](#-installation)
  - [⚙️ Configuration](#️-configuration)
    - [Sample Configuration](#sample-configuration)
  - [🔧 Usage](#-usage)
    - [Basic Concepts](#basic-concepts)
    - [Protocol Selection Guide](#protocol-selection-guide)
    - [Code Examples](#code-examples)
      - [Initialize and Register Custom Types](#initialize-and-register-custom-types)
      - [Publish Messages](#publish-messages)
      - [Subscribe to Messages](#subscribe-to-messages)
      - [NSQ Example](#nsq-example)
      - [TCP Example](#tcp-example)
      - [Redis Example](#redis-example)
      - [ZeroMQ Example](#zeromq-example)
      - [NetChan Example](#netchan-example)
      - [Using Multiple Messaging Systems Simultaneously](#using-multiple-messaging-systems-simultaneously)
  - [🔒 Security](#-security)
    - [Generating TLS Certificates](#generating-tls-certificates)
  - [📚 API Reference](#-api-reference)
    - [Core Interface](#core-interface)
    - [Key Functions](#key-functions)
    - [Broker Type Constants](#broker-type-constants)
  - [🏗️ Architecture](#️-architecture)
    - [Design Principles](#design-principles)
  - [💻 Use Cases](#-use-cases)
    - [Microservices Communication](#microservices-communication)
    - [IoT Data Collection](#iot-data-collection)
    - [Event-Driven Architecture](#event-driven-architecture)
    - [Migration Between Messaging Systems](#migration-between-messaging-systems)
  - [📊 Implementation Status](#-implementation-status)
  - [🔄 Recent Improvements](#-recent-improvements)
    - [New Protocol Implementations (April 19, 2025)](#new-protocol-implementations-april-19-2025)
    - [Code Quality Improvements (April 19, 2025)](#code-quality-improvements-april-19-2025)
  - [📈 Future Roadmap](#-future-roadmap)
    - [Short-Term Roadmap](#short-term-roadmap)
    - [Long-Term Vision](#long-term-vision)
  - [👥 Contributing](#-contributing)
  - [📄 License](#-license)

## 🔍 Overview

HybridPipe provides a unified messaging interface for various message brokers and queuing systems. It enables seamless communication between microservices or individual processes via different messaging systems using a consistent API. With HybridPipe, you can switch between messaging platforms without changing your application code.

## ✨ Features

- **Unified API**: Single consistent interface for multiple messaging systems
- **Pluggable Architecture**: Easy to add support for new messaging platforms
- **Flexible Communication Patterns**: Support for both asynchronous and pseudo-synchronous communication
- **Type Safety**: Send and receive strongly-typed messages with automatic serialization/deserialization
- **Concurrent Processing**: Thread-safe operations with proper resource management
- **Secure Communication**: Built-in TLS support for secure messaging
- **Configurable**: Environment variable support and flexible configuration options
- **Robust Error Handling**: Consistent error patterns with detailed context

## 🚀 Supported Messaging Systems

| Platform | Status | Description | Implementation |
|----------|--------|-------------|----------------|
| [Apache Kafka](https://kafka.apache.org/) | ✅ Full | Distributed streaming platform | Native library |
| [NATS](https://nats.io/) | ✅ Full | Cloud-native messaging system | Native library |
| [RabbitMQ](https://www.rabbitmq.com/) | ✅ Full | Message broker supporting multiple protocols | Native library |
| [AMQP 1.0](https://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-overview-v1.0-os.html) | ✅ Full | Advanced Message Queuing Protocol | Native library |
| [Apache Qpid](https://qpid.apache.org/) | ✅ Full | Enterprise messaging system | Native library |
| [MQTT](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html) | ✅ Full | IoT connectivity protocol | Native library |
| [NSQ](https://nsq.io/) | ✅ Full | Realtime distributed messaging platform | Native library |
| [Redis](https://redis.io/) | ✅ Full | In-memory data structure store | Native library |
| [TCP/IP](https://en.wikipedia.org/wiki/Internet_protocol_suite) | ✅ Full | Direct TCP socket communication | Native library |
| [ZeroMQ](https://zeromq.org/) | ✅ Full | Distributed messaging library | Native library |
| [NetChan](https://github.com/AnandSGit/hybridpipe.io/netchan) | ✅ Full | Go channels over network | Native library |

## 📦 Installation

```bash
# Install the HybridPipe library
go get -u github.com/AnandSGit/hybridpipe.io

# Or add to your go.mod file
# require github.com/AnandSGit/hybridpipe.io v2.0.0 // April 19, 2025
```

## ⚙️ Configuration

HybridPipe uses a TOML configuration file for setting up connections to various messaging platforms. The configuration file can be placed in several locations:

- Current directory: `./hybridpipe_db.toml`
- System-wide: `/etc/hybridpipe/hybridpipe_db.toml`
- Application-specific: `/opt/hybridpipe/config/hybridpipe_db.toml`
- User-specific: `$HOME/.config/hybridpipe/hybridpipe_db.toml`

You can also specify a custom configuration file location using the `HYBRIDPIPE_CONFIG` environment variable.

### Sample Configuration

```toml
# HybridPipe Configuration File

[NATS]
NServers           = "localhost"
NLPort             = 4222
NMPort             = 8222
NCPort             = 6222
NATSCertFile       = "/path/to/nats-client.pem"
NATSKeyFile        = "/path/to/nats-client.key"
NATSCAFile         = "/path/to/nats-server-ca.pem"
NAllow_Reconnect   = true
NMax_Attempt       = 10
NReconnect_Wait    = 5
NTimeout           = 2

[KAFKA]
KServers   = "localhost"
KLPort     = 9093
KTimeout   = 10
KAFKACertFile       = "/path/to/kafka-client.cert.pem"
KAFKAKeyFile        = "/path/to/kafka-client.key.pem"
KAFKACAFile         = "/path/to/kafka-rootca.cert.pem"

[RABBITMQ]
RServerPort    = "amqp://guest:guest@localhost:5672/"

[AMQP1]
AMQPServer = "amqp://0.0.0.0:5672/"
AMQPPort   = 5672

[QPID]
QpidServer = "localhost"
QpidPort = 5672
QpidUsername = "guest"
QpidPassword = "guest"
QpidCertFile = "/path/to/qpid-client.pem"
QpidKeyFile = "/path/to/qpid-client.key"
QpidCAFile = "/path/to/qpid-ca.pem"
QpidTimeout = 10

[MQTT]
MQTTBroker = "tcp://localhost:1883"
MQTTClientID = "hybridpipe-client"
MQTTUsername = "user"
MQTTPassword = "password"
MQTTCleanSession = true
MQTTQoS = 1
MQTTRetained = false
MQTTCertFile = "/path/to/mqtt-client.pem"
MQTTKeyFile = "/path/to/mqtt-client.key"
MQTTCAFile = "/path/to/mqtt-ca.pem"
MQTTConnectTimeout = 10

[ZEROMQ]
ZMQEndpoints = "tcp://localhost:5555,tcp://localhost:5556"
ZMQPublishEndpoint = "tcp://*:5555"
ZMQSubscribeEndpoint = "tcp://localhost:5555"
ZMQSocketType = "PUB_SUB"
ZMQHighWaterMark = 1000
ZMQLinger = 1000
ZMQConnectTimeout = 10
ZMQIOThreads = 1
ZMQSecurityMechanism = "NULL"
ZMQCurveServerKey = ""
ZMQCurvePublicKey = ""
ZMQCurveSecretKey = ""

[NETCHAN]
NetChanServer = "localhost"
NetChanPort = 8080
NetChanBufferSize = 10
NetChanTimeout = 5000

[GENERAL]
DBLocation     = "/path/to/config"
```

## 🔧 Usage

### Basic Concepts

HybridPipe uses a few key concepts:

- **Broker Type**: The messaging system you want to use (Kafka, NATS, etc.)
- **Pipe**: A named channel for sending/receiving messages (similar to topics or subjects)
- **Process Function**: A callback function that processes received messages
- **Connection Handle**: An object representing a connection to a messaging system

### Protocol Selection Guide

Choosing the right protocol depends on your specific requirements:

| Protocol | Best For | Key Strengths | Considerations |
|----------|----------|---------------|----------------|
| **Kafka** | High-volume data streaming, event sourcing | Durability, scalability, replay capability | Higher resource usage, complexity |
| **NATS** | Microservices, request-reply patterns | Ultra-fast, lightweight, simple | Less durable by default |
| **RabbitMQ** | Complex routing, workflow processing | Flexible routing, mature ecosystem | Can be resource-intensive |
| **MQTT** | IoT devices, constrained environments | Lightweight, low bandwidth | Limited routing capabilities |
| **NSQ** | High-throughput distributed messaging | Simple, distributed by design | Less feature-rich than some alternatives |
| **Redis** | Simple pub/sub, caching integration | Fast, simple setup, widely deployed | Limited delivery guarantees |
| **TCP/IP** | Custom protocols, direct communication | Full control, no dependencies | Requires more manual implementation |
| **AMQP/Qpid** | Enterprise integration | Standardized, interoperable | More complex setup |
| **NetChan** | Go inter-process communication | Familiar Go channel semantics, type safety | Limited to Go applications |

### Code Examples

#### Initialize and Register Custom Types

```go
import (
    hp "github.com/AnandSGit/hybridpipe.io"
)

// Define a custom message type
type Person struct {
    Name    string
    Age     int
    NextGen []string
    CAge    []int
}

// Register the custom type with HybridPipe
func init() {
    hp.Enable(Person{})
}
```

#### Publish Messages

```go
// Connect to Kafka
kafkaConn, err := hp.DeployRouter(hp.KAFKA)
if err != nil {
    log.Fatalf("Failed to connect to Kafka: %v", err)
}
defer kafkaConn.Close()

// Create a message
person := Person{
    Name:    "David Gower",
    Age:     75,
    NextGen: []string{"Pringle", "NH Fairbrother", "Wasim"},
    CAge:    []int{45, 37, 39},
}

// Send the message to a pipe
err = kafkaConn.Dispatch("app.users.events", person)
if err != nil {
    log.Printf("Failed to send message: %v", err)
}
```

#### Subscribe to Messages

```go
// Connect to NATS
natsConn, err := hp.DeployRouter(hp.NATS)
if err != nil {
    log.Fatalf("Failed to connect to NATS: %v", err)
}
defer natsConn.Close()

// Define a message handler
handleMessage := func(data interface{}) {
    if person, ok := data.(*Person); ok {
        log.Printf("Received person: %s, age: %d", person.Name, person.Age)
    } else {
        log.Printf("Received data: %v", data)
    }
}

// Subscribe to a pipe
err = natsConn.Accept("app.users.events", handleMessage)
if err != nil {
    log.Printf("Failed to subscribe: %v", err)
}

// Keep the application running
select {}
```

#### NSQ Example

```go
// Connect to NSQ
nsqConn, err := hp.DeployRouter(hp.NSQ)
if err != nil {
    log.Fatalf("Failed to connect to NSQ: %v", err)
}
defer nsqConn.Close()

// Define a message handler
handleLogEvent := func(data interface{}) {
    if event, ok := data.(map[string]interface{}); ok {
        log.Printf("Log event: %v, level: %v",
            event["message"], event["level"])
    }
}

// Subscribe to a topic
nsqConn.Accept("system.logs", handleLogEvent)

// Publish a log event
logEvent := map[string]interface{}{
    "timestamp": time.Now().Unix(),
    "level":     "error",
    "message":   "Database connection failed",
    "source":    "api-server",
}
nsqConn.Dispatch("system.logs", logEvent)
```

#### TCP Example

```go
// Connect to TCP server (client mode)
tcpConn, err := hp.DeployRouter(hp.TCP)
if err != nil {
    log.Fatalf("Failed to connect to TCP server: %v", err)
}
defer tcpConn.Close()

// Define a message handler
handleTCPMessage := func(data interface{}) {
    log.Printf("Received TCP message: %v", data)
}

// Subscribe to a channel
tcpConn.Accept("direct", handleTCPMessage)

// Send a message
message := map[string]interface{}{
    "command": "status",
    "params":  []string{"cpu", "memory"},
}
tcpConn.Dispatch("direct", message)
```

#### Redis Example

```go
// Connect to Redis
redisConn, err := hp.DeployRouter(hp.REDIS)
if err != nil {
    log.Fatalf("Failed to connect to Redis: %v", err)
}
defer redisConn.Close()

// Define a message handler
handleNotification := func(data interface{}) {
    if notif, ok := data.(map[string]interface{}); ok {
        log.Printf("Notification: %v (priority: %v)",
            notif["message"], notif["priority"])
    }
}

// Subscribe to a channel
redisConn.Accept("notifications", handleNotification)

// Send a notification
notification := map[string]interface{}{
    "type":     "alert",
    "priority": "high",
    "message":  "System update required",
    "timestamp": time.Now().Format(time.RFC3339),
}
redisConn.Dispatch("notifications", notification)
```

#### ZeroMQ Example

```go
// Connect to ZeroMQ
zmqConn, err := hp.DeployRouter(hp.ZEROMQ)
if err != nil {
    log.Fatalf("Failed to connect to ZeroMQ: %v", err)
}
defer zmqConn.Close()

// Define a message handler
handleZMQMessage := func(data interface{}) {
    if msg, ok := data.(map[string]interface{}); ok {
        log.Printf("ZeroMQ message: %v (type: %v)",
            msg["payload"], msg["type"])
    }
}

// Subscribe to a topic
zmqConn.Accept("commands", handleZMQMessage)

// Send a message
message := map[string]interface{}{
    "type":    "command",
    "payload": "restart",
    "target":  "service-a",
    "timestamp": time.Now().Unix(),
}
zmqConn.Dispatch("commands", message)
```

#### NetChan Example

```go
// Connect to NetChan
netChanConn, err := hp.DeployRouter(hp.NETCHAN)
if err != nil {
    log.Fatalf("Failed to connect to NetChan: %v", err)
}
defer netChanConn.Close()

// Define a message handler
handleNetChanMessage := func(data interface{}) {
    if msg, ok := data.(map[string]interface{}); ok {
        log.Printf("NetChan message: %v (from: %v)",
            msg["content"], msg["sender"])
    }
}

// Subscribe to a channel
netChanConn.Accept("interprocess", handleNetChanMessage)

// Send a message
message := map[string]interface{}{
    "sender":   "service-a",
    "content":  "Hello from another process!",
    "timestamp": time.Now().Unix(),
}
netChanConn.Dispatch("interprocess", message)

// Using native Go channel syntax with NetChan
// Create a channel adapter
adapter := NewNetChanAdapter(10)
defer adapter.Close()

// Get send and receive channels for a pipe
sendChan, _ := adapter.GetSendChan("my-channel")
recvChan, _ := adapter.GetRecvChan("my-channel")

// Send using native channel syntax
sendChan <- "Hello, NetChan!"

// Receive using native channel syntax
msg, ok := <-recvChan
if ok {
    fmt.Printf("Received: %s\n", msg)
}
```

#### Using Multiple Messaging Systems Simultaneously

```go
// Connect to multiple messaging systems
kafkaConn, _ := hp.DeployRouter(hp.KAFKA)
defer kafkaConn.Close()

natsConn, _ := hp.DeployRouter(hp.NATS)
defer natsConn.Close()

qpidConn, _ := hp.DeployRouter(hp.QPID)
defer qpidConn.Close()

mqttConn, _ := hp.DeployRouter(hp.MQTT)
defer mqttConn.Close()

nsqConn, _ := hp.DeployRouter(hp.NSQ)
defer nsqConn.Close()

redisConn, _ := hp.DeployRouter(hp.REDIS)
defer redisConn.Close()

tcpConn, _ := hp.DeployRouter(hp.TCP)
defer tcpConn.Close()

zmqConn, _ := hp.DeployRouter(hp.ZEROMQ)
defer zmqConn.Close()

netChanConn, _ := hp.DeployRouter(hp.NETCHAN)
defer netChanConn.Close()

// Send the same message via all systems
data := map[string]string{"status": "active", "message": "System online"}

kafkaConn.Dispatch("app.status", data)
natsConn.Dispatch("app.status", data)
qpidConn.Dispatch("app.status", data)
mqttConn.Dispatch("app.status", data)
nsqConn.Dispatch("app.status", data)
redisConn.Dispatch("app.status", data)
tcpConn.Dispatch("app.status", data)
zmqConn.Dispatch("app.status", data)
netChanConn.Dispatch("app.status", data)
```

## 🔒 Security

HybridPipe supports TLS for secure communication with messaging platforms. You can configure TLS certificates in the configuration file.

### Generating TLS Certificates

```bash
# Generate a Certificate Authority (CA)
openssl req -new -x509 -days 365 -nodes -out ca.pem -keyout ca-key.pem

# Generate server certificate
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial -out server.pem -days 365

# Generate client certificate
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr
openssl x509 -req -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial -out client.pem -days 365
```

## 📚 API Reference

### Core Interface

```go
// HybridPipe defines the interface for all messaging implementations
type HybridPipe interface {
    // Connect establishes a connection to the messaging system
    Connect() error

    // Dispatch sends a message to the specified pipe
    Dispatch(pipe string, data interface{}) error

    // Accept subscribes to messages from the specified pipe
    Accept(pipe string, fn Process) error

    // Remove unsubscribes from the specified pipe
    Remove(pipe string) error

    // Close terminates the connection to the messaging system
    Close()
}
```

### Key Functions

| Function | Description |
|----------|-------------|
| `DeployRouter(brokerType int) (HybridPipe, error)` | Creates a connection to the specified messaging system |
| `Enable(dataType interface{})` | Registers a custom data type for serialization |
| `Encode(data interface{}) ([]byte, error)` | Serializes data to bytes |
| `Decode(data []byte, target interface{}) error` | Deserializes bytes to a target object |

### Broker Type Constants

```go
const (
    NATS = iota
    KAFKA
    RABBITMQ
    ZEROMQ
    AMQP1
    MQTT
    QPID
    NSQ
    TCP
    REDIS
    NETCHAN
    MOCK
)
```

## 🏗️ Architecture

HybridPipe uses a pluggable architecture with the following components:

1. **Core Interface**: The `HybridPipe` interface that all messaging implementations must satisfy
2. **Broker Implementations**: Concrete implementations for each messaging system
3. **Configuration Manager**: Handles loading and processing configuration
4. **Serialization System**: Handles encoding/decoding of messages
5. **Connection Factory**: Creates and manages connections to messaging systems

### Design Principles

1. **Abstraction**: Hide the complexity of different messaging systems behind a unified interface
2. **Loose Coupling**: Minimize dependencies between components
3. **Extensibility**: Make it easy to add support for new messaging platforms
4. **Reliability**: Ensure robust error handling and resource management
5. **Simplicity**: Expose a minimal, intuitive API to users

## 💻 Use Cases

HybridPipe is designed to solve a variety of messaging challenges in distributed systems:

### Microservices Communication

HybridPipe provides a consistent messaging interface for microservices, allowing them to communicate using the most appropriate protocol for each interaction.

```go
// Service A: Publishes events to Kafka for durability
kafkaRouter, _ := hp.DeployRouter(hp.KAFKA)
kafkaRouter.Dispatch("user.created", userData)

// Service B: Uses NATS for real-time notifications
natsRouter, _ := hp.DeployRouter(hp.NATS)
natsRouter.Dispatch("notifications", notification)
```

### IoT Data Collection

HybridPipe can be used to collect data from IoT devices using MQTT and then process it using other messaging systems.

```go
// IoT Gateway: Collects data from devices via MQTT
mqttRouter, _ := hp.DeployRouter(hp.MQTT)
mqttRouter.Accept("sensors/+/temperature", func(data interface{}) {
    // Process sensor data
    // Forward to data processing pipeline
    kafkaRouter.Dispatch("sensor-data", data)
})
```

### Event-Driven Architecture

HybridPipe enables event-driven architectures with different messaging systems for different types of events.

```go
// Critical events go to RabbitMQ for guaranteed delivery
rmqRouter, _ := hp.DeployRouter(hp.RABBITMQ)
rmqRouter.Dispatch("events.critical", criticalEvent)

// High-volume metrics go to Kafka for scalability
kafkaRouter, _ := hp.DeployRouter(hp.KAFKA)
kafkaRouter.Dispatch("metrics", metricData)
```

### Migration Between Messaging Systems

HybridPipe facilitates migration between different messaging systems by allowing both to run simultaneously.

```go
// Old system using RabbitMQ
oldRouter, _ := hp.DeployRouter(hp.RABBITMQ)

// New system using Kafka
newRouter, _ := hp.DeployRouter(hp.KAFKA)

// During migration, publish to both
func publishOrder(order Order) {
    oldRouter.Dispatch("orders", order)
    newRouter.Dispatch("orders", order)
}
```

## 📊 Implementation Status

- **Fully Implemented**: Kafka, NATS, RabbitMQ, AMQP 1.0, Apache Qpid, MQTT, NSQ, TCP/IP, Redis, ZeroMQ, NetChan
- **Partially Implemented**: None

An embedded system deployable version of HybridPipe is under development in Rust.

## 🔄 Recent Improvements

### New Protocol Implementations (April 19, 2025)

- **NetChan Support**: Added implementation of Go channels over network for inter-process communication
  - **Native Channel Syntax**: Added support for using Go's native channel operators (`<-` and `->`) with NetChan
  - **Channel Wrapper**: Implemented a wrapper that provides Go channel semantics over the network
  - **Select Statement Support**: Full support for using NetChan with Go's select statement
- **ZeroMQ Support**: Added full implementation of ZeroMQ protocol using zmq4 library
- **NSQ Support**: Added full implementation of NSQ protocol using go-nsq client
- **TCP/IP Support**: Added direct TCP/IP communication with both client and server modes
- **Redis Support**: Added Redis pub/sub implementation using go-redis client
- **MQTT Support**: Added full implementation of MQTT protocol using Eclipse Paho MQTT client
- **Apache Qpid Support**: Added full implementation of Apache Qpid protocol
- **Thread Safety**: Improved mutex protection across all protocol implementations
- **Error Handling**: Enhanced error context and recovery mechanisms

### Code Quality Improvements (April 19, 2025)

- **Documentation**: Added comprehensive package and function documentation
- **Error Handling**: Implemented consistent error patterns with context
- **Configuration**: Added flexible configuration loading with environment variable support
- **Thread Safety**: Added mutex protection and proper resource cleanup
- **Robustness**: Added connection validation, panic recovery, and timeouts
- **Consistency**: Standardized function signatures and naming conventions

## 📈 Future Roadmap

HybridPipe has several planned enhancements:

### Short-Term Roadmap

1. **Additional Protocol Support**:
   - Apache Pulsar integration
   - gRPC streaming support
   - WebSocket support

2. **Enhanced Features**:
   - Message batching for high-throughput scenarios
   - Automatic reconnection and recovery for all protocols
   - Circuit breaker pattern implementation

3. **Performance Improvements**:
   - Optimized serialization options (Protocol Buffers, MessagePack)
   - Connection pooling for all protocols
   - Asynchronous dispatch operations

### Long-Term Vision

1. **Cross-Language Support**:
   - Client libraries for other languages (Python, Java, JavaScript)
   - Language-agnostic serialization formats

2. **Advanced Patterns**:
   - Request-reply pattern support
   - Publish-subscribe with filtering
   - Competing consumers pattern

3. **Enterprise Features**:
   - Message tracing and monitoring
   - Integration with service meshes
   - Enhanced security features

## 👥 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

