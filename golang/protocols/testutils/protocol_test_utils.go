// Package testutils provides utilities for testing protocol implementations.
package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hybridpipe.io/core"
)

// TestMessage is a test message structure used for testing protocol implementations.
type TestMessage struct {
	ID      string    `json:"id"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

// NewTestMessage creates a new test message with the given content.
func NewTestMessage(content string) TestMessage {
	return TestMessage{
		ID:      fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Content: content,
		Time:    time.Now(),
	}
}

// TestProtocolBasicOperations tests the basic operations of a HybridPipe implementation.
// It tests Connect, Dispatch, Subscribe, Unsubscribe, and Disconnect.
func TestProtocolBasicOperations(t *testing.T, router core.HybridPipe) {
	// Test connection
	err := router.Connect()
	require.NoError(t, err, "Failed to connect")
	assert.True(t, router.IsConnected(), "Router should be connected after Connect()")

	// Test pipe name
	pipeName := fmt.Sprintf("test-pipe-%d", time.Now().UnixNano())

	// Test data
	testData := NewTestMessage("Hello, Protocol Test!")

	// Wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe to the pipe
	var receivedData []byte
	err = router.Subscribe(pipeName, func(data []byte) error {
		receivedData = data
		wg.Done()
		return nil
	})
	require.NoError(t, err, "Failed to subscribe to pipe")

	// Dispatch a message
	err = router.Dispatch(pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message")

	// Wait for the message to be received
	waitTimeout := 5 * time.Second
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Message received
	case <-time.After(waitTimeout):
		t.Fatal("Timed out waiting for message")
	}

	// Verify the received data
	var receivedMsg TestMessage
	err = json.Unmarshal(receivedData, &receivedMsg)
	require.NoError(t, err, "Failed to unmarshal received data")
	assert.Equal(t, testData.Content, receivedMsg.Content, "Received message content should match sent message")

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")

	// Close the router
	err = router.Close()
	require.NoError(t, err, "Failed to close router")
	assert.False(t, router.IsConnected(), "Router should not be connected after Close()")
}

// TestProtocolContextAwareOperations tests the context-aware operations of a HybridPipe implementation.
// It tests DispatchWithContext and SubscribeWithContext.
func TestProtocolContextAwareOperations(t *testing.T, router core.HybridPipe) {
	// Check if the router implements the ContextAwareHybridPipe interface
	ctxRouter, ok := router.(core.ContextAwareHybridPipe)
	if !ok {
		t.Skip("Router does not implement ContextAwareHybridPipe interface")
		return
	}

	// Test connection
	err := router.Connect()
	require.NoError(t, err, "Failed to connect")

	// Test pipe name
	pipeName := fmt.Sprintf("test-ctx-pipe-%d", time.Now().UnixNano())

	// Test data
	testData := NewTestMessage("Hello, Context-Aware Protocol Test!")

	// Wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(1)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe to the pipe with context
	var receivedData []byte
	err = ctxRouter.SubscribeWithContext(ctx, pipeName, func(data []byte) error {
		receivedData = data
		wg.Done()
		return nil
	})
	require.NoError(t, err, "Failed to subscribe to pipe with context")

	// Dispatch a message with context
	err = ctxRouter.DispatchWithContext(ctx, pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message with context")

	// Wait for the message to be received
	waitTimeout := 5 * time.Second
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Message received
	case <-time.After(waitTimeout):
		t.Fatal("Timed out waiting for message")
	}

	// Verify the received data
	var receivedMsg TestMessage
	err = json.Unmarshal(receivedData, &receivedMsg)
	require.NoError(t, err, "Failed to unmarshal received data")
	assert.Equal(t, testData.Content, receivedMsg.Content, "Received message content should match sent message")

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")

	// Close the router
	err = router.Close()
	require.NoError(t, err, "Failed to close router")
}

// TestProtocolErrorHandling tests the error handling of a HybridPipe implementation.
// It tests error cases for Connect, Dispatch, Subscribe, Unsubscribe, and Disconnect.
func TestProtocolErrorHandling(t *testing.T, router core.HybridPipe) {
	// Test dispatch before connect
	err := router.Dispatch("test-pipe", "test-data")
	assert.Error(t, err, "Dispatch before connect should fail")

	// Test subscribe before connect
	err = router.Subscribe("test-pipe", func(data []byte) error { return nil })
	assert.Error(t, err, "Subscribe before connect should fail")

	// Test unsubscribe before connect
	err = router.Unsubscribe("test-pipe")
	assert.Error(t, err, "Unsubscribe before connect should fail")

	// Connect
	err = router.Connect()
	require.NoError(t, err, "Failed to connect")

	// Test subscribe with nil callback
	err = router.Subscribe("test-pipe", nil)
	assert.Error(t, err, "Subscribe with nil callback should fail")

	// Test unsubscribe from non-existent pipe
	err = router.Unsubscribe("non-existent-pipe")
	assert.Error(t, err, "Unsubscribe from non-existent pipe should fail")

	// Close the router
	err = router.Close()
	require.NoError(t, err, "Failed to close router")

	// Test dispatch after close
	err = router.Dispatch("test-pipe", "test-data")
	assert.Error(t, err, "Dispatch after close should fail")

	// Test subscribe after close
	err = router.Subscribe("test-pipe", func(data []byte) error { return nil })
	assert.Error(t, err, "Subscribe after close should fail")

	// Test unsubscribe after close
	err = router.Unsubscribe("test-pipe")
	assert.Error(t, err, "Unsubscribe after close should fail")
}

// TestProtocolConcurrentOperations tests the concurrent operations of a HybridPipe implementation.
// It tests concurrent Dispatch and Subscribe operations.
func TestProtocolConcurrentOperations(t *testing.T, router core.HybridPipe) {
	// Test connection
	err := router.Connect()
	require.NoError(t, err, "Failed to connect")

	// Number of concurrent operations
	numOperations := 10

	// Wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(numOperations)

	// Error channel for collecting errors
	errChan := make(chan error, numOperations)

	// Run concurrent operations
	for i := 0; i < numOperations; i++ {
		go func(index int) {
			defer wg.Done()

			// Test pipe name
			pipeName := fmt.Sprintf("test-concurrent-pipe-%d", index)

			// Test data
			testData := NewTestMessage(fmt.Sprintf("Concurrent message %d", index))

			// Subscribe to the pipe
			var receivedWg sync.WaitGroup
			receivedWg.Add(1)

			err := router.Subscribe(pipeName, func(data []byte) error {
				receivedWg.Done()
				return nil
			})
			if err != nil {
				errChan <- fmt.Errorf("failed to subscribe to pipe %s: %w", pipeName, err)
				return
			}

			// Dispatch a message
			err = router.Dispatch(pipeName, testData)
			if err != nil {
				errChan <- fmt.Errorf("failed to dispatch message to pipe %s: %w", pipeName, err)
				return
			}

			// Wait for the message to be received
			waitTimeout := 5 * time.Second
			done := make(chan struct{})
			go func() {
				receivedWg.Wait()
				close(done)
			}()

			select {
			case <-done:
				// Message received
			case <-time.After(waitTimeout):
				errChan <- fmt.Errorf("timed out waiting for message on pipe %s", pipeName)
				return
			}

			// Unsubscribe from the pipe
			err = router.Unsubscribe(pipeName)
			if err != nil {
				errChan <- fmt.Errorf("failed to unsubscribe from pipe %s: %w", pipeName, err)
				return
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Error(err)
	}

	// Close the router
	err = router.Close()
	require.NoError(t, err, "Failed to close router")
}

// MockExternalService provides a mock implementation of an external service for testing.
// This can be used to mock external dependencies like message brokers.
type MockExternalService struct {
	// Messages stores messages sent to the service
	Messages map[string][][]byte
	// Subscribers stores callbacks for each topic
	Subscribers map[string][]func([]byte) error
	// Running indicates if the service is running
	Running bool
	// Port is the port the service is listening on
	Port int
	// Host is the host the service is listening on
	Host string
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// errorRate is the percentage of operations that should fail (0-100)
	ErrorRate int
	// latency is the simulated latency in milliseconds
	Latency int
}

// NewMockExternalService creates a new mock external service.
func NewMockExternalService() *MockExternalService {
	return &MockExternalService{
		Messages:    make(map[string][][]byte),
		Subscribers: make(map[string][]func([]byte) error),
		Running:     false,
		Port:        0, // Will be assigned when started
		Host:        "localhost",
		ErrorRate:   0,
		Latency:     0,
	}
}

// Start starts the mock external service.
func (m *MockExternalService) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.Running {
		return fmt.Errorf("service already running")
	}

	// Assign a random port if not specified
	if m.Port == 0 {
		m.Port = 10000 + rand.Intn(10000)
	}

	m.Running = true
	return nil
}

// Stop stops the mock external service.
func (m *MockExternalService) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Running {
		return fmt.Errorf("service not running")
	}

	m.Running = false
	return nil
}

// Publish publishes a message to the specified topic.
func (m *MockExternalService) Publish(topic string, data []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Running {
		return fmt.Errorf("service not running")
	}

	// Simulate latency
	if m.Latency > 0 {
		time.Sleep(time.Duration(m.Latency) * time.Millisecond)
	}

	// Simulate errors
	if m.ErrorRate > 0 && rand.Intn(100) < m.ErrorRate {
		return fmt.Errorf("simulated error")
	}

	// Store the message
	if _, exists := m.Messages[topic]; !exists {
		m.Messages[topic] = make([][]byte, 0)
	}
	m.Messages[topic] = append(m.Messages[topic], data)

	// Notify subscribers
	if subscribers, exists := m.Subscribers[topic]; exists {
		for _, subscriber := range subscribers {
			go subscriber(data)
		}
	}

	return nil
}

// Subscribe subscribes to the specified topic.
func (m *MockExternalService) Subscribe(topic string, callback func([]byte) error) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Running {
		return fmt.Errorf("service not running")
	}

	// Simulate latency
	if m.Latency > 0 {
		time.Sleep(time.Duration(m.Latency) * time.Millisecond)
	}

	// Simulate errors
	if m.ErrorRate > 0 && rand.Intn(100) < m.ErrorRate {
		return fmt.Errorf("simulated error")
	}

	// Add the subscriber
	if _, exists := m.Subscribers[topic]; !exists {
		m.Subscribers[topic] = make([]func([]byte) error, 0)
	}
	m.Subscribers[topic] = append(m.Subscribers[topic], callback)

	// Send existing messages to the new subscriber
	if messages, exists := m.Messages[topic]; exists {
		for _, msg := range messages {
			go callback(msg)
		}
	}

	return nil
}

// Unsubscribe unsubscribes from the specified topic.
func (m *MockExternalService) Unsubscribe(topic string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.Running {
		return fmt.Errorf("service not running")
	}

	// Simulate latency
	if m.Latency > 0 {
		time.Sleep(time.Duration(m.Latency) * time.Millisecond)
	}

	// Simulate errors
	if m.ErrorRate > 0 && rand.Intn(100) < m.ErrorRate {
		return fmt.Errorf("simulated error")
	}

	delete(m.Subscribers, topic)
	return nil
}

// GetAddress returns the address of the mock service.
func (m *MockExternalService) GetAddress() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

// SetErrorRate sets the error rate for the mock service.
func (m *MockExternalService) SetErrorRate(rate int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if rate < 0 {
		rate = 0
	} else if rate > 100 {
		rate = 100
	}

	m.ErrorRate = rate
}

// SetLatency sets the simulated latency for the mock service.
func (m *MockExternalService) SetLatency(latency int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if latency < 0 {
		latency = 0
	}

	m.Latency = latency
}
