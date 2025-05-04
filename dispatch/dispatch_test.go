package dispatch

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hybridpipe.io/core"
	"hybridpipe.io/protocols/mock"
)

func TestDispatcher(t *testing.T) {
	t.Skip("Skipping dispatcher tests until the dispatch package is fully implemented")
	
	// Create a mock router
	router := mock.NewMockRouter()
	err := router.Connect()
	require.NoError(t, err, "Failed to connect to mock router")
	defer router.Close()
	
	// Create a dispatcher
	dispatcher := NewDispatcher(router)
	
	// Test dispatching messages
	testDispatchMessages(t, dispatcher, router)
	
	// Test concurrent dispatching
	testConcurrentDispatching(t, dispatcher, router)
	
	// Test context-aware dispatching
	testContextAwareDispatching(t, dispatcher, router)
}

// testDispatchMessages tests dispatching messages through the dispatcher.
func testDispatchMessages(t *testing.T, dispatcher *Dispatcher, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-dispatch-pipe"
	
	// Test data
	testData := map[string]interface{}{
		"message": "Hello, Dispatcher!",
		"time":    time.Now().Unix(),
	}
	
	// Wait group for synchronization
	done := make(chan struct{})
	
	// Subscribe to the pipe through the router
	err := router.Subscribe(pipeName, func(data []byte) error {
		var decoded interface{}
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		assert.Equal(t, testData["message"], decoded.(map[string]interface{})["message"], "Received message does not match sent message")
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

// testConcurrentDispatching tests concurrent dispatching of messages.
func testConcurrentDispatching(t *testing.T, dispatcher *Dispatcher, router core.HybridPipe) {
	// Number of concurrent operations
	numOperations := 10
	
	// Wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(numOperations)
	
	// Create a channel to collect errors
	errChan := make(chan error, numOperations)
	
	// Run concurrent operations
	for i := 0; i < numOperations; i++ {
		go func(index int) {
			defer wg.Done()
			
			// Test pipe name
			pipeName := "test-concurrent-dispatch-pipe-" + string(rune('A'+index))
			
			// Test data
			testData := map[string]interface{}{
				"message": "Hello, Concurrent Dispatcher!",
				"index":   index,
				"time":    time.Now().Unix(),
			}
			
			// Wait group for synchronization
			done := make(chan struct{})
			
			// Subscribe to the pipe through the router
			err := router.Subscribe(pipeName, func(data []byte) error {
				var decoded interface{}
				if err := core.Decode(data, &decoded); err != nil {
					errChan <- err
					return err
				}
				close(done)
				return nil
			})
			if err != nil {
				errChan <- err
				return
			}
			
			// Dispatch a message through the dispatcher
			err = dispatcher.Dispatch(pipeName, testData)
			if err != nil {
				errChan <- err
				return
			}
			
			// Wait for the message to be received
			select {
			case <-done:
				// Message received
			case <-time.After(5 * time.Second):
				errChan <- err
				return
			}
			
			// Unsubscribe from the pipe
			err = router.Unsubscribe(pipeName)
			if err != nil {
				errChan <- err
				return
			}
		}(i)
	}
	
	// Wait for all operations to complete
	wg.Wait()
	close(errChan)
	
	// Check for errors
	for err := range errChan {
		if err != nil {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	}
}

// testContextAwareDispatching tests context-aware dispatching of messages.
func testContextAwareDispatching(t *testing.T, dispatcher *Dispatcher, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-ctx-dispatch-pipe"
	
	// Test data
	testData := map[string]interface{}{
		"message": "Hello, Context-Aware Dispatcher!",
		"time":    time.Now().Unix(),
	}
	
	// Wait group for synchronization
	done := make(chan struct{})
	
	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Subscribe to the pipe through the router
	if ctxRouter, ok := router.(core.ContextAwareHybridPipe); ok {
		err := ctxRouter.SubscribeWithContext(ctx, pipeName, func(data []byte) error {
			var decoded interface{}
			if err := core.Decode(data, &decoded); err != nil {
				t.Errorf("Failed to decode message: %v", err)
				return err
			}
			assert.Equal(t, testData["message"], decoded.(map[string]interface{})["message"], "Received message does not match sent message")
			close(done)
			return nil
		})
		require.NoError(t, err, "Failed to subscribe to pipe with context")
	} else {
		err := router.Subscribe(pipeName, func(data []byte) error {
			var decoded interface{}
			if err := core.Decode(data, &decoded); err != nil {
				t.Errorf("Failed to decode message: %v", err)
				return err
			}
			assert.Equal(t, testData["message"], decoded.(map[string]interface{})["message"], "Received message does not match sent message")
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
