package dispatch

import (
	"context"
	"encoding/gob"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hybridpipe.io/core"
	"hybridpipe.io/protocols/mock"
)

func TestDispatcher(t *testing.T) {
	// Create a mock router
	router := mock.NewMockRouter()
	err := router.Connect()
	require.NoError(t, err, "Failed to connect to mock router")
	defer router.Close()

	// Create a dispatcher
	dispatcher := NewDispatcher(router)

	// Test dispatching messages
	testDispatchMessages(t, dispatcher, router)

	// Test batch dispatching
	testBatchDispatching(t, dispatcher, router)

	// Test context-aware dispatching
	testContextAwareDispatching(t, dispatcher, router)

	// Test batch dispatching with context
	testBatchDispatchingWithContext(t, dispatcher, router)

	// Test GetRouter method
	testGetRouter(t, dispatcher, router)
}

// testDispatchMessages tests dispatching messages through the dispatcher.
func testDispatchMessages(t *testing.T, dispatcher *Dispatcher, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-dispatch-pipe"

	// Test data - use a simple string instead of a map to avoid serialization issues
	testData := "Hello, Dispatcher!"

	// Wait group for synchronization
	done := make(chan struct{})

	// Subscribe to the pipe through the router
	err := router.Subscribe(pipeName, func(data []byte) error {
		var decoded string
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		assert.Equal(t, testData, decoded, "Received message does not match sent message")
		close(done)
		return nil
	})
	require.NoError(t, err, "Failed to subscribe to pipe")

	// Dispatch a message through the dispatcher
	err = dispatcher.Dispatch(pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message")

	// Wait for the message to be received
	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")
}

// testBatchDispatching tests batch dispatching of messages.
func testBatchDispatching(t *testing.T, dispatcher *Dispatcher, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-batch-dispatch-pipe"

	// Test data - use simple strings to avoid serialization issues
	testData := []interface{}{
		"Hello, Batch Dispatcher! 1",
		"Hello, Batch Dispatcher! 2",
		"Hello, Batch Dispatcher! 3",
	}

	// Counter for received messages
	receivedCount := 0
	var mu sync.Mutex

	// Wait group for synchronization
	done := make(chan struct{})

	// Register types for Gob
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})

	// Subscribe to the pipe through the router
	err := router.Subscribe(pipeName, func(data []byte) error {
		var decoded string
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}

		mu.Lock()
		receivedCount++
		if receivedCount == len(testData) {
			close(done)
		}
		mu.Unlock()

		return nil
	})
	require.NoError(t, err, "Failed to subscribe to pipe")

	// Dispatch a batch of messages through the dispatcher
	err = dispatcher.DispatchBatch(pipeName, testData)
	require.NoError(t, err, "Failed to dispatch batch of messages")

	// Wait for all messages to be received
	select {
	case <-done:
		// All messages received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for messages")
	}

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")
}

// testBatchDispatchingWithContext tests batch dispatching of messages with context.
func testBatchDispatchingWithContext(t *testing.T, dispatcher *Dispatcher, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-batch-ctx-dispatch-pipe"

	// Test data - use simple strings to avoid serialization issues
	testData := []interface{}{
		"Hello, Batch Context Dispatcher! 1",
		"Hello, Batch Context Dispatcher! 2",
		"Hello, Batch Context Dispatcher! 3",
	}

	// Counter for received messages
	receivedCount := 0
	var mu sync.Mutex

	// Wait group for synchronization
	done := make(chan struct{})

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register types for Gob
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})

	// Subscribe to the pipe through the router
	if ctxRouter, ok := router.(core.ContextAwareHybridPipe); ok {
		err := ctxRouter.SubscribeWithContext(ctx, pipeName, func(data []byte) error {
			var decoded string
			if err := core.Decode(data, &decoded); err != nil {
				t.Errorf("Failed to decode message: %v", err)
				return err
			}

			mu.Lock()
			receivedCount++
			if receivedCount == len(testData) {
				close(done)
			}
			mu.Unlock()

			return nil
		})
		require.NoError(t, err, "Failed to subscribe to pipe with context")
	} else {
		err := router.Subscribe(pipeName, func(data []byte) error {
			var decoded string
			if err := core.Decode(data, &decoded); err != nil {
				t.Errorf("Failed to decode message: %v", err)
				return err
			}

			mu.Lock()
			receivedCount++
			if receivedCount == len(testData) {
				close(done)
			}
			mu.Unlock()

			return nil
		})
		require.NoError(t, err, "Failed to subscribe to pipe")
	}

	// Dispatch a batch of messages through the dispatcher with context
	err := dispatcher.DispatchBatchWithContext(ctx, pipeName, testData)
	require.NoError(t, err, "Failed to dispatch batch of messages with context")

	// Wait for all messages to be received
	select {
	case <-done:
		// All messages received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for messages")
	}

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")
}

// testGetRouter tests the GetRouter method.
func testGetRouter(t *testing.T, dispatcher *Dispatcher, router core.HybridPipe) {
	// Get the router from the dispatcher
	gotRouter := dispatcher.GetRouter()

	// Check that the router is the same as the one we passed in
	assert.Equal(t, router, gotRouter, "GetRouter should return the router passed to NewDispatcher")
}

// testContextAwareDispatching tests context-aware dispatching of messages.
func testContextAwareDispatching(t *testing.T, dispatcher *Dispatcher, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-ctx-dispatch-pipe"

	// Test data - use a simple string instead of a map to avoid serialization issues
	testData := "Hello, Context-Aware Dispatcher!"

	// Wait group for synchronization
	done := make(chan struct{})

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to the pipe through the router
	if ctxRouter, ok := router.(core.ContextAwareHybridPipe); ok {
		err := ctxRouter.SubscribeWithContext(ctx, pipeName, func(data []byte) error {
			var decoded string
			if err := core.Decode(data, &decoded); err != nil {
				t.Errorf("Failed to decode message: %v", err)
				return err
			}
			assert.Equal(t, testData, decoded, "Received message does not match sent message")
			close(done)
			return nil
		})
		require.NoError(t, err, "Failed to subscribe to pipe with context")
	} else {
		err := router.Subscribe(pipeName, func(data []byte) error {
			var decoded string
			if err := core.Decode(data, &decoded); err != nil {
				t.Errorf("Failed to decode message: %v", err)
				return err
			}
			assert.Equal(t, testData, decoded, "Received message does not match sent message")
			close(done)
			return nil
		})
		require.NoError(t, err, "Failed to subscribe to pipe")
	}

	// Dispatch a message through the dispatcher with context
	err := dispatcher.DispatchWithContext(ctx, pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message with context")

	// Wait for the message to be received
	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")
}
