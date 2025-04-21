package hybridpipe

import (
	"context"
	"testing"
	"time"
)

// TestTracingBasic tests the basic tracing functionality
func TestTracingBasic(t *testing.T) {
	// Skip the test in short mode
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	// Initialize the tracer
	tracer, err := NewTracer(Config{
		ServiceName:  "test-service",
		SamplingRate: 1.0, // Sample all messages
		Exporter: ExporterConfig{
			Type:     "console", // Use console exporter for demonstration
			Endpoint: "",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	// Use a nil meter for testing
	var meter Meter

	// Create a mock router for testing
	router := &mockRouter{}

	// Wrap the router with tracing and metrics middleware
	tracedRouter := WrapRouterWithMetrics(router, tracer, meter)

	// Create a channel to signal when the message is received
	messageReceived := make(chan struct{})

	// Define a message handler with tracing
	handleMessage := func(data any) {
		t.Logf("Received message: %v", data)
		// Signal that the message was received
		close(messageReceived)
	}

	// Subscribe to a pipe
	err = tracedRouter.Accept("test-pipe", handleMessage)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Send a message
	message := map[string]any{
		"content":   "Hello, world!",
		"timestamp": time.Now().Unix(),
	}
	err = tracedRouter.Dispatch("test-pipe", message)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for the message to be processed with a timeout
	select {
	case <-messageReceived:
		// Message was received successfully
		t.Log("Message received successfully")
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for message to be received")
	}

	// Close the router
	tracedRouter.Close()
}

// mockRouter is a mock implementation of the HybridPipe interface for testing
type mockRouter struct {
	handlers map[string]Process
}

// Connect establishes a connection to the messaging system
func (r *mockRouter) Connect() error {
	r.handlers = make(map[string]Process)
	return nil
}

// Dispatch sends a message to the specified pipe
func (r *mockRouter) Dispatch(pipe string, data any) error {
	if handler, ok := r.handlers[pipe]; ok {
		go handler(data)
	}
	return nil
}

// Accept subscribes to messages from the specified pipe
func (r *mockRouter) Accept(pipe string, fn Process) error {
	if r.handlers == nil {
		r.handlers = make(map[string]Process)
	}
	r.handlers[pipe] = fn
	return nil
}

// Remove unsubscribes from the specified pipe
func (r *mockRouter) Remove(pipe string) error {
	delete(r.handlers, pipe)
	return nil
}

// Close terminates the connection to the messaging system
func (r *mockRouter) Close() {
	r.handlers = nil
}
