# HybridPipe Core Package

The core package provides the fundamental interfaces and functionality for the HybridPipe messaging system. It defines the core abstractions that all protocol implementations must adhere to.

## Components

### Interface

The `HybridPipe` interface defines the contract that all messaging implementations must satisfy:

```go
type HybridPipe interface {
    // Connect establishes a connection to the messaging system.
    Connect() error

    // Dispatch sends a message to the specified pipe.
    Dispatch(pipe string, data any) error

    // Accept subscribes to messages from the specified pipe and processes them with the provided function.
    Accept(pipe string, fn Process) error

    // Remove unsubscribes from the specified pipe.
    Remove(pipe string) error

    // Close terminates the connection to the messaging system.
    Close()
}
```

### Registry

The registry system manages the registration and instantiation of protocol implementations:

- `RegisterRouter`: Registers a protocol implementation with the system
- `DeployRouter`: Creates and connects to a specified messaging system

### Configuration

The configuration system handles loading and processing configuration for all protocols:

- `ReadConfig`: Reads configuration from files or environment variables
- `HPipeConfig`: Global configuration instance

### Serialization

The serialization system provides a unified way to encode and decode messages:

- `Encode`: Encodes data for transmission
- `Decode`: Decodes received data
- `SerializationOptions`: Configures serialization behavior

## Usage

Protocol implementations should import the core package and register themselves:

```go
import (
    "reflect"
    "hybridpipe.io/core"
)

// Register the protocol implementation
func init() {
    core.RegisterRouter(core.PROTOCOL_TYPE, reflect.TypeOf((*ProtocolImplementation)(nil)).Elem())
}
```

Applications should use the main `hybridpipe` package rather than importing the core package directly.
