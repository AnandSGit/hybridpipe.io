package testutils

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockExternalService tests the MockExternalService implementation.
func TestMockExternalService(t *testing.T) {
	// Create a new mock service
	service := NewMockExternalService()
	require.NotNil(t, service, "MockExternalService should not be nil")

	// Test initial state
	assert.False(t, service.Running, "Service should not be running initially")
	assert.Equal(t, 0, service.Port, "Port should be 0 initially")
	assert.Equal(t, "localhost", service.Host, "Host should be localhost initially")
	assert.Equal(t, 0, service.ErrorRate, "Error rate should be 0 initially")
	assert.Equal(t, 0, service.Latency, "Latency should be 0 initially")

	// Test Start
	err := service.Start()
	require.NoError(t, err, "Start should not return an error")
	assert.True(t, service.Running, "Service should be running after Start()")
	assert.Greater(t, service.Port, 0, "Port should be assigned after Start()")

	// Test Start when already running
	err = service.Start()
	assert.Error(t, err, "Start when already running should return an error")

	// Test Publish and Subscribe
	topic := "test-topic"
	data := []byte("test-data")

	// Subscribe to the topic
	var receivedData []byte
	var wg sync.WaitGroup
	wg.Add(1)

	err = service.Subscribe(topic, func(data []byte) error {
		receivedData = data
		wg.Done()
		return nil
	})
	require.NoError(t, err, "Subscribe should not return an error")

	// Publish a message
	err = service.Publish(topic, data)
	require.NoError(t, err, "Publish should not return an error")

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
	assert.Equal(t, data, receivedData, "Received data should match sent data")

	// Test Unsubscribe
	err = service.Unsubscribe(topic)
	require.NoError(t, err, "Unsubscribe should not return an error")

	// Test GetAddress
	address := service.GetAddress()
	assert.Contains(t, address, "localhost:", "Address should contain localhost")

	// Test SetErrorRate
	service.SetErrorRate(50)
	assert.Equal(t, 50, service.ErrorRate, "Error rate should be 50 after SetErrorRate(50)")

	// Test SetErrorRate with invalid values
	service.SetErrorRate(-10)
	assert.Equal(t, 0, service.ErrorRate, "Error rate should be 0 after SetErrorRate(-10)")
	service.SetErrorRate(150)
	assert.Equal(t, 100, service.ErrorRate, "Error rate should be 100 after SetErrorRate(150)")

	// Test SetLatency
	service.SetLatency(100)
	assert.Equal(t, 100, service.Latency, "Latency should be 100 after SetLatency(100)")

	// Test SetLatency with invalid values
	service.SetLatency(-10)
	assert.Equal(t, 0, service.Latency, "Latency should be 0 after SetLatency(-10)")

	// Test Stop
	err = service.Stop()
	require.NoError(t, err, "Stop should not return an error")
	assert.False(t, service.Running, "Service should not be running after Stop()")

	// Test Stop when not running
	err = service.Stop()
	assert.Error(t, err, "Stop when not running should return an error")

	// Test operations when not running
	err = service.Publish(topic, data)
	assert.Error(t, err, "Publish when not running should return an error")

	err = service.Subscribe(topic, func(data []byte) error { return nil })
	assert.Error(t, err, "Subscribe when not running should return an error")

	err = service.Unsubscribe(topic)
	assert.Error(t, err, "Unsubscribe when not running should return an error")
}

// TestMockExternalServiceWithErrorRate tests the MockExternalService with error simulation.
func TestMockExternalServiceWithErrorRate(t *testing.T) {
	// Create a new mock service with 100% error rate
	service := NewMockExternalService()
	service.Start()
	service.SetErrorRate(100)

	// Test Publish with error simulation
	err := service.Publish("test-topic", []byte("test-data"))
	assert.Error(t, err, "Publish with 100% error rate should return an error")

	// Test Subscribe with error simulation
	err = service.Subscribe("test-topic", func(data []byte) error { return nil })
	assert.Error(t, err, "Subscribe with 100% error rate should return an error")

	// Test Unsubscribe with error simulation
	err = service.Unsubscribe("test-topic")
	assert.Error(t, err, "Unsubscribe with 100% error rate should return an error")

	// Clean up
	service.SetErrorRate(0)
	service.Stop()
}

// TestMockExternalServiceWithLatency tests the MockExternalService with latency simulation.
func TestMockExternalServiceWithLatency(t *testing.T) {
	// Create a new mock service with latency
	service := NewMockExternalService()
	service.Start()
	service.SetLatency(100) // 100ms latency

	// Test Publish with latency simulation
	start := time.Now()
	err := service.Publish("test-topic", []byte("test-data"))
	elapsed := time.Since(start)
	require.NoError(t, err, "Publish should not return an error")
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(100), "Publish should take at least 100ms with 100ms latency")

	// Test Subscribe with latency simulation
	start = time.Now()
	err = service.Subscribe("test-topic", func(data []byte) error { return nil })
	elapsed = time.Since(start)
	require.NoError(t, err, "Subscribe should not return an error")
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(100), "Subscribe should take at least 100ms with 100ms latency")

	// Test Unsubscribe with latency simulation
	start = time.Now()
	err = service.Unsubscribe("test-topic")
	elapsed = time.Since(start)
	require.NoError(t, err, "Unsubscribe should not return an error")
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(100), "Unsubscribe should take at least 100ms with 100ms latency")

	// Clean up
	service.SetLatency(0)
	service.Stop()
}
