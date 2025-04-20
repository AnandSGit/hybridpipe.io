# HybridPipe Mock Implementation

This document describes the mock implementation of the HybridPipe interface, which allows for testing and development without requiring actual messaging middleware systems.

## Overview

The mock implementation (`MockPacket`) provides an in-memory simulation of a messaging system, allowing developers to:

1. Test applications that use HybridPipe without setting up actual messaging servers
2. Develop and debug code offline
3. Create automated tests that don't depend on external services
4. Simulate error conditions and network delays

## Features

- **In-Memory Message Storage**: Messages are stored in memory instead of being sent to actual servers
- **Subscription Simulation**: Simulates the subscription and message delivery process
- **Error Simulation**: Can be configured to simulate connection or operation failures
- **Delay Simulation**: Can simulate network latency
- **Message Inspection**: Provides methods to inspect stored messages
- **Statistics**: Tracks message and subscription counts

## Usage

### Basic Usage

```go
// Import the package
import hp "hybridpipe.io"

// Create a mock router
router, err := hp.DeployRouter(hp.MOCK)
if err != nil {
    log.Fatalf("Failed to create mock router: %v", err)
}
defer router.Close()

// Use it like any other HybridPipe implementation
router.Accept("my.topic", func(data interface{}) {
    fmt.Printf("Received: %v\n", data)
})

router.Dispatch("my.topic", "Hello, Mock!")
```

### Advanced Usage

For more advanced testing scenarios, you can cast the router to a `MockPacket` to access additional methods:

```go
// Create a mock router
router, _ := hp.DeployRouter(hp.MOCK)
defer router.Close()

// Cast to MockPacket to access additional methods
mockPacket, ok := router.(*hp.MockPacket)
if !ok {
    log.Fatalf("Failed to cast to MockPacket")
}

// Simulate errors
mockPacket.SetErrorMode(true)
err := router.Dispatch("my.topic", "This should fail")
fmt.Printf("Error: %v\n", err) // Should print an error

// Reset error mode
mockPacket.SetErrorMode(false)

// Simulate network delay
mockPacket.SetDelay(100 * time.Millisecond)
router.Dispatch("my.topic", "This will be delayed")

// Get message statistics
fmt.Printf("Messages sent: %d\n", mockPacket.GetMessageCount())
fmt.Printf("Subscriptions: %d\n", mockPacket.GetSubscriptionCount())

// Inspect messages
messages := mockPacket.GetMessages("my.topic")
for i, msg := range messages {
    fmt.Printf("Message %d: %v\n", i, msg)
}

// Clear messages
mockPacket.ClearMessages("my.topic")

// Reset the mock to initial state
mockPacket.Reset()
```

## Mock-Specific Methods

The `MockPacket` struct provides several methods that are not part of the `HybridPipe` interface:

| Method | Description |
|--------|-------------|
| `NewMockPacket(name string)` | Creates a new named mock packet |
| `SetErrorMode(enable bool)` | Enables or disables error simulation |
| `SetDelay(delay time.Duration)` | Sets the simulated network latency |
| `GetMessageCount()` | Returns the number of messages sent |
| `GetSubscriptionCount()` | Returns the number of subscriptions |
| `GetMessages(pipe string)` | Returns all messages for a pipe |
| `ClearMessages(pipe string)` | Removes all messages from a pipe |
| `Reset()` | Resets the mock to its initial state |

## Testing with Mocks

The mock implementation is particularly useful for unit testing. Here's an example of how to use it in tests:

```go
func TestMyFunction(t *testing.T) {
    // Create a mock router
    mock, _ := hp.DeployRouter(hp.MOCK)
    defer mock.Close()
    
    // Get the mock packet for additional testing
    mockPacket := mock.(*hp.MockPacket)
    
    // Set up a test subscription
    received := make(chan bool, 1)
    mock.Accept("test.topic", func(data interface{}) {
        // Verify the received data
        if str, ok := data.(string); ok && str == "test message" {
            received <- true
        }
    })
    
    // Call the function being tested
    MyFunction(mock)
    
    // Verify that a message was sent
    if count := mockPacket.GetMessageCount(); count != 1 {
        t.Errorf("Expected 1 message, got %d", count)
    }
    
    // Verify that the message was received by the handler
    select {
    case <-received:
        // Success
    case <-time.After(1 * time.Second):
        t.Fatal("Timed out waiting for message")
    }
}
```

## Benefits of Using Mocks

1. **Faster Tests**: Tests run much faster without connecting to actual servers
2. **Deterministic**: Tests are more reliable and deterministic
3. **No External Dependencies**: Tests don't require external services to be running
4. **Offline Development**: Can develop and test without internet access
5. **CI/CD Friendly**: Easier to integrate into continuous integration pipelines

## Limitations

1. **Behavior Differences**: The mock implementation may not perfectly replicate the behavior of actual messaging systems
2. **Protocol-Specific Features**: Protocol-specific features are not simulated
3. **Performance Characteristics**: Performance characteristics will differ from actual messaging systems

## Conclusion

The mock implementation provides a convenient way to test and develop applications that use HybridPipe without requiring actual messaging middleware systems. It's particularly useful for unit testing, offline development, and CI/CD pipelines.
