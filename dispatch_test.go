package hybridpipe

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestDispatch tests the Dispatch function with different scenarios.
func TestDispatch(t *testing.T) {
	// Create a mock router
	router, err := DeployRouter(MOCK)
	if err != nil {
		t.Fatalf("Failed to create mock router: %v", err)
	}
	defer router.Close()

	// Test basic dispatch
	t.Run("BasicDispatch", func(t *testing.T) {
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
		err = router.Dispatch("test.basic", "Hello, Dispatch!")
		if err != nil {
			t.Fatalf("Failed to dispatch: %v", err)
		}

		// Wait for the message
		select {
		case msg := <-received:
			if msg != "Hello, Dispatch!" {
				t.Errorf("Received message = %s, want %s", msg, "Hello, Dispatch!")
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for message")
		}
	})

	// Test dispatching different data types
	t.Run("DifferentDataTypes", func(t *testing.T) {
		// Register custom types
		type Person struct {
			Name string
			Age  int
		}
		Enable(Person{})

		// Subscribe to topics for different data types
		stringReceived := make(chan string, 1)
		err := router.Accept("test.string", func(data any) {
			if str, ok := data.(string); ok {
				stringReceived <- str
			}
		})
		if err != nil {
			t.Fatalf("Failed to subscribe to string topic: %v", err)
		}

		intReceived := make(chan int, 1)
		err = router.Accept("test.int", func(data any) {
			if num, ok := data.(int); ok {
				intReceived <- num
			}
		})
		if err != nil {
			t.Fatalf("Failed to subscribe to int topic: %v", err)
		}

		boolReceived := make(chan bool, 1)
		err = router.Accept("test.bool", func(data any) {
			if b, ok := data.(bool); ok {
				boolReceived <- b
			}
		})
		if err != nil {
			t.Fatalf("Failed to subscribe to bool topic: %v", err)
		}

		structReceived := make(chan Person, 1)
		err = router.Accept("test.struct", func(data any) {
			if p, ok := data.(Person); ok {
				structReceived <- p
			}
		})
		if err != nil {
			t.Fatalf("Failed to subscribe to struct topic: %v", err)
		}

		// Dispatch messages of different types
		err = router.Dispatch("test.string", "Hello, String!")
		if err != nil {
			t.Fatalf("Failed to dispatch string: %v", err)
		}

		err = router.Dispatch("test.int", 42)
		if err != nil {
			t.Fatalf("Failed to dispatch int: %v", err)
		}

		err = router.Dispatch("test.bool", true)
		if err != nil {
			t.Fatalf("Failed to dispatch bool: %v", err)
		}

		person := Person{Name: "Alice", Age: 30}
		err = router.Dispatch("test.struct", person)
		if err != nil {
			t.Fatalf("Failed to dispatch struct: %v", err)
		}

		// Wait for the messages
		select {
		case msg := <-stringReceived:
			if msg != "Hello, String!" {
				t.Errorf("Received string = %s, want %s", msg, "Hello, String!")
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for string message")
		}

		select {
		case num := <-intReceived:
			if num != 42 {
				t.Errorf("Received int = %d, want %d", num, 42)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for int message")
		}

		select {
		case b := <-boolReceived:
			if !b {
				t.Errorf("Received bool = %v, want %v", b, true)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for bool message")
		}

		select {
		case p := <-structReceived:
			if p.Name != "Alice" || p.Age != 30 {
				t.Errorf("Received struct = %+v, want %+v", p, person)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for struct message")
		}
	})

	// Test dispatching to a topic with no subscribers
	t.Run("NoSubscribers", func(t *testing.T) {
		// Dispatch a message to a topic with no subscribers
		err := router.Dispatch("test.nosubscribers", "Hello, Nobody!")
		if err != nil {
			t.Fatalf("Failed to dispatch to topic with no subscribers: %v", err)
		}

		// If we got here, the dispatch succeeded
	})

	// Test dispatching nil
	t.Run("DispatchNil", func(t *testing.T) {
		// Subscribe to a topic
		received := make(chan bool, 1)
		err := router.Accept("test.nil", func(data any) {
			received <- data == nil
		})
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Dispatch nil
		err = router.Dispatch("test.nil", nil)
		if err != nil {
			t.Fatalf("Failed to dispatch nil: %v", err)
		}

		// Wait for the message
		select {
		case isNil := <-received:
			if !isNil {
				t.Error("Received data is not nil")
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for nil message")
		}
	})

	// Test concurrent dispatches
	t.Run("ConcurrentDispatches", func(t *testing.T) {
		// Number of goroutines
		const numGoroutines = 10
		// Number of messages per goroutine
		const numMessages = 100

		// Create a wait group to wait for all goroutines
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Create a mutex to protect the received map
		var receivedMutex sync.Mutex
		// Create a map to track received messages
		received := make(map[string]int)

		// Subscribe to the topic
		err := router.Accept("test.concurrent", func(data any) {
			if str, ok := data.(string); ok {
				receivedMutex.Lock()
				received[str]++
				receivedMutex.Unlock()
			}
		})
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Start goroutines
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				// Dispatch messages
				for j := 0; j < numMessages; j++ {
					message := fmt.Sprintf("Message %d-%d", id, j)
					err := router.Dispatch("test.concurrent", message)
					if err != nil {
						t.Errorf("Failed to dispatch message %s: %v", message, err)
					}
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// Wait a bit for all messages to be processed
		time.Sleep(100 * time.Millisecond)

		// Check that all messages were received
		receivedMutex.Lock()
		defer receivedMutex.Unlock()

		if len(received) != numGoroutines*numMessages {
			t.Errorf("Expected %d unique messages, got %d", numGoroutines*numMessages, len(received))
		}

		// Check that each message was received exactly once
		for i := 0; i < numGoroutines; i++ {
			for j := 0; j < numMessages; j++ {
				message := fmt.Sprintf("Message %d-%d", i, j)
				count, ok := received[message]
				if !ok {
					t.Errorf("Message %s was not received", message)
				} else if count != 1 {
					t.Errorf("Message %s was received %d times, expected 1", message, count)
				}
			}
		}
	})

	// Test dispatching large messages
	t.Run("LargeMessages", func(t *testing.T) {
		// Create a large message
		largeMessage := make([]byte, 1000000) // 1MB
		for i := range largeMessage {
			largeMessage[i] = byte(i % 256)
		}

		// Subscribe to the topic
		received := make(chan bool, 1)
		err := router.Accept("test.large", func(data any) {
			if bytes, ok := data.([]byte); ok {
				// Check that the received message matches the sent message
				if len(bytes) != len(largeMessage) {
					t.Errorf("Received message length = %d, want %d", len(bytes), len(largeMessage))
					received <- false
					return
				}

				// Check a few sample bytes
				for i := 0; i < len(bytes); i += 100000 {
					if bytes[i] != largeMessage[i] {
						t.Errorf("Byte at position %d = %d, want %d", i, bytes[i], largeMessage[i])
						received <- false
						return
					}
				}

				received <- true
			}
		})
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Dispatch the large message
		err = router.Dispatch("test.large", largeMessage)
		if err != nil {
			t.Fatalf("Failed to dispatch large message: %v", err)
		}

		// Wait for the message
		select {
		case success := <-received:
			if !success {
				t.Error("Large message verification failed")
			}
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for large message")
		}
	})
}
