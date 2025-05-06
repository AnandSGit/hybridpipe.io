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
	// Skip in short mode as this test requires a NATS server
	if testing.Short() {
		t.Skip("Skipping NATS tests in short mode")
	}

	// Create a mock NATS config
	config := &core.NATSConfig{
		NServer:        "localhost",
		NLport:         4222,
		NReconnectWait: 1,
		NMaxReconnects: 3,
	}

	// Create a new NATS router
	router := New(config)

	// Test the router
	t.Run("RouterImplementsInterfaces", func(t *testing.T) {
		testRouterImplementsInterfaces(t, router)
	})

	// Test connection methods without an actual server
	t.Run("ConnectionMethods", func(t *testing.T) {
		testConnectionMethods(t, router)
	})

	// Test error handling
	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, router)
	})
}

// testRouterImplementsInterfaces tests that the router implements the required interfaces.
func testRouterImplementsInterfaces(t *testing.T, router *Packet) {
	// Test that the router implements HybridPipe
	var _ core.HybridPipe = router

	// Test that the router implements ContextAwareHybridPipe
	var _ core.ContextAwareHybridPipe = router
}

// testConnectionMethods tests the connection-related methods of the NATS router.
func testConnectionMethods(t *testing.T, router *Packet) {
	// Initially, the router should not be connected
	assert.False(t, router.IsConnected(), "Router should not be connected initially")

	// Test that Disconnect works even when not connected
	err := router.Disconnect()
	assert.NoError(t, err, "Disconnect should not return an error when not connected")

	// Test that Close works even when not connected
	err = router.Close()
	assert.NoError(t, err, "Close should not return an error when not connected")
}

// testErrorHandling tests the error handling of the NATS router.
func testErrorHandling(t *testing.T, router *Packet) {
	// Test dispatching when not connected
	err := router.Dispatch("test-pipe", "test-data")
	assert.Error(t, err, "Dispatch should return an error when not connected")
	assert.Equal(t, core.ErrNotConnected, err, "Dispatch should return ErrNotConnected when not connected")

	// Test dispatching with context when not connected
	ctx := context.Background()
	err = router.DispatchWithContext(ctx, "test-pipe", "test-data")
	assert.Error(t, err, "DispatchWithContext should return an error when not connected")
	assert.Equal(t, core.ErrNotConnected, err, "DispatchWithContext should return ErrNotConnected when not connected")

	// Test subscribing when not connected
	err = router.Subscribe("test-pipe", func(data []byte) error {
		return nil
	})
	assert.Error(t, err, "Subscribe should return an error when not connected")
	assert.Equal(t, core.ErrNotConnected, err, "Subscribe should return ErrNotConnected when not connected")

	// Test subscribing with context when not connected
	err = router.SubscribeWithContext(ctx, "test-pipe", func(data []byte) error {
		return nil
	})
	assert.Error(t, err, "SubscribeWithContext should return an error when not connected")
	assert.Equal(t, core.ErrNotConnected, err, "SubscribeWithContext should return ErrNotConnected when not connected")

	// Test unsubscribing when not connected
	err = router.Unsubscribe("test-pipe")
	assert.Error(t, err, "Unsubscribe should return an error when not connected")
	assert.Equal(t, core.ErrNotConnected, err, "Unsubscribe should return ErrNotConnected when not connected")

	// Test removing when not connected
	err = router.Remove("test-pipe")
	assert.Error(t, err, "Remove should return an error when not connected")
	assert.Equal(t, core.ErrNotConnected, err, "Remove should return ErrNotConnected when not connected")
}

// TestNATSAccept tests the Accept method of the NATS router.
func TestNATSAccept(t *testing.T) {
	// Skip in short mode as this test requires a NATS server
	if testing.Short() {
		t.Skip("Skipping NATS Accept test in short mode")
	}

	// Create a mock NATS config
	config := &core.NATSConfig{
		NServer:        "localhost",
		NLport:         4222,
		NReconnectWait: 1,
		NMaxReconnects: 3,
	}

	// Create a new NATS router
	router := New(config)

	// Test the Accept method
	pipeName := "test-accept-pipe"

	// Accept messages from the pipe
	err := router.Accept(pipeName, func(data interface{}) {
		// This function won't be called because we're not connected
	})

	// This should fail because we're not connected
	assert.Error(t, err, "Accept should return an error when not connected")
}

// TestNATSWithMockServer tests the NATS router with a mock server.
func TestNATSWithMockServer(t *testing.T) {
	// Skip this test as it requires a mock NATS server
	t.Skip("Skipping test that requires a mock NATS server")

	// Create a mock NATS config
	config := &core.NATSConfig{
		NServer:        "localhost",
		NLport:         4222,
		NReconnectWait: 1,
		NMaxReconnects: 3,
	}

	// Create a new NATS router
	router := New(config)

	// Connect to the mock server
	err := router.Connect()
	require.NoError(t, err, "Failed to connect to mock NATS server")
	defer router.Close()

	// Test basic operations
	testBasicOperations(t, router)

	// Test context-aware operations
	testContextAwareOperations(t, router)
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
