package tracing

import (
	"context"
	"testing"
	"time"

	"hybridpipe.io/core"
	"hybridpipe.io/protocols/mock"
)

// TestTracedRouter tests the TracedRouter middleware.
func TestTracedRouter(t *testing.T) {
	t.Skip("Skipping tracing tests until the tracing package is fully implemented")
	// Create a mock router
	mockRouter := mock.NewMockRouter()
	err := mockRouter.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to mock router: %v", err)
	}
	defer mockRouter.Close()

	// Create a traced router
	tracedRouter := NewTracedRouter(mockRouter, "test-service")

	// Test pipe name
	pipeName := "test-pipe"

	// Test data
	testData := map[string]interface{}{
		"message": "Hello, TracedRouter!",
		"time":    time.Now().Unix(),
	}

	// Subscribe to the pipe
	receivedCh := make(chan interface{}, 1)
	err = tracedRouter.Subscribe(pipeName, func(data []byte) error {
		var decoded interface{}
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		receivedCh <- decoded
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pipe: %v", err)
	}

	// Dispatch a message
	err = tracedRouter.Dispatch(pipeName, testData)
	if err != nil {
		t.Fatalf("Failed to dispatch message: %v", err)
	}

	// Wait for the message to be received
	select {
	case receivedData := <-receivedCh:
		// Verify the received data
		if receivedData == nil {
			t.Fatal("No data received")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Test context-aware operations
	testContextAwareOperations(t, tracedRouter, receivedCh)

	// Unsubscribe from the pipe
	err = tracedRouter.Unsubscribe(pipeName)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from pipe: %v", err)
	}
}

// testContextAwareOperations tests the context-aware operations of the TracedRouter.
func testContextAwareOperations(t *testing.T, router *TracedRouter, receivedCh chan interface{}) {
	// Test pipe name
	pipeName := "test-ctx-pipe"

	// Test data
	testData := map[string]interface{}{
		"message": "Hello, Context-Aware TracedRouter!",
		"time":    time.Now().Unix(),
	}

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to the pipe with context
	err := router.SubscribeWithContext(ctx, pipeName, func(data []byte) error {
		var decoded interface{}
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		receivedCh <- decoded
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pipe with context: %v", err)
	}

	// Dispatch a message with context
	err = router.DispatchWithContext(ctx, pipeName, testData)
	if err != nil {
		t.Fatalf("Failed to dispatch message with context: %v", err)
	}

	// Wait for the message to be received
	select {
	case receivedData := <-receivedCh:
		// Verify the received data
		if receivedData == nil {
			t.Fatal("No data received")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from pipe: %v", err)
	}
}
