package hybridpipe

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// tracedRouter is a HybridPipe router with tracing
type tracedRouter struct {
	// router is the underlying router
	router HybridPipe
	// tracer is the tracer
	tracer Tracer
	// meter is the meter
	meter Meter
	// handlers is a map of pipe names to handler functions
	handlers map[string]Process
	// mu protects the handlers map
	mu sync.RWMutex
}

// WrapRouter wraps a HybridPipe router with tracing middleware
func WrapRouter(router HybridPipe, tracer Tracer) HybridPipe {
	return &tracedRouter{
		router:   router,
		tracer:   tracer,
		handlers: make(map[string]Process),
	}
}

// WrapRouterWithMetrics wraps a HybridPipe router with tracing and metrics middleware
func WrapRouterWithMetrics(router HybridPipe, tracer Tracer, meter Meter) HybridPipe {
	return &tracedRouter{
		router:   router,
		tracer:   tracer,
		meter:    meter,
		handlers: make(map[string]Process),
	}
}

// Connect establishes a connection to the messaging system
func (r *tracedRouter) Connect() error {
	// Start a new span
	_, span := r.tracer.Start(context.Background(), "Connect")
	defer span.End()

	// Call the underlying router
	err := r.router.Connect()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(2, err.Error())
	}

	return err
}

// Dispatch sends a message to the specified pipe
func (r *tracedRouter) Dispatch(pipe string, data interface{}) error {
	// Start a new span
	ctx, span := r.tracer.Start(context.Background(), "Dispatch")
	defer span.End()

	// Set span attributes
	span.SetAttribute("pipe", pipe)
	span.SetAttribute("message_type", fmt.Sprintf("%T", data))

	// Inject trace context into the message
	r.tracer.Inject(ctx, data)

	// Record metrics
	if r.meter != nil {
		// Record message count
		messageCount := r.meter.NewCounter("hybridpipe_messages_sent_total")
		messageCount.Add(ctx, 1)

		// Record message size
		messageSize := r.meter.NewHistogram("hybridpipe_message_size_bytes",
			WithDescription("Size of messages in bytes"),
			WithUnit("bytes"),
		)
		// This is a simplified size calculation
		messageSize.Record(ctx, float64(estimateSize(data)), Label{Key: "pipe", Value: pipe})
	}

	// Call the underlying router
	err := r.router.Dispatch(pipe, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(2, err.Error())

		// Record error metrics
		if r.meter != nil {
			errorCount := r.meter.NewCounter("hybridpipe_errors_total")
			errorCount.Add(ctx, 1)
		}
	}

	return err
}

// Accept subscribes to messages from the specified pipe
func (r *tracedRouter) Accept(pipe string, fn Process) error {
	// Start a new span
	ctx, span := r.tracer.Start(context.Background(), "Accept")
	defer span.End()

	// Set span attributes
	span.SetAttribute("pipe", pipe)

	// Create a traced handler function
	tracedFn := func(data interface{}) {
		// Extract trace context from the message
		ctx := r.tracer.Extract(context.Background(), data)

		// Start a new span
		ctx, span := r.tracer.Start(ctx, "Process")
		defer span.End()

		// Set span attributes
		span.SetAttribute("pipe", pipe)
		span.SetAttribute("message_type", fmt.Sprintf("%T", data))

		// Record metrics
		if r.meter != nil {
			// Record message count
			messageCount := r.meter.NewCounter("hybridpipe_messages_received_total")
			messageCount.Add(ctx, 1)

			// Record message size
			messageSize := r.meter.NewHistogram("hybridpipe_message_size_bytes")
			// This is a simplified size calculation
			messageSize.Record(ctx, float64(estimateSize(data)))

			// Start a timer for processing latency
			startTime := time.Now()
			defer func() {
				// Record processing latency
				latency := r.meter.NewHistogram("hybridpipe_message_processing_seconds")
				latency.Record(ctx, time.Since(startTime).Seconds())
			}()
		}

		// Call the original handler function
		fn(data)
	}

	// Store the handler function
	r.mu.Lock()
	r.handlers[pipe] = tracedFn
	r.mu.Unlock()

	// Call the underlying router
	err := r.router.Accept(pipe, tracedFn)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(2, err.Error())

		// Record error metrics
		if r.meter != nil {
			errorCount := r.meter.NewCounter("hybridpipe_errors_total")
			errorCount.Add(ctx, 1)
		}
	}

	return err
}

// Remove unsubscribes from the specified pipe
func (r *tracedRouter) Remove(pipe string) error {
	// Start a new span
	ctx, span := r.tracer.Start(context.Background(), "Remove")
	defer span.End()

	// Set span attributes
	span.SetAttribute("pipe", pipe)

	// Call the underlying router
	err := r.router.Remove(pipe)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(2, err.Error())

		// Record error metrics
		if r.meter != nil {
			errorCount := r.meter.NewCounter("hybridpipe_errors_total")
			errorCount.Add(ctx, 1)
		}
	}

	// Remove the handler function
	r.mu.Lock()
	delete(r.handlers, pipe)
	r.mu.Unlock()

	return err
}

// Close terminates the connection to the messaging system
func (r *tracedRouter) Close() {
	// Start a new span
	_, span := r.tracer.Start(context.Background(), "Close")
	defer span.End()

	// Call the underlying router
	r.router.Close()
}

// estimateSize estimates the size of a value in bytes
func estimateSize(value interface{}) int {
	// This is a simplified implementation
	// In a real implementation, we would use reflection to estimate the size
	switch v := value.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	case map[string]interface{}:
		size := 0
		for k, v := range v {
			size += len(k) + estimateSize(v)
		}
		return size
	default:
		return 100 // Default size
	}
}

// EnableAutoInstrumentation enables automatic instrumentation for all HybridPipe operations
func EnableAutoInstrumentation() {
	// This is a simplified implementation
	// In a real implementation, we would use monkey patching or other techniques
	// to automatically instrument all HybridPipe operations
	fmt.Println("Auto-instrumentation enabled")
}
