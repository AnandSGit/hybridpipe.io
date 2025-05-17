# Message Tracing Documentation

HybridPipe.io provides comprehensive message tracing capabilities that allow you to track messages as they flow through your system. This document describes how to use the tracing functionality.

## Overview

Message tracing in HybridPipe.io allows you to:

- Track messages as they flow through your system
- Measure message processing time
- Identify bottlenecks and performance issues
- Debug message delivery problems
- Monitor system health

The tracing system consists of two main components:

1. **Tracer**: Collects and stores trace information
2. **TracingMiddleware**: Wraps a HybridPipe router to add tracing capabilities

## Tracing Architecture

The tracing system is designed to be lightweight and non-intrusive. It works by wrapping a HybridPipe router with a TracingMiddleware, which intercepts all messages and adds tracing information.

```
┌─────────────┐     ┌───────────────────┐     ┌─────────────┐
│ Application │────▶│ TracingMiddleware │────▶│ HybridPipe  │
└─────────────┘     └───────────────────┘     └─────────────┘
                            │
                            ▼
                     ┌─────────────┐
                     │   Tracer    │
                     └─────────────┘
```

## Using Tracing

### Go Implementation

```go
package main

import (
    "fmt"
    "log"

    "hybridpipe.io/core"
    "hybridpipe.io/monitoring"
    "hybridpipe.io/protocols/nats"
)

func main() {
    // Create a tracer
    tracer := monitoring.NewMessageTracer(1000)

    // Create a router
    router := nats.New(nil)

    // Create a tracing middleware
    tracingRouter := monitoring.NewTracingMiddleware(router, tracer, "NATS")

    // Connect to the router
    if err := tracingRouter.Connect(); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer tracingRouter.Close()

    // Subscribe to a pipe
    if err := tracingRouter.Subscribe("test", func(data []byte) error {
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
    if err := tracingRouter.Dispatch("test", "Hello, Tracing!"); err != nil {
        log.Fatalf("Failed to dispatch: %v", err)
    }

    // Get the traces
    traces := tracer.GetTraces()
    for _, trace := range traces {
        fmt.Println(trace)
    }
}
```

### Rust Implementation

```rust
use hybridpipe::{
    deploy_router, BrokerType, InMemoryTracer, Process, Result, Trace, TraceStatus, TracingMiddleware,
};
use log::{error, info};
use std::sync::Arc;
use std::time::Duration;
use tokio::time;

#[tokio::main]
async fn main() -> Result<()> {
    // Create a tracer
    let tracer = Arc::new(InMemoryTracer::new(100));

    // Deploy a router
    let router = deploy_router(BrokerType::MOCK)?;

    // Create a tracing middleware
    let tracing_router = Arc::new(TracingMiddleware::new_with_broker_type(
        router,
        tracer.clone(),
        BrokerType::MOCK,
    ));

    // Connect to the router
    tracing_router.connect().await?;

    // Subscribe to a pipe
    let callback: Process = Box::new(|data| {
        Box::pin(async move {
            info!("Received message: {:?}", data);
            Ok(())
        })
    });
    tracing_router.subscribe("test", callback).await?;

    // Dispatch a message
    let message = "Hello, Tracing!".as_bytes().to_vec();
    tracing_router.dispatch("test", Box::new(message)).await?;

    // Get the traces
    let traces = tracer.get_traces();
    for trace in traces {
        info!("Trace: {:?}", trace);
    }

    Ok(())
}
```

## Trace Information

Each trace contains the following information:

- **ID**: A unique identifier for the trace
- **Protocol**: The protocol used to send the message
- **Pipe**: The pipe the message was sent on
- **Timestamp**: The time the message was sent
- **Size**: The size of the message in bytes
- **Duration**: The time it took to process the message
- **Success**: Whether the message was successfully processed
- **Error**: The error message if the message failed to process
- **Metadata**: Additional metadata about the message

## Monitoring Server

The monitoring package includes a monitoring server that exposes trace information via HTTP endpoints. You can start the monitoring server as follows:

```go
// Start the monitoring server
if err := monitoring.StartDefaultMonitoringServer(":8080"); err != nil {
    log.Fatalf("Failed to start monitoring server: %v", err)
}
defer monitoring.StopDefaultMonitoringServer()
```

The monitoring server provides the following endpoints:

- `/traces`: Returns all traces as JSON
- `/metrics`: Returns all metrics as JSON
- `/health`: Returns a health check response
- `/`: Returns information about the monitoring server

## Command Line Tools

The `accept` and `dispatch` command line tools support tracing via the `-monitor` flag:

```bash
# Start a subscriber with tracing
$ accept -protocol nats -pipe test -monitor

# Send a message with tracing
$ dispatch -protocol nats -pipe test -message "Hello, Tracing!" -monitor
```

## Supported Protocols

Tracing is supported for all protocols implemented in HybridPipe.io:

- NATS
- Kafka
- RabbitMQ
- AMQP
- MQTT
- Qpid
- NSQ
- TCP
- Redis
- ZeroMQ
- NetChan

## Performance Considerations

The tracing system is designed to be lightweight, but it does add some overhead to message processing. Here are some tips for minimizing the impact:

- Use a reasonable value for the maximum number of traces to keep
- Clear traces periodically if you're collecting a large number of them
- Disable tracing in production environments if performance is critical

## Examples

See the [examples](../golang/monitoring/example) directory for more examples of using the tracing functionality.

## Integration with External Systems

The tracing system is designed to be easily integrated with external monitoring systems. You can implement custom tracers that send trace information to external systems like Prometheus, Grafana, or Jaeger.

Here's an example of a custom tracer that sends trace information to a hypothetical external system:

```go
type ExternalTracer struct {
    client ExternalClient
}

func NewExternalTracer(client ExternalClient) *ExternalTracer {
    return &ExternalTracer{
        client: client,
    }
}

func (et *ExternalTracer) StartTrace(protocol, pipe string) *monitoring.MessageTrace {
    trace := monitoring.NewMessageTrace(protocol, pipe)
    return trace
}

func (et *ExternalTracer) EndTrace(trace *monitoring.MessageTrace) {
    // Send the trace to the external system
    et.client.SendTrace(trace)
}

func (et *ExternalTracer) GetTraces() []*monitoring.MessageTrace {
    // Get traces from the external system
    return et.client.GetTraces()
}

func (et *ExternalTracer) ClearTraces() {
    // Clear traces in the external system
    et.client.ClearTraces()
}
```

## Troubleshooting

If you're having issues with tracing, here are some common problems and solutions:

- **No traces are being collected**: Make sure you're using the TracingMiddleware and that the tracer is properly initialized
- **Traces are missing information**: Make sure you're using the latest version of HybridPipe.io
- **Monitoring server is not accessible**: Check that the monitoring server is running and that the port is not blocked by a firewall
