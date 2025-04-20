# NetChan: Network Channels for Go

NetChan is a Go library that extends the concept of Go channels across network boundaries, allowing different processes to communicate using the familiar channel semantics. It is fully integrated with the HybridPipe messaging system, providing a consistent interface for inter-process communication.

## Features

- **Go Channel Semantics**: Use the same familiar send/receive operations as native Go channels
- **Native Channel Syntax**: Optional support for using Go's native channel operators (`<-` and `->`) with network channels
- **Type Safety**: Maintains Go's type safety across the network
- **Buffered and Unbuffered Channels**: Support for both buffered and unbuffered channels
- **Automatic Reconnection**: Handles network failures and reconnects automatically
- **Secure Communication**: Optional TLS encryption for secure communication
- **Compression**: Optional compression for efficient network usage
- **Select Support**: Compatible with Go's select statement for handling multiple channels

## Installation

```bash
go get github.com/AnandSGit/hybridpipe.io/netchan
```

## HybridPipe Integration

NetChan is fully integrated with the HybridPipe messaging system, allowing you to use it alongside other messaging protocols with a consistent interface:

```go
import (
    "fmt"
    "log"

    hp "github.com/AnandSGit/hybridpipe.io"
)

func main() {
    // Create a NetChan router through HybridPipe
    router, err := hp.DeployRouter(hp.NETCHAN)
    if err != nil {
        log.Fatalf("Failed to deploy NetChan router: %v", err)
    }
    defer router.Close()

    // Subscribe to a channel
    router.Accept("my-channel", func(data interface{}) {
        fmt.Printf("Received: %v\n", data)
    })

    // Send a message
    router.Dispatch("my-channel", "Hello, NetChan!")

    // Keep the application running
    select {}
}
```

This allows you to easily switch between NetChan and other messaging systems without changing your application code.

## Using Native Channel Syntax

NetChan provides a `ChannelWrapper` that allows you to use Go's native channel operators (`<-` and `->`) with network channels:

```go
// Create a NetChan
netChan, err := netchan.Dial[string]("localhost:8080", "")
if err != nil {
    log.Fatalf("Failed to connect: %v", err)
}

// Create a channel wrapper
wrapper := netchan.NewChannelWrapper(netChan, 10)
if err := wrapper.Initialize(); err != nil {
    log.Fatalf("Failed to initialize wrapper: %v", err)
}

// Get send and receive channels
sendChan := wrapper.SendChan()
recvChan := wrapper.RecvChan()

// Send using native channel syntax
sendChan <- "Hello, NetChan!"

// Receive using native channel syntax
msg, ok := <-recvChan
if ok {
    fmt.Printf("Received: %s\n", msg)
}

// Use with select statement
select {
case sendChan <- "Another message":
    fmt.Println("Sent message")
case msg, ok := <-recvChan:
    if ok {
        fmt.Printf("Received: %s\n", msg)
    }
case <-time.After(5 * time.Second):
    fmt.Println("Timeout")
}
```

## Quick Start

### Receiver

```go
package main

import (
    "fmt"
    "log"

    "github.com/AnandSGit/hybridpipe.io/netchan"
)

func main() {
    // Create a receiver for int values
    receiver, err := netchan.Listen[int](":8080", 0)
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    defer receiver.Close()

    fmt.Println("Listening on :8080")

    // Receive values
    for {
        value, ok, err := receiver.Recv()
        if err != nil {
            log.Printf("Error: %v", err)
            continue
        }
        if !ok {
            log.Println("Channel closed")
            break
        }
        fmt.Printf("Received: %d\n", value)
    }
}
```

### Sender

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/AnandSGit/hybridpipe.io/netchan"
)

func main() {
    // Connect to the receiver
    sender, err := netchan.Dial[int]("localhost:8080", 0)
    if err != nil {
        log.Fatalf("Failed to dial: %v", err)
    }
    defer sender.Close()

    // Send values
    for i := 0; i < 10; i++ {
        if err := sender.Send(i); err != nil {
            log.Printf("Error: %v", err)
            continue
        }
        fmt.Printf("Sent: %d\n", i)
        time.Sleep(time.Second)
    }
}
```

## API Reference

### Creating Channels

```go
// Create a new unbuffered channel
ch := netchan.Make(T{}) // T is the type of values to be sent

// Create a new buffered channel with capacity
ch := netchan.MakeBuffered(T{}, capacity)

// Listen for incoming channel connections
receiver := netchan.Listen[T](address)

// Connect to a channel
sender := netchan.Dial[T](address)
```

### Channel Operations

```go
// Send a value (blocks if the channel is full)
err := ch.Send(value)

// Try to send a value (non-blocking)
ok, err := ch.TrySend(value)

// Receive a value (blocks if the channel is empty)
value, ok, err := ch.Recv()

// Try to receive a value (non-blocking)
value, ok, err := ch.TryRecv()

// Close a channel
err := ch.Close()

// Check if a channel is closed
closed := ch.IsClosed()
```

### Select Operations

```go
// Create a new select case for sending
sendCase := netchan.SendCase(ch, value)

// Create a new select case for receiving
recvCase := netchan.RecvCase(ch)

// Create a new select case for a timeout
timeoutCase := netchan.TimeoutCase(duration)

// Perform a select operation
chosen, recv, recvOK, err := netchan.Select([]netchan.Case{sendCase, recvCase, timeoutCase})
```

### Configuration

```go
// Configure a channel with options
ch := netchan.Make(T{}, netchan.WithBufferSize(10), netchan.WithTimeout(5*time.Second))

// Available options
netchan.WithBufferSize(n)           // Set the channel buffer size
netchan.WithTimeout(duration)       // Set operation timeout
netchan.WithReconnect(true)         // Enable automatic reconnection
netchan.WithReconnectBackoff(policy) // Set reconnection backoff policy
netchan.WithCompression(level)      // Enable compression
netchan.WithTLS(config)             // Enable TLS encryption
```

## Advanced Usage

### Using Select

```go
func main() {
    ch1 := netchan.Dial[int]("localhost:8080")
    ch2 := netchan.Dial[string]("localhost:8081")

    for {
        // Create select cases
        case1 := netchan.RecvCase(ch1)
        case2 := netchan.RecvCase(ch2)
        timeout := netchan.TimeoutCase(5 * time.Second)

        // Perform select
        chosen, recv, recvOK, err := netchan.Select([]netchan.Case{case1, case2, timeout})

        if err != nil {
            log.Printf("Select error: %v", err)
            continue
        }

        switch chosen {
        case 0: // ch1
            if recvOK {
                value := recv.(int)
                log.Printf("Received from ch1: %d", value)
            } else {
                log.Println("ch1 closed")
            }
        case 1: // ch2
            if recvOK {
                value := recv.(string)
                log.Printf("Received from ch2: %s", value)
            } else {
                log.Println("ch2 closed")
            }
        case 2: // timeout
            log.Println("Select timed out")
        }
    }
}
```

### Secure Communication

```go
// Load certificates
cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
if err != nil {
    log.Fatalf("Failed to load certificates: %v", err)
}

// Create TLS config
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    MinVersion:   tls.VersionTLS12,
}

// Create a secure channel
ch := netchan.Make(0, netchan.WithTLS(tlsConfig))
```

## Performance Considerations

- **Message Size**: Large messages may impact performance. Consider chunking large data.
- **Buffer Size**: Choose an appropriate buffer size based on your message rate and processing speed.
- **Compression**: Enable compression for large messages or when bandwidth is limited.
- **Reconnection**: Adjust reconnection parameters based on your network reliability.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

- Inspired by the original `netchan` package from the Go standard library (now deprecated)
- Built on top of the Go standard library's networking and serialization capabilities
