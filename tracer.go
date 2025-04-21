package hybridpipe

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StatusCode represents the status of a span
type StatusCode int

const (
	// StatusUnset is the default status
	StatusUnset StatusCode = 0
	// StatusOK indicates success
	StatusOK StatusCode = 1
	// StatusError indicates an error
	StatusError StatusCode = 2
)

// Config contains configuration for the tracer
type Config struct {
	// Enabled indicates whether tracing is enabled
	Enabled bool
	// ServiceName is the name of the service
	ServiceName string
	// SamplingRate is the rate at which traces are sampled (0.0-1.0)
	SamplingRate float64
	// Exporter is the configuration for the exporter
	Exporter ExporterConfig
	// Storage is the configuration for the storage backend
	Storage StorageConfig
}

// ExporterConfig contains configuration for the exporter
type ExporterConfig struct {
	// Type is the type of exporter (e.g., "jaeger", "otlp", "zipkin")
	Type string
	// Endpoint is the endpoint for the exporter
	Endpoint string
	// Headers are additional headers to include in export requests
	Headers map[string]string
	// Timeout is the timeout for export requests
	Timeout time.Duration
	// BatchSize is the number of spans to batch before exporting
	BatchSize int
	// MaxQueueSize is the maximum number of spans to queue
	MaxQueueSize int
}

// StorageConfig contains configuration for the storage backend
type StorageConfig struct {
	// Type is the type of storage backend (e.g., "memory", "file", "postgres")
	Type string
	// ConnectionString is the connection string for the storage backend
	ConnectionString string
	// RetentionPeriod is the period for which trace data is retained
	RetentionPeriod time.Duration
}

// StartOption is an option for starting a span
type StartOption func(*startOptions)

// startOptions contains options for starting a span
type startOptions struct {
	// Attributes are key-value pairs to add to the span
	Attributes map[string]interface{}
	// StartTime is the time at which the span starts
	StartTime time.Time
	// Links are links to other spans
	Links []Link
}

// WithAttributes adds attributes to a span
func WithAttributes(attributes map[string]interface{}) StartOption {
	return func(opts *startOptions) {
		for k, v := range attributes {
			opts.Attributes[k] = v
		}
	}
}

// WithStartTime sets the start time of a span
func WithStartTime(startTime time.Time) StartOption {
	return func(opts *startOptions) {
		opts.StartTime = startTime
	}
}

// WithLinks adds links to a span
func WithLinks(links []Link) StartOption {
	return func(opts *startOptions) {
		opts.Links = append(opts.Links, links...)
	}
}

// Link represents a link to another span
type Link struct {
	// TraceID is the trace ID of the linked span
	TraceID string
	// SpanID is the span ID of the linked span
	SpanID string
	// Attributes are key-value pairs associated with the link
	Attributes map[string]interface{}
}

// SpanContext contains the context of a span
type SpanContext struct {
	// TraceID is the ID of the trace that this span belongs to
	TraceID string
	// SpanID is the ID of this span
	SpanID string
	// ParentSpanID is the ID of the parent span
	ParentSpanID string
	// TraceFlags contains flags for the trace
	TraceFlags byte
	// TraceState contains vendor-specific trace information
	TraceState string
	// Remote indicates whether the span context was propagated from a remote parent
	Remote bool
}

// Tracer is the interface for creating and managing spans
type Tracer interface {
	// Start starts a new span
	Start(ctx context.Context, name string, opts ...StartOption) (context.Context, Span)
	// Extract extracts a span context from a carrier
	Extract(ctx context.Context, carrier interface{}) context.Context
	// Inject injects a span context into a carrier
	Inject(ctx context.Context, carrier interface{})
	// Shutdown shuts down the tracer
	Shutdown(ctx context.Context) error
}

// Span is the interface for a span
type Span interface {
	// End ends the span
	End()
	// SetAttribute sets an attribute on the span
	SetAttribute(key string, value interface{})
	// AddEvent adds an event to the span
	AddEvent(name string, attributes map[string]interface{})
	// SetStatus sets the status of the span
	SetStatus(code StatusCode, description string)
	// RecordError records an error
	RecordError(err error)
	// SpanContext returns the span context
	SpanContext() SpanContext
}

// tracerImpl is the implementation of Tracer
type tracerImpl struct {
	// config is the configuration for the tracer
	config Config
	// exporter is the exporter for spans
	exporter Exporter
	// storage is the storage backend for spans
	storage Storage
	// idGenerator is the generator for trace and span IDs
	idGenerator IDGenerator
	// sampler is the sampler for traces
	sampler Sampler
	// propagator is the propagator for span contexts
	propagator Propagator
	// spans is a map of active spans
	spans sync.Map
}

// NewTracer creates a new tracer
func NewTracer(config Config) (Tracer, error) {
	// Create ID generator
	idGenerator := &randomIDGenerator{}

	// Create sampler
	sampler := &probabilitySampler{
		probability: config.SamplingRate,
	}

	// Create propagator
	propagator := &w3cPropagator{}

	// Create exporter
	var exporter Exporter
	var err error
	if config.Exporter.Type != "" {
		exporter, err = NewExporter(config.Exporter)
		if err != nil {
			return nil, fmt.Errorf("failed to create exporter: %w", err)
		}
	} else {
		// Use a no-op exporter if none is configured
		exporter = &noopExporter{}
	}

	// Create storage
	var storage Storage
	if config.Storage.Type != "" {
		storage, err = NewStorage(config.Storage)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage: %w", err)
		}
	} else {
		// Use in-memory storage if none is configured
		storage = &memoryStorage{}
	}

	return &tracerImpl{
		config:      config,
		exporter:    exporter,
		storage:     storage,
		idGenerator: idGenerator,
		sampler:     sampler,
		propagator:  propagator,
	}, nil
}

// Start starts a new span
func (t *tracerImpl) Start(ctx context.Context, name string, opts ...StartOption) (context.Context, Span) {
	// Apply options
	startOpts := &startOptions{
		Attributes: make(map[string]interface{}),
		StartTime:  time.Now(),
	}
	for _, opt := range opts {
		opt(startOpts)
	}

	// Get parent span context from context
	parentSpanContext := SpanContextFromContext(ctx)

	// Generate span ID
	spanID := t.idGenerator.GenerateSpanID()

	// Create span context
	var traceID string
	var parentSpanID string
	var traceFlags byte
	if parentSpanContext.TraceID != "" {
		// This is a child span
		traceID = parentSpanContext.TraceID
		parentSpanID = parentSpanContext.SpanID
		traceFlags = parentSpanContext.TraceFlags
	} else {
		// This is a root span
		traceID = t.idGenerator.GenerateTraceID()
		// Sample the trace
		if t.sampler.ShouldSample(traceID, name, startOpts.Attributes) {
			traceFlags = 1 // Sampled
		}
	}

	spanContext := SpanContext{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		TraceFlags:   traceFlags,
	}

	// Create span
	span := &spanImpl{
		tracer:      t,
		name:        name,
		startTime:   startOpts.StartTime,
		attributes:  startOpts.Attributes,
		events:      make([]Event, 0),
		links:       startOpts.Links,
		spanContext: spanContext,
	}

	// Store span
	t.spans.Store(spanID, span)

	// Create new context with span context
	newCtx := ContextWithSpanContext(ctx, spanContext)

	return newCtx, span
}

// Extract extracts a span context from a carrier
func (t *tracerImpl) Extract(ctx context.Context, carrier interface{}) context.Context {
	spanContext := t.propagator.Extract(carrier)
	if spanContext.TraceID == "" {
		// No span context in carrier
		return ctx
	}
	return ContextWithSpanContext(ctx, spanContext)
}

// Inject injects a span context into a carrier
func (t *tracerImpl) Inject(ctx context.Context, carrier interface{}) {
	spanContext := SpanContextFromContext(ctx)
	t.propagator.Inject(spanContext, carrier)
}

// Shutdown shuts down the tracer
func (t *tracerImpl) Shutdown(ctx context.Context) error {
	// Export any remaining spans
	t.spans.Range(func(key, value interface{}) bool {
		span := value.(*spanImpl)
		span.End()
		return true
	})

	// Shutdown exporter
	if err := t.exporter.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown exporter: %w", err)
	}

	// Shutdown storage
	if err := t.storage.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown storage: %w", err)
	}

	return nil
}

// Event represents an event that occurred during a span
type Event struct {
	// Name is the name of the event
	Name string
	// Attributes are key-value pairs associated with the event
	Attributes map[string]interface{}
	// Timestamp is the time at which the event occurred
	Timestamp time.Time
}

// spanImpl is the implementation of Span
type spanImpl struct {
	// tracer is the tracer that created this span
	tracer *tracerImpl
	// name is the name of the span
	name string
	// startTime is the time at which the span started
	startTime time.Time
	// endTime is the time at which the span ended
	endTime time.Time
	// attributes are key-value pairs associated with the span
	attributes map[string]interface{}
	// events are events that occurred during the span
	events []Event
	// links are links to other spans
	links []Link
	// status is the status of the span
	status struct {
		// code is the status code
		code StatusCode
		// description is a description of the status
		description string
	}
	// spanContext is the context of the span
	spanContext SpanContext
	// ended indicates whether the span has ended
	ended bool
	// mu protects the span from concurrent modification
	mu sync.Mutex
}

// End ends the span
func (s *spanImpl) End() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ended {
		return
	}

	s.endTime = time.Now()
	s.ended = true

	// Export the span
	s.tracer.exporter.Export(s.toExportedSpan())

	// Store the span
	s.tracer.storage.Store(s.toExportedSpan())

	// Remove the span from the tracer's active spans
	s.tracer.spans.Delete(s.spanContext.SpanID)
}

// SetAttribute sets an attribute on the span
func (s *spanImpl) SetAttribute(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ended {
		return
	}

	if s.attributes == nil {
		s.attributes = make(map[string]interface{})
	}

	s.attributes[key] = value
}

// AddEvent adds an event to the span
func (s *spanImpl) AddEvent(name string, attributes map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ended {
		return
	}

	s.events = append(s.events, Event{
		Name:       name,
		Attributes: attributes,
		Timestamp:  time.Now(),
	})
}

// SetStatus sets the status of the span
func (s *spanImpl) SetStatus(code StatusCode, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ended {
		return
	}

	s.status.code = code
	s.status.description = description
}

// RecordError records an error
func (s *spanImpl) RecordError(err error) {
	if err == nil {
		return
	}

	s.AddEvent("error", map[string]interface{}{
		"error.message": err.Error(),
	})
}

// SpanContext returns the span context
func (s *spanImpl) SpanContext() SpanContext {
	return s.spanContext
}

// toExportedSpan converts the span to an exported span
func (s *spanImpl) toExportedSpan() ExportedSpan {
	s.mu.Lock()
	defer s.mu.Unlock()

	endTime := s.endTime
	if endTime.IsZero() {
		endTime = time.Now()
	}

	return ExportedSpan{
		TraceID:      s.spanContext.TraceID,
		SpanID:       s.spanContext.SpanID,
		ParentSpanID: s.spanContext.ParentSpanID,
		Name:         s.name,
		StartTime:    s.startTime,
		EndTime:      endTime,
		Attributes:   s.attributes,
		Events:       s.events,
		Links:        s.links,
		Status: Status{
			Code:        s.status.code,
			Description: s.status.description,
		},
		ServiceName: s.tracer.config.ServiceName,
	}
}

// ExportedSpan represents a span that has been exported
type ExportedSpan struct {
	// TraceID is the ID of the trace that this span belongs to
	TraceID string
	// SpanID is the ID of this span
	SpanID string
	// ParentSpanID is the ID of the parent span
	ParentSpanID string
	// Name is the name of the span
	Name string
	// StartTime is the time at which the span started
	StartTime time.Time
	// EndTime is the time at which the span ended
	EndTime time.Time
	// Attributes are key-value pairs associated with the span
	Attributes map[string]interface{}
	// Events are events that occurred during the span
	Events []Event
	// Links are links to other spans
	Links []Link
	// Status is the status of the span
	Status Status
	// ServiceName is the name of the service that created the span
	ServiceName string
}

// Status represents the status of a span
type Status struct {
	// Code is the status code
	Code StatusCode
	// Description is a description of the status
	Description string
}

// contextKey is the type of the keys used in the context
type contextKey int

const (
	// spanContextKey is the key for the span context in the context
	spanContextKey contextKey = iota
)

// ContextWithSpanContext returns a new context with the given span context
func ContextWithSpanContext(ctx context.Context, spanContext SpanContext) context.Context {
	return context.WithValue(ctx, spanContextKey, spanContext)
}

// SpanContextFromContext returns the span context from the context
func SpanContextFromContext(ctx context.Context) SpanContext {
	if ctx == nil {
		return SpanContext{}
	}
	if sc, ok := ctx.Value(spanContextKey).(SpanContext); ok {
		return sc
	}
	return SpanContext{}
}
