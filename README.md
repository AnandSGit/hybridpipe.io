# HybridPipe

<div align="center">

![HybridPipe Logo](https://user-images.githubusercontent.com/5903620/170846556-73feced0-73fc-4fb2-bb88-e04101954968.png)

**A unified messaging interface for microservices and distributed systems**

[![Go Reference](https://pkg.go.dev/badge/github.com/AnandSGit/hybridpipe.io.svg)](https://pkg.go.dev/github.com/AnandSGit/hybridpipe.io)
[![Go Report Card](https://goreportcard.com/badge/github.com/AnandSGit/hybridpipe.io)](https://goreportcard.com/report/github.com/AnandSGit/hybridpipe.io)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

*Version: 2.0.0 (June 2023)*

</div>

## 📋 Table of Contents

- [Overview](#-overview)
- [Features](#-features)
- [Supported Messaging Systems](#-supported-messaging-systems)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Usage](#-usage)
  - [Basic Concepts](#basic-concepts)
  - [Code Examples](#code-examples)
- [Security](#-security)
- [API Reference](#-api-reference)
- [Architecture](#-architecture)
- [Implementation Status](#-implementation-status)
- [Recent Improvements](#-recent-improvements)
- [Contributing](#-contributing)
- [License](#-license)

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

| Platform | Status | Description |
|----------|--------|-------------|
| [Apache Kafka](https://kafka.apache.org/) | ✅ Full | Distributed streaming platform |
| [NATS](https://nats.io/) | ✅ Full | Cloud-native messaging system |
| [RabbitMQ](https://www.rabbitmq.com/) | ✅ Full | Message broker supporting multiple protocols |
| [AMQP 1.0](https://www.amqp.org) | ✅ Full | Advanced Message Queuing Protocol |
| [Apache Qpid](https://qpid.apache.org/) | ✅ Full | Enterprise messaging system |
| [MQTT](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html) | ✅ Full | IoT connectivity protocol |
| [ZeroMQ](https://zeromq.org/) | 🔄 Partial | Distributed messaging library |
| [NSQ](https://nsq.io/) | 📅 Planned | Realtime distributed messaging platform |
| TCP/Redis | 📅 Planned | Direct TCP and Redis-based messaging |

## 📦 Installation

```bash
# Install the HybridPipe library
go get -u github.com/AnandSGit/hybridpipe.io

# Or add to your go.mod file
# require github.com/AnandSGit/hybridpipe.io v2.0.0
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

#### Using Multiple Messaging Systems Simultaneously

```go
// Connect to Kafka, NATS, Qpid, and MQTT
kafkaConn, _ := hp.DeployRouter(hp.KAFKA)
defer kafkaConn.Close()

natsConn, _ := hp.DeployRouter(hp.NATS)
defer natsConn.Close()

qpidConn, _ := hp.DeployRouter(hp.QPID)
defer qpidConn.Close()

mqttConn, _ := hp.DeployRouter(hp.MQTT)
defer mqttConn.Close()

// Send the same message via all four systems
data := map[string]string{"status": "active", "message": "System online"}

kafkaConn.Dispatch("app.status", data)
natsConn.Dispatch("app.status", data)
qpidConn.Dispatch("app.status", data)
mqttConn.Dispatch("app.status", data)
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

## 📊 Implementation Status

- **Fully Implemented**: Kafka, NATS, RabbitMQ, AMQP 1.0, Apache Qpid, MQTT
- **Partially Implemented**: ZeroMQ (structure ready)
- **Planned**: NSQ, TCP, Redis

An embedded system deployable version of HybridPipe is under development in Rust.

## 🔄 Recent Improvements

### New Protocol Implementations (June 2023)

- **MQTT Support**: Added full implementation of MQTT protocol using Eclipse Paho MQTT client
- **Apache Qpid Support**: Added full implementation of Apache Qpid protocol
- **Thread Safety**: Improved mutex protection across all protocol implementations
- **Error Handling**: Enhanced error context and recovery mechanisms

### Code Quality Improvements (June 2023)

- **Documentation**: Added comprehensive package and function documentation
- **Error Handling**: Implemented consistent error patterns with context
- **Configuration**: Added flexible configuration loading with environment variable support
- **Thread Safety**: Added mutex protection and proper resource cleanup
- **Robustness**: Added connection validation, panic recovery, and timeouts
- **Consistency**: Standardized function signatures and naming conventions

## 👥 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

