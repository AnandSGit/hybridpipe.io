# Middleware

This directory contains middleware components for the HybridPipe system.

## Purpose

The `middleware` directory is intended for components that can be inserted between the application and the messaging system to provide additional functionality, such as:

- Message tracing
- Logging
- Metrics collection
- Rate limiting
- Retries
- Circuit breaking
- Authentication and authorization

## Usage

Middleware components implement the `HybridPipe` interface and wrap another `HybridPipe` implementation, allowing them to be chained together.

Example:

```go
// Create a base router
router, err := hybridpipe.DeployRouter(hybridpipe.NATS)
if err != nil {
    log.Fatalf("Failed to deploy router: %v", err)
}

// Wrap it with tracing middleware
tracedRouter := middleware.NewTracedRouter(router, "my-service")

// Use the traced router
tracedRouter.Dispatch("my-pipe", "Hello, world!")
```

## Available Middleware

- `TracedRouter` - Adds tracing information to messages
- `LoggingRouter` - Logs all messages
- `MetricsRouter` - Collects metrics on message flow
- `RetryRouter` - Automatically retries failed dispatches

Last updated: April 19, 2025
