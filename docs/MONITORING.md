# Message Monitoring Documentation

The Message Monitoring package provides comprehensive message tracing and performance monitoring capabilities for the HybridPipe system. It allows you to track messages through the system, measure performance, and identify bottlenecks.

## Overview

The Message Monitoring package consists of two main components:

1. **Message Tracing**: Tracks messages as they flow through the system, recording information such as message size, processing time, and success/failure status.
2. **Metrics Collection**: Collects performance metrics such as message counts, processing times, and error rates.

These components can be used together or separately, depending on your needs.

## Installation

The Message Monitoring package is included with HybridPipe. No additional installation is required.

## Message Tracing

### Overview

Message tracing allows you to track messages as they flow through the system. Each message is assigned a unique trace ID, and information about the message is recorded at various points in its lifecycle.

### Usage

```go
package main

import (
    "fmt"
    "log"

    "hybridpipe.io/core"
    "hybridpipe.io/monitoring"
    "hybridpipe.io/protocols/mock"
)

func main() {
    // Create a message tracer
    tracer := monitoring.NewMessageTracer(1000)

    // Create a mock router
    router := mock.NewMockRouter()

    // Create a tracing middleware
    tracingRouter := monitoring.NewTracingMiddleware(router, tracer, "MOCK")

    // Connect to the router
    if err := tracingRouter.Connect(); err != nil {
        log.Fatalf("Failed to connect to router: %v", err)
    }
    defer tracingRouter.Close()

    // Subscribe to a pipe
    if err := tracingRouter.Subscribe("test", func(data []byte) error {
        var message string
        if err := core.Decode(data, &message); err != nil {
            return err
        }
        fmt.Printf("Received message: %s\n", message)
        return nil
    }); err != nil {
        log.Fatalf("Failed to subscribe to pipe: %v", err)
    }

    // Send a message
    if err := tracingRouter.Dispatch("test", "Hello, Tracing!"); err != nil {
        log.Fatalf("Failed to send message: %v", err)
    }

    // Print the traces
    for _, trace := range tracer.GetTraces() {
        fmt.Println(trace)
    }
}
```

### API Reference

```go
// MessageTrace represents a trace of a message through the system.
type MessageTrace struct {
    // ID is the unique identifier for the trace
    ID string
    // Protocol is the protocol used to send the message
    Protocol string
    // Pipe is the pipe the message was sent on
    Pipe string
    // Timestamp is the time the message was sent
    Timestamp time.Time
    // Size is the size of the message in bytes
    Size int
    // Duration is the time it took to process the message
    Duration time.Duration
    // Success indicates if the message was successfully processed
    Success bool
    // Error is the error message if the message failed to process
    Error string
    // Metadata is additional metadata about the message
    Metadata map[string]string
}

// Tracer is the interface for message tracers.
type Tracer interface {
    // StartTrace starts a new trace.
    StartTrace(protocol, pipe string) *MessageTrace
    // EndTrace ends a trace.
    EndTrace(trace *MessageTrace)
    // GetTraces returns all traces.
    GetTraces() []*MessageTrace
    // ClearTraces clears all traces.
    ClearTraces()
}

// NewMessageTracer creates a new message tracer.
func NewMessageTracer(maxTraces int) *MessageTracer

// TracingMiddleware is a middleware that adds tracing to a HybridPipe.
type TracingMiddleware struct {
    // ...
}

// NewTracingMiddleware creates a new tracing middleware.
func NewTracingMiddleware(router core.HybridPipe, tracer Tracer, protocol string) *TracingMiddleware
```

## Metrics Collection

### Overview

Metrics collection allows you to measure the performance of your system. You can track metrics such as message counts, processing times, and error rates.

### Usage

```go
package main

import (
    "fmt"
    "log"
    "time"

    "hybridpipe.io/monitoring"
)

func main() {
    // Create metrics
    messagesReceived := monitoring.NewCounter("messages_received")
    messagesSent := monitoring.NewCounter("messages_sent")
    messageSize := monitoring.NewHistogram("message_size", []int64{10, 100, 1000, 10000})
    activeConnections := monitoring.NewGauge("active_connections")

    // Register the metrics
    monitoring.Register(messagesReceived)
    monitoring.Register(messagesSent)
    monitoring.Register(messageSize)
    monitoring.Register(activeConnections)

    // Start the monitoring server
    if err := monitoring.StartDefaultMonitoringServer(":8080"); err != nil {
        log.Fatalf("Failed to start monitoring server: %v", err)
    }
    defer monitoring.StopDefaultMonitoringServer()

    // Update metrics
    messagesReceived.Increment(1)
    messagesSent.Increment(1)
    messageSize.Observe(100)
    activeConnections.Set(1)

    // Print the metrics
    fmt.Printf("Messages received: %d\n", messagesReceived.GetValue())
    fmt.Printf("Messages sent: %d\n", messagesSent.GetValue())
    fmt.Printf("Active connections: %d\n", activeConnections.GetValue())

    // Print the histogram buckets
    buckets := messageSize.GetBuckets()
    fmt.Println("Message size histogram:")
    for bucket, count := range buckets {
        fmt.Printf("  <= %d bytes: %d\n", bucket, count)
    }

    // Wait for the user to press Ctrl+C
    fmt.Println("Monitoring server is running on http://localhost:8080")
    fmt.Println("Press Ctrl+C to exit")
    select {}
}
```

### API Reference

```go
// MetricType represents the type of a metric.
type MetricType int

const (
    // MetricTypeCounter is a metric that only increases.
    MetricTypeCounter MetricType = iota
    // MetricTypeGauge is a metric that can increase or decrease.
    MetricTypeGauge
    // MetricTypeHistogram is a metric that tracks the distribution of values.
    MetricTypeHistogram
)

// Metric represents a metric.
type Metric struct {
    // ...
}

// NewCounter creates a new counter metric.
func NewCounter(name string) *Metric

// NewGauge creates a new gauge metric.
func NewGauge(name string) *Metric

// NewHistogram creates a new histogram metric.
func NewHistogram(name string, buckets []int64) *Metric

// MetricsRegistry is a registry of metrics.
type MetricsRegistry struct {
    // ...
}

// NewMetricsRegistry creates a new metrics registry.
func NewMetricsRegistry() *MetricsRegistry

// MonitoringServer is a server that exposes metrics and traces.
type MonitoringServer struct {
    // ...
}

// NewMonitoringServer creates a new monitoring server.
func NewMonitoringServer(address string, metricsRegistry *MetricsRegistry, tracer Tracer) *MonitoringServer

// StartDefaultMonitoringServer starts the default monitoring server.
func StartDefaultMonitoringServer(address string) error

// StopDefaultMonitoringServer stops the default monitoring server.
func StopDefaultMonitoringServer() error
```

## Monitoring Server

The monitoring server exposes metrics and traces via HTTP endpoints. You can access these endpoints using a web browser or any HTTP client.

### Endpoints

- `/metrics`: Returns all metrics as JSON
- `/traces`: Returns all traces as JSON
- `/health`: Returns a health check response
- `/`: Returns information about the monitoring server

### Example

```bash
# Start the monitoring server
$ go run main.go

# Access the metrics endpoint
$ curl http://localhost:8080/metrics
[{"name":"messages_received","type":0,"value":1,"tags":{},"timestamp":"2025-04-19T12:34:56Z"},...]

# Access the traces endpoint
$ curl http://localhost:8080/traces
[{"id":"1650372896123456789","protocol":"MOCK","pipe":"test","timestamp":"2025-04-19T12:34:56Z","size":16,"duration":123456,"success":true,"error":"","metadata":{}}]
```

## Integration with HybridPipe

The Message Monitoring package is fully integrated with the HybridPipe system. You can use it with any protocol implementation by wrapping the router with the tracing middleware.

```go
// Create a NATS router
config := &core.NATSConfig{
    URL: "nats://localhost:4222",
}
router := nats.New(config)

// Create a tracing middleware
tracer := monitoring.NewMessageTracer(1000)
tracingRouter := monitoring.NewTracingMiddleware(router, tracer, "NATS")

// Use the tracing router instead of the original router
if err := tracingRouter.Connect(); err != nil {
    log.Fatalf("Failed to connect to NATS server: %v", err)
}
defer tracingRouter.Close()
```

## Examples

See the [examples](../golang/monitoring/example) directory for more examples of using the Message Monitoring package.
