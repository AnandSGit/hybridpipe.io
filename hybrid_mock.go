package hybridpipe

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// MockPacket implements the HybridPipe interface for testing purposes.
// It simulates the behavior of a messaging system without requiring actual servers.
type MockPacket struct {
	// messages maps pipe names to their messages
	messages map[string][][]byte
	// subscribers maps pipe names to their subscriber functions
	subscribers map[string][]Process
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// connected indicates if the mock is "connected"
	connected bool
	// errorMode can be set to simulate connection or operation failures
	errorMode bool
	// delay simulates network latency in milliseconds
	delay time.Duration
	// name identifies this mock instance (useful for debugging)
	name string
	// messageCount tracks the number of messages sent
	messageCount int
	// subscriptionCount tracks the number of subscriptions
	subscriptionCount int
}

// NewMockPacket creates a new MockPacket with the given name.
func NewMockPacket(name string) *MockPacket {
	return &MockPacket{
		messages:    make(map[string][][]byte),
		subscribers: make(map[string][]Process),
		connected:   false,
		errorMode:   false,
		delay:       0,
		name:        name,
	}
}

// SetErrorMode enables or disables error simulation.
func (mp *MockPacket) SetErrorMode(enable bool) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	mp.errorMode = enable
}

// SetDelay sets the simulated network latency.
func (mp *MockPacket) SetDelay(delay time.Duration) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	mp.delay = delay
}

// GetMessageCount returns the number of messages sent.
func (mp *MockPacket) GetMessageCount() int {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()
	return mp.messageCount
}

// GetSubscriptionCount returns the number of subscriptions.
func (mp *MockPacket) GetSubscriptionCount() int {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()
	return mp.subscriptionCount
}

// GetMessages returns all messages for a pipe.
func (mp *MockPacket) GetMessages(pipe string) [][]byte {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()

	if messages, exists := mp.messages[pipe]; exists {
		// Return a copy to avoid race conditions
		result := make([][]byte, len(messages))
		copy(result, messages)
		return result
	}

	return nil
}

// ClearMessages removes all messages from a pipe.
func (mp *MockPacket) ClearMessages(pipe string) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	if _, exists := mp.messages[pipe]; exists {
		mp.messages[pipe] = [][]byte{}
	}
}

// Connect simulates connecting to a messaging system.
func (mp *MockPacket) Connect() error {
	// Simulate network delay
	if mp.delay > 0 {
		time.Sleep(mp.delay)
	}

	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	// Simulate connection error if in error mode
	if mp.errorMode {
		return fmt.Errorf("mock connection error for %s", mp.name)
	}

	mp.connected = true
	log.Printf("[MOCK:%s] Connected successfully", mp.name)
	return nil
}

// Dispatch simulates publishing a message to a pipe.
func (mp *MockPacket) Dispatch(pipe string, data interface{}) error {
	// Simulate network delay
	if mp.delay > 0 {
		time.Sleep(mp.delay)
	}

	mp.mutex.RLock()
	if !mp.connected {
		mp.mutex.RUnlock()
		return fmt.Errorf("not connected to mock messaging system %s", mp.name)
	}

	// Simulate dispatch error if in error mode
	if mp.errorMode {
		mp.mutex.RUnlock()
		return fmt.Errorf("mock dispatch error for %s", mp.name)
	}
	mp.mutex.RUnlock()

	// Encode the data
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	mp.mutex.Lock()
	// Store the message
	if _, exists := mp.messages[pipe]; !exists {
		mp.messages[pipe] = [][]byte{}
	}
	mp.messages[pipe] = append(mp.messages[pipe], bytes)
	mp.messageCount++
	messageCount := mp.messageCount
	mp.mutex.Unlock()

	log.Printf("[MOCK:%s] Dispatched message #%d to pipe %s", mp.name, messageCount, pipe)

	// Deliver the message to subscribers
	mp.deliverMessage(pipe, bytes)

	return nil
}

// deliverMessage delivers a message to all subscribers of a pipe.
func (mp *MockPacket) deliverMessage(pipe string, data []byte) {
	mp.mutex.RLock()
	subscribers := mp.subscribers[pipe]
	mp.mutex.RUnlock()

	// If there are no subscribers, return
	if len(subscribers) == 0 {
		return
	}

	// Decode the message for each subscriber
	for _, subscriber := range subscribers {
		var decodedData interface{}
		if err := Decode(data, &decodedData); err != nil {
			log.Printf("[MOCK:%s] Failed to decode message for pipe %s: %v", mp.name, pipe, err)
			continue
		}

		// Process the message in a goroutine to simulate asynchronous delivery
		go subscriber(decodedData)
	}
}

// Accept simulates subscribing to messages from a pipe.
func (mp *MockPacket) Accept(pipe string, fn Process) error {
	// Simulate network delay
	if mp.delay > 0 {
		time.Sleep(mp.delay)
	}

	mp.mutex.RLock()
	if !mp.connected {
		mp.mutex.RUnlock()
		return fmt.Errorf("not connected to mock messaging system %s", mp.name)
	}

	// Simulate accept error if in error mode
	if mp.errorMode {
		mp.mutex.RUnlock()
		return fmt.Errorf("mock accept error for %s", mp.name)
	}
	mp.mutex.RUnlock()

	mp.mutex.Lock()
	// Add the subscriber
	if _, exists := mp.subscribers[pipe]; !exists {
		mp.subscribers[pipe] = []Process{}
	}
	mp.subscribers[pipe] = append(mp.subscribers[pipe], fn)
	mp.subscriptionCount++
	subscriptionCount := mp.subscriptionCount
	mp.mutex.Unlock()

	log.Printf("[MOCK:%s] Subscribed to pipe %s (subscription #%d)", mp.name, pipe, subscriptionCount)

	// Deliver any existing messages to the new subscriber
	mp.mutex.RLock()
	messages := mp.messages[pipe]
	mp.mutex.RUnlock()

	for _, message := range messages {
		var decodedData interface{}
		if err := Decode(message, &decodedData); err != nil {
			log.Printf("[MOCK:%s] Failed to decode existing message for pipe %s: %v", mp.name, pipe, err)
			continue
		}

		// Process the message
		fn(decodedData)
	}

	return nil
}

// Remove simulates unsubscribing from a pipe.
func (mp *MockPacket) Remove(pipe string) error {
	// Simulate network delay
	if mp.delay > 0 {
		time.Sleep(mp.delay)
	}

	mp.mutex.RLock()
	if !mp.connected {
		mp.mutex.RUnlock()
		return fmt.Errorf("not connected to mock messaging system %s", mp.name)
	}

	// Simulate remove error if in error mode
	if mp.errorMode {
		mp.mutex.RUnlock()
		return fmt.Errorf("mock remove error for %s", mp.name)
	}
	mp.mutex.RUnlock()

	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	// Remove all subscribers for the pipe
	if _, exists := mp.subscribers[pipe]; exists {
		// Decrement subscription count by the number of subscribers for this pipe
		mp.subscriptionCount -= len(mp.subscribers[pipe])
		delete(mp.subscribers, pipe)
		log.Printf("[MOCK:%s] Unsubscribed from pipe %s", mp.name, pipe)
		return nil
	}

	return fmt.Errorf("pipe %s is not subscribed in mock %s", pipe, mp.name)
}

// Close simulates closing the connection.
func (mp *MockPacket) Close() {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	if !mp.connected {
		return
	}

	mp.connected = false
	log.Printf("[MOCK:%s] Connection closed", mp.name)
}

// Reset resets the mock to its initial state.
func (mp *MockPacket) Reset() {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.messages = make(map[string][][]byte)
	mp.subscribers = make(map[string][]Process)
	mp.connected = false
	mp.messageCount = 0
	mp.subscriptionCount = 0

	log.Printf("[MOCK:%s] Reset to initial state", mp.name)
}
