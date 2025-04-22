package hybridpipe

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestAccept tests the Accept function with different scenarios.
func TestAccept(t *testing.T) {
	// Create a mock router
	router, err := DeployRouter(MOCK)
	if err != nil {
		t.Fatalf("Failed to create mock router: %v", err)
	}
	defer router.Close()

	// Test basic subscription
	t.Run("BasicSubscription", func(t *testing.T) {
		// Subscribe to a topic
		received := make(chan string, 1)
		err := router.Accept("test.basic", func(data any) {
			if str, ok := data.(string); ok {
				received <- str
			}
		})
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Dispatch a message
		err = router.Dispatch("test.basic", "Hello, Accept!")
		if err != nil {
			t.Fatalf("Failed to dispatch: %v", err)
		}

		// Wait for the message
		select {
		case msg := <-received:
			if msg != "Hello, Accept!" {
				t.Errorf("Received message = %s, want %s", msg, "Hello, Accept!")
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for message")
		}
	})

	// Test multiple subscriptions to the same topic
	t.Run("MultipleSubscriptions", func(t *testing.T) {
		// Create channels to receive messages
		received1 := make(chan string, 1)
		received2 := make(chan string, 1)

		// Subscribe first handler
		err := router.Accept("test.multiple", func(data any) {
			if str, ok := data.(string); ok {
				received1 <- str
			}
		})
		if err != nil {
			t.Fatalf("Failed to subscribe first handler: %v", err)
		}

		// Subscribe second handler
		err = router.Accept("test.multiple", func(data any) {
			if str, ok := data.(string); ok {
				received2 <- str
			}
		})
		if err != nil {
			t.Fatalf("Failed to subscribe second handler: %v", err)
		}

		// Dispatch a message
		err = router.Dispatch("test.multiple", "Hello, Multiple!")
		if err != nil {
			t.Fatalf("Failed to dispatch: %v", err)
		}

		// Wait for both handlers to receive the message
		select {
		case msg := <-received1:
			if msg != "Hello, Multiple!" {
				t.Errorf("Handler 1: received message = %s, want %s", msg, "Hello, Multiple!")
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for handler 1")
		}

		select {
		case msg := <-received2:
			if msg != "Hello, Multiple!" {
				t.Errorf("Handler 2: received message = %s, want %s", msg, "Hello, Multiple!")
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for handler 2")
		}
	})

	// Test removing a subscription
	t.Run("RemoveSubscription", func(t *testing.T) {
		// Subscribe to a topic
		called := false
		err := router.Accept("test.remove", func(data any) {
			called = true
		})
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Remove the subscription
		err = router.Remove("test.remove")
		if err != nil {
			t.Fatalf("Failed to remove subscription: %v", err)
		}

		// Dispatch a message
		err = router.Dispatch("test.remove", "This should not be received")
		if err != nil {
			t.Fatalf("Failed to dispatch: %v", err)
		}

		// Wait a bit to ensure the message is not received
		time.Sleep(100 * time.Millisecond)

		// Check that the handler was not called
		if called {
			t.Error("Handler was called after subscription was removed")
		}
	})

	// Test subscribing to a non-existent topic
	t.Run("NonExistentTopic", func(t *testing.T) {
		// Subscribe to a topic that doesn't exist yet
		received := make(chan string, 1)
		err := router.Accept("test.nonexistent", func(data any) {
			if str, ok := data.(string); ok {
				received <- str
			}
		})
		if err != nil {
			t.Fatalf("Failed to subscribe to non-existent topic: %v", err)
		}

		// Dispatch a message to the topic
		err = router.Dispatch("test.nonexistent", "Hello, NonExistent!")
		if err != nil {
			t.Fatalf("Failed to dispatch to non-existent topic: %v", err)
		}

		// Wait for the message
		select {
		case msg := <-received:
			if msg != "Hello, NonExistent!" {
				t.Errorf("Received message = %s, want %s", msg, "Hello, NonExistent!")
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for message")
		}
	})

	// Test error handling in the handler
	t.Run("HandlerError", func(t *testing.T) {
		// Subscribe to a topic with a handler that panics
		err := router.Accept("test.panic", func(data any) {
			panic("This is a test panic")
		})
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Dispatch a message
		err = router.Dispatch("test.panic", "This should cause a panic")
		if err != nil {
			t.Fatalf("Failed to dispatch: %v", err)
		}

		// Wait a bit to ensure the panic is handled
		time.Sleep(100 * time.Millisecond)

		// If we got here, the panic was handled and didn't crash the test
	})

	// Test concurrent subscriptions
	t.Run("ConcurrentSubscriptions", func(t *testing.T) {
		// Number of goroutines
		const numGoroutines = 10

		// Create a wait group to wait for all goroutines
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Start goroutines
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				// Create a unique topic for this goroutine
				topic := fmt.Sprintf("test.concurrent.%d", id)

				// Subscribe to the topic
				err := router.Accept(topic, func(data any) {
					// Do nothing
				})
				if err != nil {
					t.Errorf("Goroutine %d: failed to subscribe: %v", id, err)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// Check that all subscriptions were created
		mockRouter, ok := router.(*MockPacket)
		if !ok {
			t.Fatal("Failed to cast router to MockPacket")
		}

		// Verify that we have the expected number of subscriptions
		if count := mockRouter.GetSubscriptionCount(); count < numGoroutines {
			t.Errorf("Expected at least %d subscriptions, got %d", numGoroutines, count)
		}
	})
}
