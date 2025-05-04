# HybridPipe Tracing

## Overview

HybridPipe provides a tracing system that allows you to track messages as they flow through your application. This document describes the tracing system in detail.

## Features

- **Message Tracing**: Track messages as they flow through your application
- **Span Linking**: Link related spans to form a trace
- **Customizable**: Configure tracing to suit your needs
- **Middleware Integration**: Easily add tracing to your application
- **Callback Support**: Process trace information as it's generated

## Basic Usage

### Adding Tracing to a Router

```go
package main

import (
    "log"
    "time"

    "hybridpipe.io"
    "hybridpipe.io/tracing"
)

func main() {
    // Deploy a router for the desired messaging system
    baseRouter, err := hybridpipe.DeployRouter(hybridpipe.NATS)
    if err != nil {
        log.Fatalf("Failed to deploy router: %v", err)
    }
    defer baseRouter.Close()

    // Add tracing middleware
    tracedRouter := tracing.NewTracedRouter(baseRouter, "my-service")

    // Subscribe to a pipe
    err = tracedRouter.Subscribe("my-pipe", func(data []byte) error {
        var message string
        if err := hybridpipe.Decode(data, &message); err != nil {
            return err
        }
        log.Printf("Received message: %s", message)
        return nil
    })
    if err != nil {
        log.Fatalf("Failed to subscribe: %v", err)
    }

    // Dispatch a message
    err = tracedRouter.Dispatch("my-pipe", "Hello, Traced HybridPipe!")
    if err != nil {
        log.Fatalf("Failed to dispatch message: %v", err)
    }

    // Wait for the message to be processed
    time.Sleep(1 * time.Second)
}
```

### Customizing Tracing

```go
package main

import (
    "log"
    "time"

    "hybridpipe.io"
    "hybridpipe.io/tracing"
)

func main() {
    // Deploy a router for the desired messaging system
    baseRouter, err := hybridpipe.DeployRouter(hybridpipe.NATS)
    if err != nil {
        log.Fatalf("Failed to deploy router: %v", err)
    }
    defer baseRouter.Close()

    // Add tracing middleware
    tracedRouter := tracing.NewTracedRouter(baseRouter, "my-service")

    // Customize tracing
    tracedRouter.SetTracingEnabled(true)
    tracedRouter.SetTracingCallback(func(info tracing.TracingInfo) {
        log.Printf("Trace: %s, Span: %s, Operation: %s, Duration: %dms, Status: %s",
            info.TraceID, info.SpanID, info.Operation, info.Duration, info.Status)
        
        // You can store the trace information in a database or send it to a tracing system
    })

    // Subscribe to a pipe
    err = tracedRouter.Subscribe("my-pipe", func(data []byte) error {
        var message string
        if err := hybridpipe.Decode(data, &message); err != nil {
            return err
        }
        log.Printf("Received message: %s", message)
        return nil
    })
    if err != nil {
        log.Fatalf("Failed to subscribe: %v", err)
    }

    // Dispatch a message
    err = tracedRouter.Dispatch("my-pipe", "Hello, Traced HybridPipe!")
    if err != nil {
        log.Fatalf("Failed to dispatch message: %v", err)
    }

    // Wait for the message to be processed
    time.Sleep(1 * time.Second)
}
```

## Tracing Information

The `TracingInfo` struct contains the following information:

```go
type TracingInfo struct {
    // TraceID is a unique identifier for the trace.
    TraceID string `json:"trace_id"`
    // SpanID is a unique identifier for the span.
    SpanID string `json:"span_id"`
    // ParentSpanID is the ID of the parent span.
    ParentSpanID string `json:"parent_span_id,omitempty"`
    // ServiceName is the name of the service that generated the trace.
    ServiceName string `json:"service_name"`
    // Operation is the operation being performed.
    Operation string `json:"operation"`
    // StartTime is the time the operation started.
    StartTime time.Time `json:"start_time"`
    // EndTime is the time the operation ended.
    EndTime time.Time `json:"end_time,omitempty"`
    // Duration is the duration of the operation in milliseconds.
    Duration int64 `json:"duration,omitempty"`
    // Status is the status of the operation.
    Status string `json:"status"`
    // Error is the error message if the operation failed.
    Error string `json:"error,omitempty"`
    // Tags are additional key-value pairs associated with the trace.
    Tags map[string]string `json:"tags,omitempty"`
}
```

## Trace Propagation

HybridPipe automatically propagates trace information across service boundaries. When a message is dispatched with tracing enabled, the trace information is included in the message. When the message is received, the trace information is extracted and linked to the new span.

```go
// Service A
tracedRouter := tracing.NewTracedRouter(baseRouter, "service-a")
tracedRouter.Dispatch("my-pipe", "Hello, Traced HybridPipe!")

// Service B
tracedRouter := tracing.NewTracedRouter(baseRouter, "service-b")
tracedRouter.Subscribe("my-pipe", func(data []byte) error {
    // The trace information from Service A is automatically linked to this span
    return nil
})
```

## Context-Aware Tracing

HybridPipe supports context-aware tracing, which allows you to propagate trace information through context:

```go
package main

import (
    "context"
    "log"
    "time"

    "hybridpipe.io"
    "hybridpipe.io/core"
    "hybridpipe.io/tracing"
)

func main() {
    // Deploy a router for the desired messaging system
    baseRouter, err := hybridpipe.DeployRouter(hybridpipe.NATS)
    if err != nil {
        log.Fatalf("Failed to deploy router: %v", err)
    }
    defer baseRouter.Close()

    // Add tracing middleware
    tracedRouter := tracing.NewTracedRouter(baseRouter, "my-service")

    // Check if the router supports context-aware operations
    ctxRouter, ok := tracedRouter.(core.ContextAwareHybridPipe)
    if !ok {
        log.Fatalf("Router does not support context-aware operations")
    }

    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Subscribe to a pipe with context
    err = ctxRouter.SubscribeWithContext(ctx, "my-pipe", func(data []byte) error {
        var message string
        if err := hybridpipe.Decode(data, &message); err != nil {
            return err
        }
        log.Printf("Received message: %s", message)
        return nil
    })
    if err != nil {
        log.Fatalf("Failed to subscribe: %v", err)
    }

    // Dispatch a message with context
    err = ctxRouter.DispatchWithContext(ctx, "my-pipe", "Hello, Context-Aware Traced HybridPipe!")
    if err != nil {
        log.Fatalf("Failed to dispatch message: %v", err)
    }

    // Wait for the message to be processed
    time.Sleep(1 * time.Second)
}
```

## Integration with External Tracing Systems

HybridPipe's tracing system can be integrated with external tracing systems like OpenTelemetry, Jaeger, or Zipkin. You can customize the tracing callback to send trace information to these systems:

```go
package main

import (
    "log"
    "time"

    "hybridpipe.io"
    "hybridpipe.io/tracing"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    // Deploy a router for the desired messaging system
    baseRouter, err := hybridpipe.DeployRouter(hybridpipe.NATS)
    if err != nil {
        log.Fatalf("Failed to deploy router: %v", err)
    }
    defer baseRouter.Close()

    // Add tracing middleware
    tracedRouter := tracing.NewTracedRouter(baseRouter, "my-service")

    // Set up OpenTelemetry
    tracer := otel.Tracer("hybridpipe")

    // Customize tracing callback to send trace information to OpenTelemetry
    tracedRouter.SetTracingCallback(func(info tracing.TracingInfo) {
        // Create a span
        ctx, span := tracer.Start(
            context.Background(),
            info.Operation,
            trace.WithSpanKind(trace.SpanKindProducer),
        )
        defer span.End()

        // Set span attributes
        span.SetAttributes(
            attribute.String("trace_id", info.TraceID),
            attribute.String("span_id", info.SpanID),
            attribute.String("parent_span_id", info.ParentSpanID),
            attribute.String("service_name", info.ServiceName),
            attribute.Int64("duration_ms", info.Duration),
            attribute.String("status", info.Status),
        )

        // Set error if present
        if info.Error != "" {
            span.SetStatus(codes.Error, info.Error)
        }

        // Set tags
        for key, value := range info.Tags {
            span.SetAttributes(attribute.String(key, value))
        }
    })

    // Subscribe to a pipe
    err = tracedRouter.Subscribe("my-pipe", func(data []byte) error {
        var message string
        if err := hybridpipe.Decode(data, &message); err != nil {
            return err
        }
        log.Printf("Received message: %s", message)
        return nil
    })
    if err != nil {
        log.Fatalf("Failed to subscribe: %v", err)
    }

    // Dispatch a message
    err = tracedRouter.Dispatch("my-pipe", "Hello, OpenTelemetry-Integrated HybridPipe!")
    if err != nil {
        log.Fatalf("Failed to dispatch message: %v", err)
    }

    // Wait for the message to be processed
    time.Sleep(1 * time.Second)
}
```

## Best Practices

- **Use Descriptive Service Names**: Choose service names that clearly identify the service in your distributed system.
- **Add Custom Tags**: Add custom tags to spans to provide additional context for debugging.
- **Process Trace Information**: Process trace information in real-time to detect issues early.
- **Integrate with External Systems**: Integrate with external tracing systems for advanced visualization and analysis.
- **Enable Tracing Selectively**: Enable tracing only for critical paths to reduce overhead.

## Conclusion

HybridPipe's tracing system provides a powerful way to track messages as they flow through your distributed system. By using tracing, you can gain insights into the performance and behavior of your application, making it easier to diagnose issues and optimize performance.

Last updated: April 19, 2025
