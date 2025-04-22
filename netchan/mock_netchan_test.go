package netchan

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockNetChan is a mock implementation of NetChan for testing.
type MockNetChan[T any] struct {
	// sendChan is used to send values
	sendChan chan T
	// recvChan is used to receive values
	recvChan chan T
	// closed indicates if the channel is closed
	closed bool
	// mutex protects concurrent access to closed
	mutex sync.RWMutex
	// errorMode can be set to simulate errors
	errorMode bool
	// delay simulates network latency
	delay time.Duration
	// name identifies this mock instance
	name string
	// sendCount tracks the number of values sent
	sendCount int
	// recvCount tracks the number of values received
	recvCount int
}

// NewMockNetChan creates a new MockNetChan with the given name and buffer size.
func NewMockNetChan[T any](name string, bufferSize int) *MockNetChan[T] {
	return &MockNetChan[T]{
		sendChan:  make(chan T, bufferSize),
		recvChan:  make(chan T, bufferSize),
		closed:    false,
		errorMode: false,
		delay:     0,
		name:      name,
		sendCount: 0,
		recvCount: 0,
	}
}

// SetErrorMode enables or disables error simulation.
func (m *MockNetChan[T]) SetErrorMode(enable bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.errorMode = enable
}

// SetDelay sets the simulated network latency.
func (m *MockNetChan[T]) SetDelay(delay time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.delay = delay
}

// GetSendCount returns the number of values sent.
func (m *MockNetChan[T]) GetSendCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.sendCount
}

// GetRecvCount returns the number of values received.
func (m *MockNetChan[T]) GetRecvCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.recvCount
}

// Send sends a value to the channel.
func (m *MockNetChan[T]) Send(value T) error {
	// Simulate network delay
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	m.mutex.RLock()
	if m.closed {
		m.mutex.RUnlock()
		return fmt.Errorf("channel is closed")
	}

	// Simulate error if in error mode
	if m.errorMode {
		m.mutex.RUnlock()
		return fmt.Errorf("mock send error for %s", m.name)
	}
	m.mutex.RUnlock()

	// Send the value
	m.sendChan <- value

	// Update send count
	m.mutex.Lock()
	m.sendCount++
	sendCount := m.sendCount
	m.mutex.Unlock()

	// Forward the value to the receive channel
	go func() {
		m.recvChan <- value
	}()

	return nil
}

// TrySend attempts to send a value to the channel without blocking.
func (m *MockNetChan[T]) TrySend(value T) (bool, error) {
	// Simulate network delay
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	m.mutex.RLock()
	if m.closed {
		m.mutex.RUnlock()
		return false, fmt.Errorf("channel is closed")
	}

	// Simulate error if in error mode
	if m.errorMode {
		m.mutex.RUnlock()
		return false, fmt.Errorf("mock send error for %s", m.name)
	}
	m.mutex.RUnlock()

	// Try to send the value
	select {
	case m.sendChan <- value:
		// Update send count
		m.mutex.Lock()
		m.sendCount++
		m.mutex.Unlock()

		// Forward the value to the receive channel
		go func() {
			m.recvChan <- value
		}()

		return true, nil
	default:
		return false, nil
	}
}

// Recv receives a value from the channel.
func (m *MockNetChan[T]) Recv() (T, error) {
	// Simulate network delay
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	m.mutex.RLock()
	if m.closed {
		m.mutex.RUnlock()
		var zero T
		return zero, fmt.Errorf("channel is closed")
	}

	// Simulate error if in error mode
	if m.errorMode {
		m.mutex.RUnlock()
		var zero T
		return zero, fmt.Errorf("mock receive error for %s", m.name)
	}
	m.mutex.RUnlock()

	// Receive the value
	value, ok := <-m.recvChan
	if !ok {
		var zero T
		return zero, fmt.Errorf("channel is closed")
	}

	// Update receive count
	m.mutex.Lock()
	m.recvCount++
	m.mutex.Unlock()

	return value, nil
}

// TryRecv attempts to receive a value from the channel without blocking.
func (m *MockNetChan[T]) TryRecv() (T, bool, error) {
	// Simulate network delay
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	m.mutex.RLock()
	if m.closed {
		m.mutex.RUnlock()
		var zero T
		return zero, false, fmt.Errorf("channel is closed")
	}

	// Simulate error if in error mode
	if m.errorMode {
		m.mutex.RUnlock()
		var zero T
		return zero, false, fmt.Errorf("mock receive error for %s", m.name)
	}
	m.mutex.RUnlock()

	// Try to receive the value
	select {
	case value, ok := <-m.recvChan:
		if !ok {
			var zero T
			return zero, false, fmt.Errorf("channel is closed")
		}

		// Update receive count
		m.mutex.Lock()
		m.recvCount++
		m.mutex.Unlock()

		return value, true, nil
	default:
		var zero T
		return zero, false, nil
	}
}

// Close closes the channel.
func (m *MockNetChan[T]) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return nil
	}

	// Simulate error if in error mode
	if m.errorMode {
		return fmt.Errorf("mock close error for %s", m.name)
	}

	m.closed = true
	close(m.sendChan)
	close(m.recvChan)

	return nil
}

// TestMockNetChan tests the MockNetChan implementation.
func TestMockNetChan(t *testing.T) {
	// Create a mock NetChan
	mock := NewMockNetChan[string]("test", 10)

	// Test basic send and receive
	t.Run("BasicSendAndReceive", func(t *testing.T) {
		// Send a value
		err := mock.Send("Hello, NetChan!")
		if err != nil {
			t.Fatalf("Failed to send: %v", err)
		}

		// Receive the value
		value, err := mock.Recv()
		if err != nil {
			t.Fatalf("Failed to receive: %v", err)
		}

		// Verify the value
		if value != "Hello, NetChan!" {
			t.Errorf("Received value doesn't match sent value: got %q, want %q", value, "Hello, NetChan!")
		}

		// Verify the counts
		if mock.GetSendCount() != 1 {
			t.Errorf("Expected send count to be 1, got %d", mock.GetSendCount())
		}
		if mock.GetRecvCount() != 1 {
			t.Errorf("Expected receive count to be 1, got %d", mock.GetRecvCount())
		}
	})

	// Test TrySend and TryRecv
	t.Run("TrySendAndTryRecv", func(t *testing.T) {
		// Try to send a value
		sent, err := mock.TrySend("Try Send")
		if err != nil {
			t.Fatalf("Failed to try send: %v", err)
		}
		if !sent {
			t.Error("Expected TrySend to succeed, but it failed")
		}

		// Try to receive a value
		value, received, err := mock.TryRecv()
		if err != nil {
			t.Fatalf("Failed to try receive: %v", err)
		}
		if !received {
			t.Error("Expected TryRecv to succeed, but it failed")
		}
		if value != "Try Send" {
			t.Errorf("Received value doesn't match sent value: got %q, want %q", value, "Try Send")
		}
	})

	// Test error simulation
	t.Run("ErrorSimulation", func(t *testing.T) {
		// Enable error mode
		mock.SetErrorMode(true)

		// Send should fail
		err := mock.Send("This should fail")
		if err == nil {
			t.Error("Expected error when sending in error mode, got nil")
		}

		// Receive should fail
		_, err = mock.Recv()
		if err == nil {
			t.Error("Expected error when receiving in error mode, got nil")
		}

		// TrySend should fail
		_, err = mock.TrySend("This should fail")
		if err == nil {
			t.Error("Expected error when try sending in error mode, got nil")
		}

		// TryRecv should fail
		_, _, err = mock.TryRecv()
		if err == nil {
			t.Error("Expected error when try receiving in error mode, got nil")
		}

		// Disable error mode
		mock.SetErrorMode(false)
	})

	// Test delay simulation
	t.Run("DelaySimulation", func(t *testing.T) {
		// Set delay
		mock.SetDelay(100 * time.Millisecond)

		// Send with delay
		start := time.Now()
		err := mock.Send("Delayed message")
		if err != nil {
			t.Fatalf("Failed to send with delay: %v", err)
		}
		elapsed := time.Since(start)
		if elapsed < 100*time.Millisecond {
			t.Errorf("Expected delay of at least 100ms for send, got %v", elapsed)
		}

		// Receive with delay
		start = time.Now()
		_, err = mock.Recv()
		if err != nil {
			t.Fatalf("Failed to receive with delay: %v", err)
		}
		elapsed = time.Since(start)
		if elapsed < 100*time.Millisecond {
			t.Errorf("Expected delay of at least 100ms for receive, got %v", elapsed)
		}

		// Reset delay
		mock.SetDelay(0)
	})

	// Test closing
	t.Run("Closing", func(t *testing.T) {
		// Create a new mock for this test
		closeMock := NewMockNetChan[int]("close-test", 10)

		// Close the channel
		err := closeMock.Close()
		if err != nil {
			t.Fatalf("Failed to close: %v", err)
		}

		// Send should fail
		err = closeMock.Send(42)
		if err == nil {
			t.Error("Expected error when sending to closed channel, got nil")
		}

		// Receive should fail
		_, err = closeMock.Recv()
		if err == nil {
			t.Error("Expected error when receiving from closed channel, got nil")
		}
	})

	// Test concurrent operations
	t.Run("ConcurrentOperations", func(t *testing.T) {
		// Create a new mock for this test
		concurrentMock := NewMockNetChan[int]("concurrent-test", 100)

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
					err := concurrentMock.Send(value)
					if err != nil {
						t.Errorf("Failed to send in goroutine %d: %v", id, err)
					}
				}
			}(i)
		}

		// Start goroutines to receive messages
		receivedValues := make([]int, 0, numGoroutines*numMessages)
		var receiveMutex sync.Mutex

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numMessages; j++ {
					value, err := concurrentMock.Recv()
					if err != nil {
						t.Errorf("Failed to receive in goroutine %d: %v", id, err)
						continue
					}

					receiveMutex.Lock()
					receivedValues = append(receivedValues, value)
					receiveMutex.Unlock()
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// Verify the counts
		if concurrentMock.GetSendCount() != numGoroutines*numMessages {
			t.Errorf("Expected send count to be %d, got %d", numGoroutines*numMessages, concurrentMock.GetSendCount())
		}
		if concurrentMock.GetRecvCount() != numGoroutines*numMessages {
			t.Errorf("Expected receive count to be %d, got %d", numGoroutines*numMessages, concurrentMock.GetRecvCount())
		}

		// Verify all values were received
		receiveMutex.Lock()
		if len(receivedValues) != numGoroutines*numMessages {
			t.Errorf("Expected to receive %d values, got %d", numGoroutines*numMessages, len(receivedValues))
		}
		receiveMutex.Unlock()
	})
}
