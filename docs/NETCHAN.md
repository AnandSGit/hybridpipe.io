# NetChan Documentation

NetChan is a library that implements Go channel capability over a network between different processes. It provides a simple and intuitive API for sending and receiving messages between processes using Go's native channel operators (`<-` and `->`).

## Overview

NetChan allows you to create channels that can be used to send and receive messages between processes running on different machines. It provides a similar API to Go's native channels, making it easy to use for Go developers.

Key features:
- **Go Channel Semantics**: Use familiar Go channel operators for network communication
- **Type Safety**: Send and receive strongly-typed messages
- **Automatic Serialization**: Messages are automatically serialized and deserialized
- **Buffered Channels**: Configure buffer sizes to control message flow
- **Reconnection**: Automatic reconnection on network failures
- **Timeout Control**: Configure timeouts for send and receive operations

## Installation

```bash
go get -u hybridpipe.io/netchan
```

## Usage

### Server

```go
package main

import (
    "fmt"
    "log"

    "hybridpipe.io/netchan"
)

func main() {
    // Create a server with default configuration
    server, err := netchan.NewServer(nil)
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }
    defer server.Close()

    // Create a channel
    ch := server.Chan("example")

    // Receive a message
    value := ch.RecvOp()
    fmt.Printf("Received: %v\n", value)

    // Send a response
    ch.SendOp("Response")
}
```

### Client

```go
package main

import (
    "fmt"
    "log"

    "hybridpipe.io/netchan"
)

func main() {
    // Create a client with default configuration
    config := netchan.DefaultConfig()
    client, err := netchan.NewClient(config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // Create a channel
    ch := client.Chan("example")

    // Send a message
    ch.SendOp("Hello, NetChan!")

    // Receive a response
    response := ch.RecvOp()
    fmt.Printf("Received response: %v\n", response)
}
```

## API Reference

### Configuration

```go
// Config holds the configuration for the NetChan library.
type Config struct {
    // Network is the network type (e.g., "tcp", "udp")
    Network string
    // Address is the address to listen on or connect to
    Address string
    // BufferSize is the size of the channel buffer
    BufferSize int
    // Timeout is the timeout for network operations in milliseconds
    Timeout int
    // KeepAlive enables keep-alive for TCP connections
    KeepAlive bool
    // KeepAlivePeriod is the keep-alive period in seconds
    KeepAlivePeriod int
    // ReconnectDelay is the delay between reconnection attempts in milliseconds
    ReconnectDelay int
    // MaxReconnectAttempts is the maximum number of reconnection attempts
    MaxReconnectAttempts int
}

// DefaultConfig returns a default configuration for the NetChan library.
func DefaultConfig() *Config
```

### Server

```go
// NewServer creates a new NetChan server.
func NewServer(config *Config) (*NetChan, error)
```

### Client

```go
// NewClient creates a new NetChan client.
func NewClient(config *Config) (*NetChan, error)
```

### Channel

```go
// Chan creates a new network channel with the specified name.
func (nc *NetChan) Chan(name string) *Channel

// SendOp implements the send operator (ch <- value).
func (ch *Channel) SendOp(value interface{})

// RecvOp implements the receive operator (value := <-ch).
func (ch *Channel) RecvOp() interface{}

// Send sends a value to the channel.
func (ch *Channel) Send(value interface{}) error

// Receive receives a value from the channel.
func (ch *Channel) Receive() (interface{}, error)

// Close closes the channel.
func (ch *Channel) Close() error
```

## Advanced Usage

### Custom Serialization

NetChan uses the HybridPipe serialization system by default, which supports JSON, Protocol Buffers, and MessagePack. You can customize the serialization format by setting the appropriate configuration in the HybridPipe core.

### Error Handling

The `Send` and `Receive` methods return errors, allowing you to handle network failures and other issues. The `SendOp` and `RecvOp` methods log errors but do not return them, providing a more Go-like channel experience.

### Timeouts

You can configure timeouts for send and receive operations using the `Timeout` field in the configuration. This allows you to control how long operations will wait before failing.

### Reconnection

NetChan automatically attempts to reconnect to the server if the connection is lost. You can configure the reconnection behavior using the `ReconnectDelay` and `MaxReconnectAttempts` fields in the configuration.

## Examples

See the [examples](../golang/netchan/example) directory for more examples of using NetChan.

## Integration with HybridPipe

NetChan is fully integrated with the HybridPipe system, allowing you to use it as a messaging protocol alongside other protocols like NATS, Kafka, and RabbitMQ.

```go
package main

import (
    "fmt"
    "log"

    "hybridpipe.io/core"
    "hybridpipe.io/protocols/netchan"
)

func main() {
    // Create a NetChan router
    config := &core.NetChanConfig{
        ChannelBufferSize: 10,
        SendTimeout:       1000,
    }
    router := netchan.New(config)

    // Connect to the NetChan system
    if err := router.Connect(); err != nil {
        log.Fatalf("Failed to connect to NetChan system: %v", err)
    }
    defer router.Close()

    // Subscribe to a pipe
    if err := router.Subscribe("example", func(data []byte) error {
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
    if err := router.Dispatch("example", "Hello, NetChan!"); err != nil {
        log.Fatalf("Failed to dispatch message: %v", err)
    }
}
```
