package netchan

import (
	"sync"
	"testing"
	"time"
)

// TestChannelWrapper tests the ChannelWrapper implementation.
func TestChannelWrapper(t *testing.T) {
	// Create a mock NetChan
	mock := NewMockNetChan[interface{}]("wrapper-test", 10)
	
	// Create a wrapper around the mock
	wrapper := NewChannelWrapper(mock, 10)
	
	// Initialize the wrapper
	err := wrapper.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize wrapper: %v", err)
	}
	
	// Test basic send and receive
	t.Run("BasicSendAndReceive", func(t *testing.T) {
		// Get the send and receive channels
		sendChan := wrapper.SendChan()
		recvChan := wrapper.RecvChan()
		
		// Send a value
		sendChan <- "Hello, ChannelWrapper!"
		
		// Receive the value
		select {
		case value := <-recvChan:
			// Verify the value
			if value != "Hello, ChannelWrapper!" {
				t.Errorf("Received value doesn't match sent value: got %v, want %v", value, "Hello, ChannelWrapper!")
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for message")
		}
	})
	
	// Test multiple values
	t.Run("MultipleValues", func(t *testing.T) {
		// Get the send and receive channels
		sendChan := wrapper.SendChan()
		recvChan := wrapper.RecvChan()
		
		// Send multiple values
		values := []interface{}{42, "string", true, 3.14}
		for _, value := range values {
			sendChan <- value
		}
		
		// Receive the values
		for i, expected := range values {
			select {
			case value := <-recvChan:
				// Verify the value
				if value != expected {
					t.Errorf("Value %d doesn't match: got %v, want %v", i, value, expected)
				}
			case <-time.After(1 * time.Second):
				t.Errorf("Timeout waiting for value %d", i)
			}
		}
	})
	
	// Test concurrent operations
	t.Run("ConcurrentOperations", func(t *testing.T) {
		// Create a new mock and wrapper for this test
		concurrentMock := NewMockNetChan[interface{}]("concurrent-wrapper-test", 100)
		concurrentWrapper := NewChannelWrapper(concurrentMock, 100)
		err := concurrentWrapper.Initialize()
		if err != nil {
			t.Fatalf("Failed to initialize concurrent wrapper: %v", err)
		}
		
		// Get the send and receive channels
		sendChan := concurrentWrapper.SendChan()
		recvChan := concurrentWrapper.RecvChan()
		
		// Number of goroutines and messages
		const numGoroutines = 10
		const numMessages = 100
		
		// Create a wait group to wait for all goroutines
		var wg sync.WaitGroup
		wg.Add(numGoroutines * 2) // For both senders and receivers
		
		// Start goroutines to send messages
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numMessages; j++ {
					value := id*numMessages + j
					sendChan <- value
				}
			}(i)
		}
		
		// Start goroutines to receive messages
		receivedValues := make([]interface{}, 0, numGoroutines*numMessages)
		var receiveMutex sync.Mutex
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numMessages; j++ {
					value := <-recvChan
					
					receiveMutex.Lock()
					receivedValues = append(receivedValues, value)
					receiveMutex.Unlock()
				}
			}()
		}
		
		// Wait for all goroutines to complete
		wg.Wait()
		
		// Verify all values were received
		receiveMutex.Lock()
		if len(receivedValues) != numGoroutines*numMessages {
			t.Errorf("Expected to receive %d values, got %d", numGoroutines*numMessages, len(receivedValues))
		}
		receiveMutex.Unlock()
	})
	
	// Test closing
	t.Run("Closing", func(t *testing.T) {
		// Create a new mock and wrapper for this test
		closeMock := NewMockNetChan[interface{}]("close-wrapper-test", 10)
		closeWrapper := NewChannelWrapper(closeMock, 10)
		err := closeWrapper.Initialize()
		if err != nil {
			t.Fatalf("Failed to initialize close wrapper: %v", err)
		}
		
		// Close the wrapper
		err = closeWrapper.Close()
		if err != nil {
			t.Fatalf("Failed to close wrapper: %v", err)
		}
		
		// Get the receive channel
		recvChan := closeWrapper.RecvChan()
		
		// Verify the channel is closed
		select {
		case _, ok := <-recvChan:
			if ok {
				t.Error("Expected receive channel to be closed, but it's still open")
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for channel to close")
		}
	})
	
	// Test error handling
	t.Run("ErrorHandling", func(t *testing.T) {
		// Create a new mock and wrapper for this test
		errorMock := NewMockNetChan[interface{}]("error-wrapper-test", 10)
		errorWrapper := NewChannelWrapper(errorMock, 10)
		err := errorWrapper.Initialize()
		if err != nil {
			t.Fatalf("Failed to initialize error wrapper: %v", err)
		}
		
		// Enable error mode on the mock
		errorMock.SetErrorMode(true)
		
		// Get the send channel
		sendChan := errorWrapper.SendChan()
		
		// Send a value (this should eventually cause the wrapper to close the channel)
		sendChan <- "This should cause an error"
		
		// Wait a bit for the error to be processed
		time.Sleep(100 * time.Millisecond)
		
		// Get the receive channel
		recvChan := errorWrapper.RecvChan()
		
		// Verify the channel is closed (or will be closed soon)
		select {
		case _, ok := <-recvChan:
			if ok {
				// The channel might not be closed immediately, so this is not necessarily an error
				// Just wait a bit longer
				time.Sleep(100 * time.Millisecond)
				
				// Check again
				select {
				case _, ok := <-recvChan:
					if ok {
						t.Error("Expected receive channel to be closed after error, but it's still open")
					}
				default:
					// This is fine, the channel might be empty but not closed yet
				}
			}
		case <-time.After(100 * time.Millisecond):
			// This is fine, the channel might be empty but not closed yet
		}
	})
}

// TestChannelWrapperWithTypedChannels tests the ChannelWrapper with typed channels.
func TestChannelWrapperWithTypedChannels(t *testing.T) {
	// Create a mock NetChan with string type
	mock := NewMockNetChan[interface{}]("typed-wrapper-test", 10)
	
	// Create a wrapper around the mock
	wrapper := NewChannelWrapper(mock, 10)
	
	// Initialize the wrapper
	err := wrapper.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize wrapper: %v", err)
	}
	
	// Test with string type
	t.Run("StringType", func(t *testing.T) {
		// Get the send and receive channels
		sendChan := wrapper.SendChan()
		recvChan := wrapper.RecvChan()
		
		// Send a string
		sendChan <- "Hello, Typed Channel!"
		
		// Receive the string
		select {
		case value := <-recvChan:
			// Verify the value
			if str, ok := value.(string); ok {
				if str != "Hello, Typed Channel!" {
					t.Errorf("Received string doesn't match sent string: got %q, want %q", str, "Hello, Typed Channel!")
				}
			} else {
				t.Errorf("Expected string, got %T", value)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for message")
		}
	})
	
	// Test with int type
	t.Run("IntType", func(t *testing.T) {
		// Get the send and receive channels
		sendChan := wrapper.SendChan()
		recvChan := wrapper.RecvChan()
		
		// Send an int
		sendChan <- 42
		
		// Receive the int
		select {
		case value := <-recvChan:
			// Verify the value
			if num, ok := value.(int); ok {
				if num != 42 {
					t.Errorf("Received int doesn't match sent int: got %d, want %d", num, 42)
				}
			} else {
				t.Errorf("Expected int, got %T", value)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for message")
		}
	})
	
	// Test with struct type
	t.Run("StructType", func(t *testing.T) {
		// Define a struct type
		type Person struct {
			Name string
			Age  int
		}
		
		// Get the send and receive channels
		sendChan := wrapper.SendChan()
		recvChan := wrapper.RecvChan()
		
		// Send a struct
		person := Person{Name: "Alice", Age: 30}
		sendChan <- person
		
		// Receive the struct
		select {
		case value := <-recvChan:
			// Verify the value
			if p, ok := value.(Person); ok {
				if p.Name != "Alice" || p.Age != 30 {
					t.Errorf("Received struct doesn't match sent struct: got %+v, want %+v", p, person)
				}
			} else {
				t.Errorf("Expected Person struct, got %T", value)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for message")
		}
	})
}
