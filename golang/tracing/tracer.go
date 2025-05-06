// Package tracing provides tracing functionality for the HybridPipe system.
package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"hybridpipe.io/core"
	"hybridpipe.io/middleware"
)

// TracingInfo contains tracing information for a message.
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

// TracedMessage wraps a message with tracing information.
type TracedMessage struct {
	// Tracing contains the tracing information.
	Tracing TracingInfo `json:"tracing"`
	// Payload is the original message payload.
	Payload any `json:"payload"`
}

// TracedRouter is a middleware that adds tracing to a HybridPipe implementation.
type TracedRouter struct {
	*middleware.ContextAwareBaseMiddleware
	// ServiceName is the name of the service.
	ServiceName string
	// TraceIDGenerator generates trace IDs.
	TraceIDGenerator func() string
	// SpanIDGenerator generates span IDs.
	SpanIDGenerator func() string
	// TracingEnabled indicates whether tracing is enabled.
	TracingEnabled bool
	// TracingCallback is called when a trace is completed.
	TracingCallback func(TracingInfo)
}

// NewTracedRouter creates a new TracedRouter.
func NewTracedRouter(router core.HybridPipe, serviceName string) *TracedRouter {
	return &TracedRouter{
		ContextAwareBaseMiddleware: middleware.NewContextAwareBaseMiddleware(router),
		ServiceName:                serviceName,
		TraceIDGenerator:           generateID,
		SpanIDGenerator:            generateID,
		TracingEnabled:             true,
		TracingCallback:            defaultTracingCallback,
	}
}

// Dispatch sends a message to the specified pipe with tracing information.
func (tr *TracedRouter) Dispatch(pipe string, data any) error {
	if !tr.TracingEnabled {
		return tr.Router.Dispatch(pipe, data)
	}

	// Create tracing information
	traceInfo := TracingInfo{
		TraceID:     tr.TraceIDGenerator(),
		SpanID:      tr.SpanIDGenerator(),
		ServiceName: tr.ServiceName,
		Operation:   fmt.Sprintf("dispatch:%s", pipe),
		StartTime:   time.Now(),
		Status:      "success",
		Tags:        make(map[string]string),
	}

	// Add pipe name to tags
	traceInfo.Tags["pipe"] = pipe

	// Create traced message
	tracedMsg := TracedMessage{
		Tracing: traceInfo,
		Payload: data,
	}

	// Dispatch the traced message
	err := tr.Router.Dispatch(pipe, tracedMsg)

	// Update tracing information
	traceInfo.EndTime = time.Now()
	traceInfo.Duration = traceInfo.EndTime.Sub(traceInfo.StartTime).Milliseconds()
	if err != nil {
		traceInfo.Status = "error"
		traceInfo.Error = err.Error()
	}

	// Call tracing callback
	tr.TracingCallback(traceInfo)

	return err
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (tr *TracedRouter) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	if !tr.TracingEnabled {
		if tr.ContextRouter != nil {
			return tr.ContextRouter.DispatchWithContext(ctx, pipe, data)
		}
		return tr.Router.Dispatch(pipe, data)
	}

	// Create tracing information
	traceInfo := TracingInfo{
		TraceID:     tr.TraceIDGenerator(),
		SpanID:      tr.SpanIDGenerator(),
		ServiceName: tr.ServiceName,
		Operation:   fmt.Sprintf("dispatch_with_context:%s", pipe),
		StartTime:   time.Now(),
		Status:      "success",
		Tags:        make(map[string]string),
	}

	// Add pipe name to tags
	traceInfo.Tags["pipe"] = pipe

	// Create traced message
	tracedMsg := TracedMessage{
		Tracing: traceInfo,
		Payload: data,
	}

	// Dispatch the traced message
	var err error
	if tr.ContextRouter != nil {
		err = tr.ContextRouter.DispatchWithContext(ctx, pipe, tracedMsg)
	} else {
		err = tr.Router.Dispatch(pipe, tracedMsg)
	}

	// Update tracing information
	traceInfo.EndTime = time.Now()
	traceInfo.Duration = traceInfo.EndTime.Sub(traceInfo.StartTime).Milliseconds()
	if err != nil {
		traceInfo.Status = "error"
		traceInfo.Error = err.Error()
	}

	// Call tracing callback
	tr.TracingCallback(traceInfo)

	return err
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (tr *TracedRouter) Subscribe(pipe string, callback core.Process) error {
	if !tr.TracingEnabled {
		return tr.Router.Subscribe(pipe, callback)
	}

	// Create a wrapper function that extracts tracing information
	wrapper := func(data []byte) error {
		// Create tracing information for the receive operation
		receiveTraceInfo := TracingInfo{
			TraceID:     tr.TraceIDGenerator(),
			SpanID:      tr.SpanIDGenerator(),
			ServiceName: tr.ServiceName,
			Operation:   fmt.Sprintf("receive:%s", pipe),
			StartTime:   time.Now(),
			Status:      "success",
			Tags:        make(map[string]string),
		}

		// Add pipe name to tags
		receiveTraceInfo.Tags["pipe"] = pipe

		// Try to decode the message as a TracedMessage
		var tracedMsg TracedMessage
		if err := json.Unmarshal(data, &tracedMsg); err != nil {
			// Not a traced message, call the original callback
			err := callback(data)
			
			// Update tracing information
			receiveTraceInfo.EndTime = time.Now()
			receiveTraceInfo.Duration = receiveTraceInfo.EndTime.Sub(receiveTraceInfo.StartTime).Milliseconds()
			if err != nil {
				receiveTraceInfo.Status = "error"
				receiveTraceInfo.Error = err.Error()
			}
			
			// Call tracing callback
			tr.TracingCallback(receiveTraceInfo)
			
			return err
		}

		// Link the receive span to the dispatch span
		receiveTraceInfo.TraceID = tracedMsg.Tracing.TraceID
		receiveTraceInfo.ParentSpanID = tracedMsg.Tracing.SpanID

		// Encode the payload
		payloadBytes, err := json.Marshal(tracedMsg.Payload)
		if err != nil {
			receiveTraceInfo.Status = "error"
			receiveTraceInfo.Error = err.Error()
			tr.TracingCallback(receiveTraceInfo)
			return err
		}

		// Call the original callback with the payload
		err = callback(payloadBytes)

		// Update tracing information
		receiveTraceInfo.EndTime = time.Now()
		receiveTraceInfo.Duration = receiveTraceInfo.EndTime.Sub(receiveTraceInfo.StartTime).Milliseconds()
		if err != nil {
			receiveTraceInfo.Status = "error"
			receiveTraceInfo.Error = err.Error()
		}

		// Call tracing callback
		tr.TracingCallback(receiveTraceInfo)

		return err
	}

	return tr.Router.Subscribe(pipe, wrapper)
}

// SubscribeWithContext registers a callback with context for cancellation.
func (tr *TracedRouter) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	if !tr.TracingEnabled {
		if tr.ContextRouter != nil {
			return tr.ContextRouter.SubscribeWithContext(ctx, pipe, callback)
		}
		return tr.Router.Subscribe(pipe, callback)
	}

	// Create a wrapper function that extracts tracing information
	wrapper := func(data []byte) error {
		// Create tracing information for the receive operation
		receiveTraceInfo := TracingInfo{
			TraceID:     tr.TraceIDGenerator(),
			SpanID:      tr.SpanIDGenerator(),
			ServiceName: tr.ServiceName,
			Operation:   fmt.Sprintf("receive_with_context:%s", pipe),
			StartTime:   time.Now(),
			Status:      "success",
			Tags:        make(map[string]string),
		}

		// Add pipe name to tags
		receiveTraceInfo.Tags["pipe"] = pipe

		// Try to decode the message as a TracedMessage
		var tracedMsg TracedMessage
		if err := json.Unmarshal(data, &tracedMsg); err != nil {
			// Not a traced message, call the original callback
			err := callback(data)
			
			// Update tracing information
			receiveTraceInfo.EndTime = time.Now()
			receiveTraceInfo.Duration = receiveTraceInfo.EndTime.Sub(receiveTraceInfo.StartTime).Milliseconds()
			if err != nil {
				receiveTraceInfo.Status = "error"
				receiveTraceInfo.Error = err.Error()
			}
			
			// Call tracing callback
			tr.TracingCallback(receiveTraceInfo)
			
			return err
		}

		// Link the receive span to the dispatch span
		receiveTraceInfo.TraceID = tracedMsg.Tracing.TraceID
		receiveTraceInfo.ParentSpanID = tracedMsg.Tracing.SpanID

		// Encode the payload
		payloadBytes, err := json.Marshal(tracedMsg.Payload)
		if err != nil {
			receiveTraceInfo.Status = "error"
			receiveTraceInfo.Error = err.Error()
			tr.TracingCallback(receiveTraceInfo)
			return err
		}

		// Call the original callback with the payload
		err = callback(payloadBytes)

		// Update tracing information
		receiveTraceInfo.EndTime = time.Now()
		receiveTraceInfo.Duration = receiveTraceInfo.EndTime.Sub(receiveTraceInfo.StartTime).Milliseconds()
		if err != nil {
			receiveTraceInfo.Status = "error"
			receiveTraceInfo.Error = err.Error()
		}

		// Call tracing callback
		tr.TracingCallback(receiveTraceInfo)

		return err
	}

	if tr.ContextRouter != nil {
		return tr.ContextRouter.SubscribeWithContext(ctx, pipe, wrapper)
	}
	return tr.Router.Subscribe(pipe, wrapper)
}

// SetTracingEnabled sets whether tracing is enabled.
func (tr *TracedRouter) SetTracingEnabled(enabled bool) {
	tr.TracingEnabled = enabled
}

// SetTracingCallback sets the callback function for completed traces.
func (tr *TracedRouter) SetTracingCallback(callback func(TracingInfo)) {
	tr.TracingCallback = callback
}

// generateID generates a unique ID for traces and spans.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// defaultTracingCallback is the default callback for completed traces.
func defaultTracingCallback(info TracingInfo) {
	log.Printf("Trace: %s, Span: %s, Operation: %s, Duration: %dms, Status: %s",
		info.TraceID, info.SpanID, info.Operation, info.Duration, info.Status)
}
