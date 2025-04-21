package hybridpipe

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Exporter is the interface for exporting spans
type Exporter interface {
	// Export exports a span
	Export(span ExportedSpan)
	// Shutdown shuts down the exporter
	Shutdown(ctx context.Context) error
}

// NewExporter creates a new exporter
func NewExporter(config ExporterConfig) (Exporter, error) {
	switch config.Type {
	case "jaeger":
		return newJaegerExporter(config)
	case "otlp":
		return newOTLPExporter(config)
	case "zipkin":
		return newZipkinExporter(config)
	case "console":
		return newConsoleExporter(config)
	case "":
		return &noopExporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", config.Type)
	}
}

// noopExporter is an exporter that does nothing
type noopExporter struct{}

// Export does nothing
func (e *noopExporter) Export(span ExportedSpan) {}

// Shutdown does nothing
func (e *noopExporter) Shutdown(ctx context.Context) error {
	return nil
}

// batchExporter is a base exporter that batches spans
type batchExporter struct {
	// config is the configuration for the exporter
	config ExporterConfig
	// spans is the batch of spans to export
	spans []ExportedSpan
	// mu protects the spans slice
	mu sync.Mutex
	// done is a channel that is closed when the exporter is shut down
	done chan struct{}
	// ticker is a ticker that triggers batch exports
	ticker *time.Ticker
}

// newBatchExporter creates a new batch exporter
func newBatchExporter(config ExporterConfig) *batchExporter {
	// Set default values
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Second
	}
	if config.MaxQueueSize <= 0 {
		config.MaxQueueSize = 1000
	}

	exporter := &batchExporter{
		config: config,
		spans:  make([]ExportedSpan, 0, config.BatchSize),
		done:   make(chan struct{}),
		ticker: time.NewTicker(1 * time.Second),
	}

	// Start a goroutine to export batches periodically
	go func() {
		for {
			select {
			case <-exporter.done:
				return
			case <-exporter.ticker.C:
				exporter.exportBatch()
			}
		}
	}()

	return exporter
}

// Export adds a span to the batch
func (e *batchExporter) Export(span ExportedSpan) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Add the span to the batch
	e.spans = append(e.spans, span)

	// Export the batch if it's full
	if len(e.spans) >= e.config.BatchSize {
		e.exportBatchLocked()
	}
}

// Shutdown exports any remaining spans and shuts down the exporter
func (e *batchExporter) Shutdown(ctx context.Context) error {
	// Stop the ticker
	e.ticker.Stop()

	// Signal the goroutine to stop
	close(e.done)

	// Export any remaining spans
	e.exportBatch()

	return nil
}

// exportBatch exports the current batch of spans
func (e *batchExporter) exportBatch() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.exportBatchLocked()
}

// exportBatchLocked exports the current batch of spans (must be called with the lock held)
func (e *batchExporter) exportBatchLocked() {
	if len(e.spans) == 0 {
		return
	}

	// Create a copy of the spans
	spans := make([]ExportedSpan, len(e.spans))
	copy(spans, e.spans)

	// Clear the batch
	e.spans = e.spans[:0]

	// Export the spans in a separate goroutine
	go e.doExport(spans)
}

// doExport exports the spans
func (e *batchExporter) doExport(spans []ExportedSpan) {
	// This method should be implemented by the concrete exporter
	// Default implementation does nothing
}

// jaegerExporter exports spans to Jaeger
type jaegerExporter struct {
	*batchExporter
	// endpoint is the Jaeger endpoint
	endpoint string
	// client is the HTTP client
	client *http.Client
}

// newJaegerExporter creates a new Jaeger exporter
func newJaegerExporter(config ExporterConfig) (*jaegerExporter, error) {
	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:14268/api/traces"
	}

	return &jaegerExporter{
		batchExporter: newBatchExporter(config),
		endpoint:      config.Endpoint,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// jaegerExporter implementation

// otlpExporter exports spans using the OpenTelemetry Protocol
type otlpExporter struct {
	*batchExporter
	// endpoint is the OTLP endpoint
	endpoint string
	// client is the HTTP client
	client *http.Client
}

// newOTLPExporter creates a new OTLP exporter
func newOTLPExporter(config ExporterConfig) (*otlpExporter, error) {
	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:4318/v1/traces"
	}

	return &otlpExporter{
		batchExporter: newBatchExporter(config),
		endpoint:      config.Endpoint,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// otlpExporter implementation

// zipkinExporter exports spans to Zipkin
type zipkinExporter struct {
	*batchExporter
	// endpoint is the Zipkin endpoint
	endpoint string
	// client is the HTTP client
	client *http.Client
}

// newZipkinExporter creates a new Zipkin exporter
func newZipkinExporter(config ExporterConfig) (*zipkinExporter, error) {
	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:9411/api/v2/spans"
	}

	return &zipkinExporter{
		batchExporter: newBatchExporter(config),
		endpoint:      config.Endpoint,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// zipkinExporter implementation

// consoleExporter exports spans to the console
type consoleExporter struct {
	*batchExporter
}

// newConsoleExporter creates a new console exporter
func newConsoleExporter(config ExporterConfig) (*consoleExporter, error) {
	return &consoleExporter{
		batchExporter: newBatchExporter(config),
	}, nil
}

// consoleExporter implementation
