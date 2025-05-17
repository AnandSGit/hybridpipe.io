# Redis Protocol

## Overview

Redis is an open-source, in-memory data structure store that can be used as a database, cache, and message broker. It supports various data structures such as strings, hashes, lists, sets, and more. Redis also provides a publish/subscribe messaging paradigm that HybridPipe leverages for its Redis protocol implementation.

## Key Features

- **In-Memory**: Fast data access with optional persistence
- **Publish/Subscribe**: Decouples publishers and subscribers
- **Pattern Matching**: Subscribe to channels using patterns
- **Atomic Operations**: Ensures data consistency
- **Clustering**: Horizontal scaling with Redis Cluster
- **Sentinel**: High availability with automatic failover
- **Lua Scripting**: Execute custom scripts on the server
- **Transactions**: Execute multiple commands atomically

## Implementation in HybridPipe

HybridPipe implements the Redis protocol using the publish/subscribe feature of Redis, providing a seamless integration with Redis servers.

### Go Implementation

The Go implementation uses the [github.com/go-redis/redis/v8](https://github.com/go-redis/redis/v8) package, which provides Go bindings for Redis.

### Rust Implementation

The Rust implementation uses the [redis](https://crates.io/crates/redis) crate, which provides Rust bindings for Redis.

## Configuration

### Go Implementation

```go
type RedisConfig struct {
    // Address is the Redis server address (e.g., "localhost:6379")
    Address string
    // Password is the Redis server password
    Password string
    // Database is the Redis database index
    Database int
    // PoolSize is the maximum number of connections in the pool
    PoolSize int
    // MinIdleConns is the minimum number of idle connections in the pool
    MinIdleConns int
    // DialTimeout is the dial timeout in milliseconds
    DialTimeout int
    // ReadTimeout is the read timeout in milliseconds
    ReadTimeout int
    // WriteTimeout is the write timeout in milliseconds
    WriteTimeout int
    // PoolTimeout is the pool timeout in milliseconds
    PoolTimeout int
    // IdleTimeout is the idle timeout in milliseconds
    IdleTimeout int
    // MaxRetries is the maximum number of retries
    MaxRetries int
    // MinRetryBackoff is the minimum retry backoff in milliseconds
    MinRetryBackoff int
    // MaxRetryBackoff is the maximum retry backoff in milliseconds
    MaxRetryBackoff int
}
```

### Rust Implementation

```rust
pub struct RedisConfig {
    // Address is the Redis server address (e.g., "redis://localhost:6379")
    pub address: String,
    // Password is the Redis server password
    pub password: Option<String>,
    // Database is the Redis database index
    pub database: i64,
    // Pool size is the maximum number of connections in the pool
    pub pool_size: u32,
    // Min idle conns is the minimum number of idle connections in the pool
    pub min_idle_conns: u32,
    // Dial timeout is the dial timeout in milliseconds
    pub dial_timeout: u64,
    // Read timeout is the read timeout in milliseconds
    pub read_timeout: u64,
    // Write timeout is the write timeout in milliseconds
    pub write_timeout: u64,
    // Pool timeout is the pool timeout in milliseconds
    pub pool_timeout: u64,
    // Idle timeout is the idle timeout in milliseconds
    pub idle_timeout: u64,
    // Max retries is the maximum number of retries
    pub max_retries: u32,
    // Min retry backoff is the minimum retry backoff in milliseconds
    pub min_retry_backoff: u64,
    // Max retry backoff is the maximum retry backoff in milliseconds
    pub max_retry_backoff: u64,
}
```

## Usage

### Go Implementation

```go
// Create a Redis router with default configuration
router := redis.New(nil)

// Connect to Redis
if err := router.Connect(); err != nil {
    log.Fatalf("Failed to connect to Redis: %v", err)
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
if err := router.Dispatch("greetings", "Hello, Redis!"); err != nil {
    log.Fatalf("Failed to dispatch message: %v", err)
}
```

### Rust Implementation

```rust
// Deploy a Redis router
let router = deploy_router(BrokerType::REDIS)?;

// Connect to Redis
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
let message = "Hello, Redis!".as_bytes().to_vec();
router.dispatch("greetings", Box::new(message)).await?;
```

## Publish/Subscribe Pattern

Redis implements the publish/subscribe messaging paradigm, where publishers send messages to channels without knowledge of which subscribers (if any) will receive them. Subscribers express interest in one or more channels and only receive messages that are of interest, without knowledge of which publishers (if any) there are.

In HybridPipe, the pipe name is used as the Redis channel name.

## Pattern Matching

Redis allows subscribers to subscribe to channels using patterns with glob-style wildcards:

- `?` matches any single character
- `*` matches any sequence of characters
- `[...]` matches any character in the brackets
- `[^...]` matches any character not in the brackets

For example, `user.*` would match `user.created`, `user.updated`, etc.

HybridPipe does not currently support pattern matching in its Redis implementation, but this could be added in the future.

## Performance Considerations

Redis is designed for high performance. Here are some tips for optimizing Redis performance:

1. **Connection Pooling**: Use connection pooling to avoid the overhead of creating new connections
2. **Pipelining**: Use pipelining to send multiple commands in a single round-trip
3. **Lua Scripting**: Use Lua scripts to execute multiple commands atomically
4. **Memory Management**: Monitor Redis memory usage and configure appropriate limits

## Limitations

1. **No Message Persistence**: Redis publish/subscribe does not persist messages. If a subscriber is not connected when a message is published, it will miss the message
2. **No Message Acknowledgment**: Redis publish/subscribe does not provide a built-in acknowledgment mechanism
3. **No Message Ordering**: Redis publish/subscribe does not guarantee message ordering across different channels

## Examples

See the [examples](../../golang/protocols/redis/example) directory for more examples of using Redis with HybridPipe.io.

## References

- [Redis Official Website](https://redis.io/)
- [Redis Pub/Sub Documentation](https://redis.io/topics/pubsub)
- [Go Redis Client](https://github.com/go-redis/redis)
- [Rust Redis Client](https://crates.io/crates/redis)
