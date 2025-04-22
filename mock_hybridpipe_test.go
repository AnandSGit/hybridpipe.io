package hybridpipe

import (
	"fmt"
	"sync"
)

// MockHybridPipe is a mock implementation of the HybridPipe interface for testing.
type MockHybridPipe struct {
	// connected indicates if the pipe is connected
	connected bool
	// subscribers is a map of topic to a list of handlers
	subscribers map[string][]Process
	// messages is a map of topic to a list of messages
	messages map[string][]interface{}
	// mutex protects concurrent access
	mutex sync.RWMutex
	// errorMode can be set to simulate errors
	errorMode bool
	// delay simulates network latency
	delay int
}

// NewMockHybridPipe creates a new MockHybridPipe.
func NewMockHybridPipe() *MockHybridPipe {
	return &MockHybridPipe{
		connected:   false,
		subscribers: make(map[string][]Process),
		messages:    make(map[string][]interface{}),
		errorMode:   false,
		delay:       0,
	}
}

// Connect establishes a connection to the messaging system.
func (m *MockHybridPipe) Connect() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.errorMode {
		return fmt.Errorf("mock connect error")
	}

	m.connected = true
	return nil
}

// Dispatch sends a message to the specified pipe.
func (m *MockHybridPipe) Dispatch(pipe string, data interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.connected {
		return fmt.Errorf("not connected")
	}

	if m.errorMode {
		return fmt.Errorf("mock dispatch error")
	}

	// Store the message
	if _, ok := m.messages[pipe]; !ok {
		m.messages[pipe] = make([]interface{}, 0)
	}
	m.messages[pipe] = append(m.messages[pipe], data)

	// Notify subscribers
	if subscribers, ok := m.subscribers[pipe]; ok {
		for _, subscriber := range subscribers {
			// Make a copy of the data to avoid race conditions
			dataCopy := data
			// Call the subscriber in a goroutine to avoid blocking
			go func(fn Process, d interface{}) {
				// Recover from panics in the subscriber
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("Recovered from panic in subscriber: %v\n", r)
					}
				}()
				fn(d)
			}(subscriber, dataCopy)
		}
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (m *MockHybridPipe) Accept(pipe string, fn Process) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.connected {
		return fmt.Errorf("not connected")
	}

	if m.errorMode {
		return fmt.Errorf("mock accept error")
	}

	// Add the subscriber
	if _, ok := m.subscribers[pipe]; !ok {
		m.subscribers[pipe] = make([]Process, 0)
	}
	m.subscribers[pipe] = append(m.subscribers[pipe], fn)

	return nil
}

// Remove unsubscribes from the specified pipe.
func (m *MockHybridPipe) Remove(pipe string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.connected {
		return fmt.Errorf("not connected")
	}

	if m.errorMode {
		return fmt.Errorf("mock remove error")
	}

	// Remove all subscribers for the pipe
	delete(m.subscribers, pipe)

	return nil
}

// Close terminates the connection to the messaging system.
func (m *MockHybridPipe) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.connected = false
	m.subscribers = make(map[string][]Process)
}

// SetErrorMode enables or disables error simulation.
func (m *MockHybridPipe) SetErrorMode(enable bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.errorMode = enable
}

// SetDelay sets the simulated network latency.
func (m *MockHybridPipe) SetDelay(delay int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.delay = delay
}

// GetSubscribers returns the subscribers for a pipe.
func (m *MockHybridPipe) GetSubscribers(pipe string) []Process {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if subscribers, ok := m.subscribers[pipe]; ok {
		// Make a copy to avoid race conditions
		result := make([]Process, len(subscribers))
		copy(result, subscribers)
		return result
	}

	return nil
}

// GetMessages returns the messages for a pipe.
func (m *MockHybridPipe) GetMessages(pipe string) []interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if messages, ok := m.messages[pipe]; ok {
		// Make a copy to avoid race conditions
		result := make([]interface{}, len(messages))
		copy(result, messages)
		return result
	}

	return nil
}

// GetSubscriptionCount returns the total number of subscriptions.
func (m *MockHybridPipe) GetSubscriptionCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	count := 0
	for _, subscribers := range m.subscribers {
		count += len(subscribers)
	}
	return count
}

// IsConnected returns whether the pipe is connected.
func (m *MockHybridPipe) IsConnected() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.connected
}

// Reset resets the mock to its initial state.
func (m *MockHybridPipe) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.connected = false
	m.subscribers = make(map[string][]Process)
	m.messages = make(map[string][]interface{})
	m.errorMode = false
	m.delay = 0
}
