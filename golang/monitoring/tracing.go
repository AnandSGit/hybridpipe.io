// Package monitoring provides message tracing and monitoring capabilities for the HybridPipe system.
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"hybridpipe.io/core"
)

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

// NewMessageTrace creates a new message trace.
func NewMessageTrace(protocol, pipe string) *MessageTrace {
	return &MessageTrace{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Protocol:  protocol,
		Pipe:      pipe,
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// AddMetadata adds metadata to the trace.
func (mt *MessageTrace) AddMetadata(key, value string) {
	mt.Metadata[key] = value
}

// SetSuccess marks the trace as successful.
func (mt *MessageTrace) SetSuccess(size int, duration time.Duration) {
	mt.Size = size
	mt.Duration = duration
	mt.Success = true
}

// SetError marks the trace as failed.
func (mt *MessageTrace) SetError(err error, duration time.Duration) {
	mt.Duration = duration
	mt.Success = false
	mt.Error = err.Error()
}

// String returns a string representation of the trace.
func (mt *MessageTrace) String() string {
	status := "SUCCESS"
	if !mt.Success {
		status = "ERROR: " + mt.Error
	}
	return fmt.Sprintf("[%s] %s/%s: %d bytes in %v - %s", mt.Timestamp.Format(time.RFC3339), mt.Protocol, mt.Pipe, mt.Size, mt.Duration, status)
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

// MessageTracer implements the Tracer interface.
type MessageTracer struct {
	// traces is the list of traces
	traces []*MessageTrace
	// mutex protects concurrent access to the traces
	mutex sync.RWMutex
	// maxTraces is the maximum number of traces to keep
	maxTraces int
	// enabled indicates if tracing is enabled
	enabled bool
}

// NewMessageTracer creates a new message tracer.
func NewMessageTracer(maxTraces int) *MessageTracer {
	return &MessageTracer{
		traces:    make([]*MessageTrace, 0),
		maxTraces: maxTraces,
		enabled:   true,
	}
}

// StartTrace starts a new trace.
func (mt *MessageTracer) StartTrace(protocol, pipe string) *MessageTrace {
	if !mt.enabled {
		return nil
	}
	return NewMessageTrace(protocol, pipe)
}

// EndTrace ends a trace.
func (mt *MessageTracer) EndTrace(trace *MessageTrace) {
	if !mt.enabled || trace == nil {
		return
	}

	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	// Add the trace to the list
	mt.traces = append(mt.traces, trace)

	// Trim the list if it's too long
	if mt.maxTraces > 0 && len(mt.traces) > mt.maxTraces {
		mt.traces = mt.traces[len(mt.traces)-mt.maxTraces:]
	}
}

// GetTraces returns all traces.
func (mt *MessageTracer) GetTraces() []*MessageTrace {
	mt.mutex.RLock()
	defer mt.mutex.RUnlock()

	// Return a copy of the traces
	traces := make([]*MessageTrace, len(mt.traces))
	copy(traces, mt.traces)
	return traces
}

// ClearTraces clears all traces.
func (mt *MessageTracer) ClearTraces() {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	mt.traces = make([]*MessageTrace, 0)
}

// Enable enables tracing.
func (mt *MessageTracer) Enable() {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	mt.enabled = true
}

// Disable disables tracing.
func (mt *MessageTracer) Disable() {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	mt.enabled = false
}

// IsEnabled returns true if tracing is enabled.
func (mt *MessageTracer) IsEnabled() bool {
	mt.mutex.RLock()
	defer mt.mutex.RUnlock()

	return mt.enabled
}

// TracingMiddleware is a middleware that adds tracing to a HybridPipe.
type TracingMiddleware struct {
	// router is the underlying router
	router core.HybridPipe
	// tracer is the message tracer
	tracer Tracer
	// protocol is the protocol name
	protocol string
}

// NewTracingMiddleware creates a new tracing middleware.
func NewTracingMiddleware(router core.HybridPipe, tracer Tracer, protocol string) *TracingMiddleware {
	return &TracingMiddleware{
		router:   router,
		tracer:   tracer,
		protocol: protocol,
	}
}

// Connect connects to the underlying router.
func (tm *TracingMiddleware) Connect() error {
	return tm.router.Connect()
}

// Close closes the underlying router.
func (tm *TracingMiddleware) Close() {
	tm.router.Close()
}

// Dispatch sends a message to the specified pipe with tracing.
func (tm *TracingMiddleware) Dispatch(pipe string, data interface{}) error {
	// Start a trace
	trace := tm.tracer.StartTrace(tm.protocol, pipe)
	startTime := time.Now()

	// Encode the data to get the size
	encodedData, err := core.Encode(data)
	if err != nil {
		if trace != nil {
			trace.SetError(err, time.Since(startTime))
			tm.tracer.EndTrace(trace)
		}
		return fmt.Errorf("failed to encode data: %w", err)
	}

	// Dispatch the message
	err = tm.router.Dispatch(pipe, data)
	if err != nil {
		if trace != nil {
			trace.SetError(err, time.Since(startTime))
			tm.tracer.EndTrace(trace)
		}
		return err
	}

	// End the trace
	if trace != nil {
		trace.SetSuccess(len(encodedData), time.Since(startTime))
		tm.tracer.EndTrace(trace)
	}

	return nil
}

// DispatchWithContext sends a message to the specified pipe with context and tracing.
func (tm *TracingMiddleware) DispatchWithContext(ctx context.Context, pipe string, data interface{}) error {
	// Start a trace
	trace := tm.tracer.StartTrace(tm.protocol, pipe)
	startTime := time.Now()

	// Encode the data to get the size
	encodedData, err := core.Encode(data)
	if err != nil {
		if trace != nil {
			trace.SetError(err, time.Since(startTime))
			tm.tracer.EndTrace(trace)
		}
		return fmt.Errorf("failed to encode data: %w", err)
	}

	// Check if the router implements ContextAwareHybridPipe
	if ctxRouter, ok := tm.router.(core.ContextAwareHybridPipe); ok {
		// Dispatch the message with context
		err = ctxRouter.DispatchWithContext(ctx, pipe, data)
	} else {
		// Fall back to regular dispatch
		err = tm.router.Dispatch(pipe, data)
	}

	if err != nil {
		if trace != nil {
			trace.SetError(err, time.Since(startTime))
			tm.tracer.EndTrace(trace)
		}
		return err
	}

	// End the trace
	if trace != nil {
		trace.SetSuccess(len(encodedData), time.Since(startTime))
		tm.tracer.EndTrace(trace)
	}

	return nil
}

// Subscribe registers a callback for the specified pipe with tracing.
func (tm *TracingMiddleware) Subscribe(pipe string, callback core.Process) error {
	// Create a wrapper that adds tracing
	wrapper := func(data []byte) error {
		// Start a trace
		trace := tm.tracer.StartTrace(tm.protocol, pipe)
		startTime := time.Now()

		// Call the original callback
		err := callback(data)
		if err != nil {
			if trace != nil {
				trace.SetError(err, time.Since(startTime))
				tm.tracer.EndTrace(trace)
			}
			return err
		}

		// End the trace
		if trace != nil {
			trace.SetSuccess(len(data), time.Since(startTime))
			tm.tracer.EndTrace(trace)
		}

		return nil
	}

	// Subscribe with the wrapper
	return tm.router.Subscribe(pipe, wrapper)
}

// SubscribeWithContext registers a callback for the specified pipe with context and tracing.
func (tm *TracingMiddleware) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Create a wrapper that adds tracing
	wrapper := func(data []byte) error {
		// Start a trace
		trace := tm.tracer.StartTrace(tm.protocol, pipe)
		startTime := time.Now()

		// Call the original callback
		err := callback(data)
		if err != nil {
			if trace != nil {
				trace.SetError(err, time.Since(startTime))
				tm.tracer.EndTrace(trace)
			}
			return err
		}

		// End the trace
		if trace != nil {
			trace.SetSuccess(len(data), time.Since(startTime))
			tm.tracer.EndTrace(trace)
		}

		return nil
	}

	// Check if the router implements ContextAwareHybridPipe
	if ctxRouter, ok := tm.router.(core.ContextAwareHybridPipe); ok {
		// Subscribe with context
		return ctxRouter.SubscribeWithContext(ctx, pipe, wrapper)
	}

	// Fall back to regular subscribe
	return tm.router.Subscribe(pipe, wrapper)
}

// Unsubscribe removes all callbacks for the specified pipe.
func (tm *TracingMiddleware) Unsubscribe(pipe string) error {
	return tm.router.Unsubscribe(pipe)
}
