# MQTT Protocol

## Overview

MQTT (Message Queuing Telemetry Transport) is a lightweight, publish-subscribe network protocol designed for constrained devices and low-bandwidth, high-latency, or unreliable networks. It is ideal for Internet of Things (IoT) applications and other scenarios where a small code footprint is required or network bandwidth is limited.

## Key Features

- **Lightweight**: Minimal protocol overhead
- **Publish-Subscribe Pattern**: Decouples publishers and subscribers
- **Quality of Service Levels**: Three levels of message delivery guarantees
- **Retained Messages**: Last message on a topic is stored and sent to new subscribers
- **Last Will and Testament**: Message sent when a client disconnects unexpectedly
- **Clean/Persistent Sessions**: Option to store subscription and message state
- **Security**: TLS/SSL support and username/password authentication

## Implementation in HybridPipe

HybridPipe implements the MQTT protocol using the MQTT 3.1.1 standard, providing a seamless integration with MQTT brokers.

### Go Implementation

The Go implementation uses the [github.com/eclipse/paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang) package, which provides Go bindings for MQTT.

### Rust Implementation

The Rust implementation uses the [paho-mqtt](https://crates.io/crates/paho-mqtt) crate, which provides Rust bindings for MQTT.

## Configuration

### Go Implementation

```go
type MQTTConfig struct {
    // Broker is the MQTT broker URL (e.g., "tcp://localhost:1883")
    Broker string
    // ClientID is the MQTT client ID
    ClientID string
    // Username is the MQTT username
    Username string
    // Password is the MQTT password
    Password string
    // QoS is the MQTT quality of service level (0, 1, or 2)
    QoS int
    // CleanSession indicates whether to use a clean session
    CleanSession bool
    // KeepAlive is the keep-alive interval in seconds
    KeepAlive int
    // ConnectTimeout is the connection timeout in seconds
    ConnectTimeout int
    // ReconnectDelay is the delay between reconnection attempts in milliseconds
    ReconnectDelay int
    // MaxReconnectAttempts is the maximum number of reconnection attempts
    MaxReconnectAttempts int
}
```

### Rust Implementation

```rust
pub struct MQTTConfig {
    // Broker is the MQTT broker URL (e.g., "tcp://localhost:1883")
    pub broker: String,
    // Client ID is the MQTT client ID
    pub client_id: String,
    // Username is the MQTT username
    pub username: Option<String>,
    // Password is the MQTT password
    pub password: Option<String>,
    // QoS is the MQTT quality of service level (0, 1, or 2)
    pub qos: i32,
    // Clean session indicates whether to use a clean session
    pub clean_session: bool,
    // Keep alive is the keep-alive interval in seconds
    pub keep_alive: u64,
    // Connect timeout is the connection timeout in seconds
    pub connect_timeout: u64,
    // Reconnect delay is the delay between reconnection attempts in milliseconds
    pub reconnect_delay: u64,
    // Max reconnect attempts is the maximum number of reconnection attempts
    pub max_reconnect_attempts: u32,
}
```

## Usage

### Go Implementation

```go
// Create an MQTT router with default configuration
router := mqtt.New(nil)

// Connect to MQTT
if err := router.Connect(); err != nil {
    log.Fatalf("Failed to connect to MQTT: %v", err)
}
defer router.Close()

// Subscribe to a pipe
if err := router.Subscribe("greetings", func(data []byte) error {
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
if err := router.Dispatch("greetings", "Hello, MQTT!"); err != nil {
    log.Fatalf("Failed to dispatch message: %v", err)
}
```

### Rust Implementation

```rust
// Deploy an MQTT router
let router = deploy_router(BrokerType::MQTT)?;

// Connect to MQTT
router.connect().await?;

// Subscribe to a pipe
let callback: Process = Box::new(|data| {
    Box::pin(async move {
        info!("Received message: {:?}", data);
        Ok(())
    })
});
router.subscribe("greetings", callback).await?;

// Dispatch a message
let message = "Hello, MQTT!".as_bytes().to_vec();
router.dispatch("greetings", Box::new(message)).await?;
```

## Quality of Service Levels

MQTT provides three quality of service (QoS) levels for message delivery:

1. **QoS 0 (At Most Once)**: The message is delivered at most once, or it is not delivered at all. This is the fastest but least reliable level.
2. **QoS 1 (At Least Once)**: The message is delivered at least once, but it may be delivered multiple times. This provides more reliability but may result in duplicate messages.
3. **QoS 2 (Exactly Once)**: The message is delivered exactly once. This is the most reliable but slowest level.

In HybridPipe, the QoS level can be configured in the MQTT configuration.

## Performance Considerations

MQTT is designed for constrained devices and networks. Here are some tips for optimizing MQTT performance:

1. **Message Size**: Keep messages small to minimize bandwidth usage
2. **QoS Level**: Use the lowest QoS level that meets your reliability requirements
3. **Clean Session**: Use clean sessions for temporary clients and persistent sessions for clients that need to receive missed messages
4. **Topic Design**: Design topics carefully to minimize the number of subscriptions

## Limitations

1. **Message Size**: MQTT is designed for small messages, not for large data transfers
2. **No Request-Reply Pattern**: MQTT does not provide a built-in request-reply pattern
3. **Limited Filtering**: MQTT only supports topic-based filtering, not content-based filtering

## Examples

See the [examples](../../golang/protocols/mqtt/example) directory for more examples of using MQTT with HybridPipe.io.

## References

- [MQTT Official Website](https://mqtt.org/)
- [MQTT 3.1.1 Specification](https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html)
- [Go MQTT Client](https://github.com/eclipse/paho.mqtt.golang)
- [Rust MQTT Client](https://crates.io/crates/paho-mqtt)
