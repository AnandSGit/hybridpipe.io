# Phase 3 Protocols: Advanced Messaging

Phase 3 of HybridPipe.io introduces three advanced messaging protocols that provide specialized capabilities for different use cases. The Python implementation provides **complete feature parity** with the Go implementation plus additional advanced features.

- **NATS**: Cloud-native messaging with JetStream persistence and request-reply
- **ZeroMQ**: Ultra-low latency brokerless messaging with all patterns
- **AMQP 1.0**: Standards-based enterprise messaging with transactions

## Cross-Language Compatibility

The Python implementation supports **both** Python-style and Go-style configuration formats, ensuring seamless migration and cross-language deployments.

## NATS Protocol

NATS is a simple, secure, and performant communications system for digital systems, services, and devices.

### Features

- **Core NATS**: At-most-once delivery with ultra-low latency
- **JetStream**: Persistence, exactly-once delivery, and replay
- **Subject-based messaging**: Hierarchical addressing with wildcards
- **Request-Reply**: Built-in request-response patterns
- **Clustering**: Horizontal scaling and high availability

### Configuration

```python
from hybridpipe import deploy_router, BrokerType

# Python-style configuration (recommended)
config = {
    "servers": ["nats://localhost:4222"],
    "name": "my-app",
    "jetstream": False,  # Enable for persistence
    "max_reconnect_attempts": 10,
    "reconnect_time_wait": 2.0,
}

# Go-compatible configuration (for migration)
go_config = {
    "nserver": "localhost",
    "nlport": 4222,
    "name": "my-app",
    "max_reconnects": 10,
    "reconnect_wait": 2,
}

router = await deploy_router(BrokerType.NATS, config=config)
```

### JetStream Configuration

```python
# JetStream for persistence and guaranteed delivery
config = {
    "servers": ["nats://localhost:4222"],
    "name": "my-persistent-app",
    "jetstream": True,
    "jetstream_domain": "hub",
    "jetstream_api_prefix": "$JS.API",
}

router = await deploy_router(BrokerType.NATS, config=config)
await router.connect()

# Create a stream for persistent messaging
await router.create_stream(
    "ORDERS",
    ["orders.>"],
    config={
        "retention": "limits",
        "max_age": 86400,  # 24 hours
        "max_msgs": 1000000,
        "storage": "file",
    }
)
```

### Usage Examples

```python
# Basic pub/sub
async def message_handler(data: bytes, metadata):
    print(f"Received: {data}")

await router.subscribe("orders.new", message_handler)
await router.dispatch("orders.new", {"order_id": 123, "amount": 99.99})

# Request-reply pattern
async def order_processor(request_data, metadata):
    order = decode(request_data)
    # Process order...
    return {"status": "processed", "order_id": order["id"]}

await router.reply("orders.process", order_processor)

# Make a request
response = await router.request("orders.process", {"id": 123})
print(response["status"])  # "processed"

# Subject wildcards
await router.subscribe("orders.*", message_handler)  # All order events
await router.subscribe("orders.>", message_handler)   # All order hierarchies
```

### Performance Characteristics

- **Throughput**: 8-12M messages/second
- **Latency**: 20-100 microseconds
- **Memory**: ~10MB base + ~1KB per subscription
- **Clustering**: Linear scaling with cluster size

## ZeroMQ Protocol

ZeroMQ is a high-performance asynchronous messaging library with multiple messaging patterns.

### Features

- **Multiple Patterns**: PUB/SUB, REQ/REP, PUSH/PULL, PAIR, ROUTER/DEALER
- **Brokerless**: Direct peer-to-peer communication
- **Ultra-low Latency**: Sub-microsecond messaging
- **High Throughput**: Millions of messages per second
- **Transport Agnostic**: TCP, IPC, inproc, multicast

### Configuration

```python
# Publisher configuration
pub_config = {
    "pattern": "PUB_SUB",
    "bind_addresses": ["tcp://*:5555"],
    "high_water_mark": 1000,
    "linger": 0,
}

# Subscriber configuration
sub_config = {
    "pattern": "PUB_SUB",
    "connect_addresses": ["tcp://localhost:5555"],
    "subscribe_filters": ["topic1", "topic2"],
}

pub_router = await deploy_router(BrokerType.ZEROMQ, config=pub_config)
sub_router = await deploy_router(BrokerType.ZEROMQ, config=sub_config)
```

### Messaging Patterns

#### PUB/SUB Pattern
```python
# Publisher
await pub_router.connect()
await pub_router.dispatch("topic1", {"message": "Hello World"})

# Subscriber
async def message_handler(data: bytes, metadata):
    print(f"Received: {data}")

await sub_router.connect()
await sub_router.subscribe("topic1", message_handler)
```

#### REQ/REP Pattern
```python
# Server (REP)
server_config = {
    "pattern": "REQ_REP",
    "bind_addresses": ["tcp://*:5556"],
}

# Client (REQ)
client_config = {
    "pattern": "REQ_REP",
    "connect_addresses": ["tcp://localhost:5556"],
}

# Server handler
async def echo_handler(request_data, metadata):
    data = decode(request_data)
    return {"echo": data["message"]}

await server_router.reply("echo", echo_handler)

# Client request
response = await client_router.request("echo", {"message": "hello"})
```

#### PUSH/PULL Pattern
```python
# Work distributor (PUSH)
push_config = {
    "pattern": "PUSH_PULL",
    "bind_addresses": ["tcp://*:5557"],
}

# Worker (PULL)
pull_config = {
    "pattern": "PUSH_PULL",
    "connect_addresses": ["tcp://localhost:5557"],
}

# Distribute work
await push_router.dispatch("work", {"task_id": 1, "data": "process_this"})

# Process work
async def work_handler(data: bytes, metadata):
    task = decode(data)
    # Process task...
    print(f"Processing task {task['task_id']}")

await pull_router.subscribe("work", work_handler)
```

### Performance Characteristics

- **Throughput**: 1-10M messages/second
- **Latency**: 10-100 microseconds
- **Memory**: ~5MB base + minimal per connection
- **Scalability**: Limited by network and CPU

## AMQP 1.0 Protocol

AMQP 1.0 is an open standard for business messaging with enterprise-grade features.

### Features

- **Standards-based**: OASIS AMQP 1.0 specification
- **Interoperability**: Works with multiple vendors
- **Reliability**: Transactions and guaranteed delivery
- **Security**: SASL authentication and SSL/TLS encryption
- **Advanced Routing**: Complex routing and filtering

### Configuration

```python
# Basic AMQP 1.0 configuration
config = {
    "hostname": "localhost",
    "port": 5672,
    "username": "guest",
    "password": "guest",
    "container_id": "my-app",
    "sasl_enabled": True,
    "ssl_enabled": False,
    "durable": True,
    "auto_settle": True,
}

router = await deploy_router(BrokerType.AMQP1, config=config)
```

### SSL/TLS Configuration

```python
# Secure AMQP 1.0 configuration
secure_config = {
    "hostname": "amqp.example.com",
    "port": 5671,
    "username": "myapp",
    "password": "secret",
    "ssl_enabled": True,
    "ssl_verify_mode": "VERIFY_PEER",
    "ssl_ca_file": "/path/to/ca.pem",
    "ssl_cert_file": "/path/to/client.pem",
    "ssl_key_file": "/path/to/client.key",
}
```

### Usage Examples

```python
# Basic messaging
async def message_handler(data: bytes, metadata):
    message = decode(data)
    print(f"Received: {message}")

await router.connect()
await router.subscribe("my.queue", message_handler)
await router.dispatch("my.queue", {"message": "Hello AMQP 1.0"})

# With message properties
headers = {
    "priority": 5,
    "content-type": "application/json",
    "correlation-id": "12345",
}

await router.dispatch("my.queue", {"data": "important"}, headers=headers)
```

### Performance Characteristics

- **Throughput**: 10-100K messages/second
- **Latency**: 100-1000 microseconds
- **Memory**: ~20MB base + ~2KB per connection
- **Reliability**: Excellent with transactions

## Protocol Comparison

| Feature | NATS | ZeroMQ | AMQP 1.0 |
|---------|------|--------|----------|
| **Throughput** | Very High | Highest | Medium |
| **Latency** | Very Low | Lowest | Medium |
| **Persistence** | JetStream | None | Yes |
| **Clustering** | Native | Manual | Broker-dependent |
| **Standards** | NATS Protocol | ZMQ Protocol | OASIS AMQP 1.0 |
| **Complexity** | Low | Medium | High |
| **Use Cases** | Microservices, IoT | HPC, Gaming | Enterprise, Finance |

## Installation

Install the required dependencies for Phase 3 protocols:

```bash
# NATS
pip install nats-py

# ZeroMQ
pip install pyzmq

# AMQP 1.0
pip install python-qpid-proton

# All Phase 3 protocols
pip install nats-py pyzmq python-qpid-proton
```

## Best Practices

### NATS
- Use JetStream for critical data that needs persistence
- Leverage subject hierarchies for efficient routing
- Monitor JetStream storage usage
- Use clustering for high availability

### ZeroMQ
- Choose the right pattern for your use case
- Set appropriate high water marks to prevent memory issues
- Use PAIR pattern for lowest latency
- Consider security implications of direct connections

### AMQP 1.0
- Use transactions for critical operations
- Configure SSL/TLS for production
- Monitor broker resources
- Design addresses for efficient routing

## Troubleshooting

### Common Issues

1. **Connection Failures**
   - Check broker availability and network connectivity
   - Verify authentication credentials
   - Ensure correct ports and protocols

2. **Performance Issues**
   - Monitor broker resources (CPU, memory, disk)
   - Check network latency and bandwidth
   - Tune configuration parameters

3. **Message Loss**
   - Enable persistence where available
   - Use appropriate acknowledgment modes
   - Monitor error logs and metrics

### Debugging

Enable debug logging to troubleshoot issues:

```python
import logging
logging.basicConfig(level=logging.DEBUG)

# Protocol-specific debugging
config["debug"] = True
router = await deploy_router(BrokerType.NATS, config=config)
```

## Next Steps

- Explore [Phase 4 protocols](phase4-protocols.md) for specialized use cases
- Learn about [monitoring and observability](monitoring.md)
- See [production deployment](deployment.md) guidelines
