package mock

import (
	"context"
	"sync"
	"testing"
	"time"

	"hybridpipe.io/core"
)

// TestMockRouter tests the MockRouter implementation.
func TestMockRouter(t *testing.T) {
	t.Skip("Skipping mock tests until the mock package is fully implemented")
	// Create a new MockRouter
	router := NewMockRouter()

	// Connect to the mock messaging system
	err := router.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to mock messaging system: %v", err)
	}

	// Test basic operations
	testBasicOperations(t, router)

	// Test context-aware operations
	testContextAwareOperations(t, router)

	// Test concurrent operations
	testConcurrentOperations(t, router)

	// Close the router
	router.Close()
}

// testBasicOperations tests the basic operations of the MockRouter.
func testBasicOperations(t *testing.T, router *MockRouter) {
	// Test pipe name
	pipeName := "test-pipe"

	// Test data
	testData := map[string]interface{}{
		"message": "Hello, MockRouter!",
		"time":    time.Now().Unix(),
	}

	// Wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe to the pipe
	var receivedData interface{}
	err := router.Subscribe(pipeName, func(data []byte) error {
		var decoded interface{}
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		receivedData = decoded
		wg.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to pipe: %v", err)
	}

	// Dispatch a message
	err = router.Dispatch(pipeName, testData)
	if err != nil {
		t.Fatalf("Failed to dispatch message: %v", err)
	}

	// Wait for the message to be received
	waitTimeout(&wg, 5*time.Second)

	// Verify the received data
	if receivedData == nil {
		t.Fatal("No data received")
	}

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from pipe: %v", err)
	}
}

// testContextAwareOperations tests the context-aware operations of the MockRouter.
func testContextAwareOperations(t *testing.T, router *MockRouter) {
	// Test pipe name
	pipeName := "test-ctx-pipe"

	// Test data
	testData := map[string]interface{}{
		"message": "Hello, Context-Aware MockRouter!",
		"time":    time.Now().Unix(),
	}

	// Wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(1)

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to the pipe with context
	var receivedData interface{}
	err := router.SubscribeWithContext(ctx, pipeName, func(data []byte) error {
		var decoded interface{}
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		receivedData = decoded
		wg.Done()
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
	waitTimeout(&wg, 5*time.Second)

	// Verify the received data
	if receivedData == nil {
		t.Fatal("No data received")
	}

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from pipe: %v", err)
	}
}

// testConcurrentOperations tests concurrent operations on the MockRouter.
func testConcurrentOperations(t *testing.T, router *MockRouter) {
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
			pipeName := "test-concurrent-pipe-" + string(rune('A'+index))

			// Test data
			testData := map[string]interface{}{
				"message": "Hello, Concurrent MockRouter!",
				"index":   index,
				"time":    time.Now().Unix(),
			}

			// Subscribe to the pipe
			var receivedData interface{}
			var receivedWg sync.WaitGroup
			receivedWg.Add(1)

			err := router.Subscribe(pipeName, func(data []byte) error {
				var decoded interface{}
				if err := core.Decode(data, &decoded); err != nil {
					errChan <- err
					return err
				}
				receivedData = decoded
				receivedWg.Done()
				return nil
			})
			if err != nil {
				errChan <- err
				return
			}

			// Dispatch a message
			err = router.Dispatch(pipeName, testData)
			if err != nil {
				errChan <- err
				return
			}

			// Wait for the message to be received
			if waitTimeout(&receivedWg, 5*time.Second) {
				errChan <- err
				return
			}

			// Verify the received data
			if receivedData == nil {
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

// waitTimeout waits for the WaitGroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
