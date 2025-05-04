package nats

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hybridpipe.io/core"
)

func TestNATSRouter(t *testing.T) {
	t.Skip("Skipping NATS tests until the NATS protocol is fully implemented")

	// Skip the test for now
	t.Skip("NATS protocol not implemented yet")

}

// testBasicOperations tests the basic operations of the NATS router.
func testBasicOperations(t *testing.T, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-pipe"

	// Test data
	testData := map[string]interface{}{
		"message": "Hello, NATS!",
		"time":    time.Now().Unix(),
	}

	// Wait group for synchronization
	done := make(chan struct{})

	// Subscribe to the pipe
	var receivedData interface{}
	err := router.Subscribe(pipeName, func(data []byte) error {
		var decoded interface{}
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		receivedData = decoded
		close(done)
		return nil
	})
	require.NoError(t, err, "Failed to subscribe to pipe")

	// Dispatch a message
	err = router.Dispatch(pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message")

	// Wait for the message to be received
	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Verify the received data
	assert.NotNil(t, receivedData, "No data received")

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")
}

// testContextAwareOperations tests the context-aware operations of the NATS router.
func testContextAwareOperations(t *testing.T, router core.HybridPipe) {
	// Check if the router implements the ContextAwareHybridPipe interface
	ctxRouter, ok := router.(core.ContextAwareHybridPipe)
	if !ok {
		t.Skip("Router does not implement ContextAwareHybridPipe")
		return
	}

	// Test pipe name
	pipeName := "test-ctx-pipe"

	// Test data
	testData := map[string]interface{}{
		"message": "Hello, Context-Aware NATS!",
		"time":    time.Now().Unix(),
	}

	// Wait group for synchronization
	done := make(chan struct{})

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to the pipe with context
	var receivedData interface{}
	err := ctxRouter.SubscribeWithContext(ctx, pipeName, func(data []byte) error {
		var decoded interface{}
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		receivedData = decoded
		close(done)
		return nil
	})
	require.NoError(t, err, "Failed to subscribe to pipe with context")

	// Dispatch a message with context
	err = ctxRouter.DispatchWithContext(ctx, pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message with context")

	// Wait for the message to be received
	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Verify the received data
	assert.NotNil(t, receivedData, "No data received")

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")
}
