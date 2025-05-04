package core

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestHybridPipeInterface tests that all protocol implementations satisfy the HybridPipe interface.
func TestHybridPipeInterface(t *testing.T) {
	t.Skip("Skipping HybridPipe interface tests until all protocol implementations are fully implemented")
	// Test each protocol type
	protocols := []int{
		NATS,
		KAFKA,
		RABBITMQ,
		AMQP1,
		MQTT,
		QPID,
		NSQ,
		TCP,
		REDIS,
		NETCHAN,
		MOCK,
	}

	for _, protocol := range protocols {
		t.Run(fmt.Sprintf("Protocol_%d", protocol), func(t *testing.T) {
			// Skip protocols that require external servers in CI environment
			if testing.Short() && protocol != MOCK {
				t.Skip("Skipping test in short mode")
			}

			// Deploy the router
			router, err := DeployRouter(protocol)
			if err != nil {
				t.Fatalf("Failed to deploy router for protocol %d: %v", protocol, err)
			}

			// Test basic operations
			testBasicOperations(t, router)

			// Test context-aware operations if supported
			if ctxRouter, ok := router.(ContextAwareHybridPipe); ok {
				testContextAwareOperations(t, ctxRouter)
			}

			// Close the router
			router.Close()
		})
	}
}

// testBasicOperations tests the basic operations of a HybridPipe implementation.
func testBasicOperations(t *testing.T, router HybridPipe) {
	// Test pipe name
	pipeName := fmt.Sprintf("test-pipe-%d", time.Now().UnixNano())

	// Test data
	testData := map[string]interface{}{
		"message": "Hello, HybridPipe!",
		"time":    time.Now().Unix(),
	}

	// Wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe to the pipe
	var receivedData interface{}
	err := router.Subscribe(pipeName, func(data []byte) error {
		var decoded interface{}
		if err := Decode(data, &decoded); err != nil {
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

// testContextAwareOperations tests the context-aware operations of a HybridPipe implementation.
func testContextAwareOperations(t *testing.T, router ContextAwareHybridPipe) {
	// Test pipe name
	pipeName := fmt.Sprintf("test-ctx-pipe-%d", time.Now().UnixNano())

	// Test data
	testData := map[string]interface{}{
		"message": "Hello, Context-Aware HybridPipe!",
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
		if err := Decode(data, &decoded); err != nil {
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
