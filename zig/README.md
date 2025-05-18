# HybridPipe - Zig Implementation

This is the Zig implementation of the HybridPipe messaging middleware system. HybridPipe provides a unified interface for various messaging protocols, allowing applications to switch between different messaging systems without changing code.

## Features

- **Core Architecture**: Unified interface for all messaging protocols
- **Protocol Implementations**:
  - TCP/IP: Direct TCP/IP communication
  - Mock: In-memory mock for testing
- **Serialization**: Binary and JSON serialization
- **Middleware**: Extensible middleware system for logging and monitoring
- **Monitoring**: Message statistics and metrics collection

## Requirements

- Zig 0.11.0 or later

## Building

To build the library and examples:

```bash
cd zig
zig build
```

## Running Examples

### Mock Example

The Mock example demonstrates using the in-memory mock protocol for testing:

```bash
zig build run-mock
```

### TCP Example

The TCP example demonstrates using the TCP protocol for communication:

```bash
zig build run-tcp
```

Note: The TCP example requires a TCP server running on localhost:8080. You can use netcat to create a simple server:

```bash
nc -l 8080
```

## Usage

### Basic Usage

```zig
const std = @import("std");
const hybridpipe = @import("hybridpipe");

pub fn main() !void {
    // Initialize the allocator
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();
    
    // Initialize the registry and configuration
    hybridpipe.registry.initRegistry(allocator);
    hybridpipe.config.initConfig(allocator);
    
    // Initialize the library
    hybridpipe.init();
    
    // Deploy a router
    var router = try hybridpipe.deployRouter(hybridpipe.MOCK);
    defer _ = router.disconnect();
    
    // Subscribe to a pipe
    try router.subscribe("example", messageHandler);
    
    // Send a message
    try router.dispatch("example", "Hello, HybridPipe!");
}

fn messageHandler(data: []const u8) hybridpipe.Error!void {
    std.debug.print("Received message: {s}\n", .{data});
}
```

### Using Middleware

```zig
// Create a logging middleware
var logging_middleware = hybridpipe.middleware.LoggingMiddleware.init(.Info);

// Use the middleware
try logging_middleware.middleware.beforeDispatch(router, "example", "Hello");
try router.dispatch("example", "Hello");
try logging_middleware.middleware.afterDispatch(router, "example", "Hello");
```

### Using Monitoring

```zig
// Create a monitoring middleware
var monitoring = hybridpipe.monitoring.MonitoringMiddleware.init(allocator);
defer monitoring.deinit();

// Use the middleware
try monitoring.middleware.beforeDispatch(router, "example", "Hello");
try router.dispatch("example", "Hello");

// Get statistics
var stats = try monitoring.getStats("example");
std.debug.print("Messages sent: {d}\n", .{stats.messages_sent});
```

## Architecture

HybridPipe uses a modular architecture with the following components:

1. **Core Interface**: The `HybridPipe` interface that all messaging implementations must satisfy
2. **Protocol Implementations**: Concrete implementations for each messaging system
3. **Registry System**: Manages registration and instantiation of protocol implementations
4. **Serialization System**: Handles encoding/decoding of messages
5. **Middleware System**: Provides hooks for extending functionality
6. **Monitoring System**: Collects metrics and statistics

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgements

This is a Zig implementation of the HybridPipe system, originally developed in Go and Rust.
