package hybridpipe

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Storage is the interface for storing spans
type Storage interface {
	// Store stores a span
	Store(span ExportedSpan) error
	// Query queries for spans
	Query(query Query) ([]ExportedSpan, error)
	// Shutdown shuts down the storage
	Shutdown(ctx context.Context) error
}

// Query represents a query for spans
type Query struct {
	// TraceID is the ID of the trace to query
	TraceID string
	// SpanID is the ID of the span to query
	SpanID string
	// ServiceName is the name of the service to query
	ServiceName string
	// StartTime is the start time of the time range to query
	StartTime time.Time
	// EndTime is the end time of the time range to query
	EndTime time.Time
	// Limit is the maximum number of spans to return
	Limit int
	// Offset is the number of spans to skip
	Offset int
}

// NewStorage creates a new storage
func NewStorage(config StorageConfig) (Storage, error) {
	switch config.Type {
	case "memory":
		return newMemoryStorage(config)
	case "file":
		return newFileStorage(config)
	case "postgres":
		return newPostgresStorage(config)
	case "":
		return newMemoryStorage(config)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}
}

// memoryStorage stores spans in memory
type memoryStorage struct {
	// spans is a map of trace ID to spans
	spans map[string][]ExportedSpan
	// mu protects the spans map
	mu sync.RWMutex
	// config is the configuration for the storage
	config StorageConfig
	// cleanupTicker is a ticker that triggers cleanup of old spans
	cleanupTicker *time.Ticker
	// done is a channel that is closed when the storage is shut down
	done chan struct{}
}

// newMemoryStorage creates a new memory storage
func newMemoryStorage(config StorageConfig) (*memoryStorage, error) {
	// Set default values
	if config.RetentionPeriod <= 0 {
		config.RetentionPeriod = 24 * time.Hour
	}

	storage := &memoryStorage{
		spans:         make(map[string][]ExportedSpan),
		config:        config,
		cleanupTicker: time.NewTicker(1 * time.Hour),
		done:          make(chan struct{}),
	}

	// Start a goroutine to clean up old spans
	go func() {
		for {
			select {
			case <-storage.done:
				return
			case <-storage.cleanupTicker.C:
				storage.cleanup()
			}
		}
	}()

	return storage, nil
}

// Store stores a span
func (s *memoryStorage) Store(span ExportedSpan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get the spans for this trace
	spans, ok := s.spans[span.TraceID]
	if !ok {
		spans = make([]ExportedSpan, 0)
	}

	// Add the span
	spans = append(spans, span)

	// Store the spans
	s.spans[span.TraceID] = spans

	return nil
}

// Query queries for spans
func (s *memoryStorage) Query(query Query) ([]ExportedSpan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If a trace ID is specified, return only spans from that trace
	if query.TraceID != "" {
		spans, ok := s.spans[query.TraceID]
		if !ok {
			return nil, nil
		}

		// Filter by span ID if specified
		if query.SpanID != "" {
			for _, span := range spans {
				if span.SpanID == query.SpanID {
					return []ExportedSpan{span}, nil
				}
			}
			return nil, nil
		}

		// Filter by service name if specified
		if query.ServiceName != "" {
			filteredSpans := make([]ExportedSpan, 0)
			for _, span := range spans {
				if span.ServiceName == query.ServiceName {
					filteredSpans = append(filteredSpans, span)
				}
			}
			return filteredSpans, nil
		}

		// Filter by time range if specified
		if !query.StartTime.IsZero() || !query.EndTime.IsZero() {
			filteredSpans := make([]ExportedSpan, 0)
			for _, span := range spans {
				if (!query.StartTime.IsZero() && span.StartTime.Before(query.StartTime)) ||
					(!query.EndTime.IsZero() && span.EndTime.After(query.EndTime)) {
					continue
				}
				filteredSpans = append(filteredSpans, span)
			}
			return filteredSpans, nil
		}

		return spans, nil
	}

	// If no trace ID is specified, return all spans that match the query
	allSpans := make([]ExportedSpan, 0)
	for _, spans := range s.spans {
		for _, span := range spans {
			// Filter by span ID if specified
			if query.SpanID != "" && span.SpanID != query.SpanID {
				continue
			}

			// Filter by service name if specified
			if query.ServiceName != "" && span.ServiceName != query.ServiceName {
				continue
			}

			// Filter by time range if specified
			if (!query.StartTime.IsZero() && span.StartTime.Before(query.StartTime)) ||
				(!query.EndTime.IsZero() && span.EndTime.After(query.EndTime)) {
				continue
			}

			allSpans = append(allSpans, span)
		}
	}

	// Apply limit and offset
	if query.Offset > 0 && query.Offset < len(allSpans) {
		allSpans = allSpans[query.Offset:]
	}
	if query.Limit > 0 && query.Limit < len(allSpans) {
		allSpans = allSpans[:query.Limit]
	}

	return allSpans, nil
}

// Shutdown shuts down the storage
func (s *memoryStorage) Shutdown(ctx context.Context) error {
	// Stop the cleanup ticker
	s.cleanupTicker.Stop()

	// Signal the goroutine to stop
	close(s.done)

	return nil
}

// cleanup removes old spans
func (s *memoryStorage) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate the cutoff time
	cutoff := time.Now().Add(-s.config.RetentionPeriod)

	// Remove old spans
	for traceID, spans := range s.spans {
		// Check if all spans in the trace are older than the cutoff
		allOld := true
		for _, span := range spans {
			if span.EndTime.After(cutoff) {
				allOld = false
				break
			}
		}

		// Remove the trace if all spans are old
		if allOld {
			delete(s.spans, traceID)
		}
	}
}

// fileStorage stores spans in a file
type fileStorage struct {
	// config is the configuration for the storage
	config StorageConfig
	// file is the file to store spans in
	file string
}

// newFileStorage creates a new file storage
func newFileStorage(config StorageConfig) (*fileStorage, error) {
	if config.ConnectionString == "" {
		return nil, fmt.Errorf("file path not specified")
	}

	return &fileStorage{
		config: config,
		file:   config.ConnectionString,
	}, nil
}

// Store stores a span
func (s *fileStorage) Store(span ExportedSpan) error {
	// This is a simplified implementation
	// In a real implementation, we would append the span to the file
	return nil
}

// Query queries for spans
func (s *fileStorage) Query(query Query) ([]ExportedSpan, error) {
	// This is a simplified implementation
	// In a real implementation, we would read spans from the file and filter them
	return nil, nil
}

// Shutdown shuts down the storage
func (s *fileStorage) Shutdown(ctx context.Context) error {
	// This is a simplified implementation
	// In a real implementation, we would close the file
	return nil
}

// postgresStorage stores spans in a PostgreSQL database
type postgresStorage struct {
	// config is the configuration for the storage
	config StorageConfig
	// db is the database connection
	db interface{}
}

// newPostgresStorage creates a new PostgreSQL storage
func newPostgresStorage(config StorageConfig) (*postgresStorage, error) {
	if config.ConnectionString == "" {
		return nil, fmt.Errorf("connection string not specified")
	}

	// This is a simplified implementation
	// In a real implementation, we would connect to the database
	return &postgresStorage{
		config: config,
	}, nil
}

// Store stores a span
func (s *postgresStorage) Store(span ExportedSpan) error {
	// This is a simplified implementation
	// In a real implementation, we would insert the span into the database
	return nil
}

// Query queries for spans
func (s *postgresStorage) Query(query Query) ([]ExportedSpan, error) {
	// This is a simplified implementation
	// In a real implementation, we would query the database
	return nil, nil
}

// Shutdown shuts down the storage
func (s *postgresStorage) Shutdown(ctx context.Context) error {
	// This is a simplified implementation
	// In a real implementation, we would close the database connection
	return nil
}
