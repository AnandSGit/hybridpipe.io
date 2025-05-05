package monitoring

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hybridpipe.io/core"
	"hybridpipe.io/protocols/mock"
)

// TestMonitoredRouter tests the MonitoredRouter middleware.
func TestMonitoredRouter(t *testing.T) {
	// Create a mock router
	router := mock.NewMockRouter()
	err := router.Connect()
	require.NoError(t, err, "Failed to connect to mock router")
	defer router.Close()

	// Create a monitored router
	monitoredRouter := NewMonitoredRouter(router, "test-router")

	// Test basic operations
	t.Run("BasicOperations", func(t *testing.T) {
		testBasicOperations(t, monitoredRouter, router)
	})

	// Test monitoring functionality
	t.Run("MonitoringFunctionality", func(t *testing.T) {
		testMonitoringFunctionality(t, monitoredRouter)
	})

	// Test monitoring callback
	t.Run("MonitoringCallback", func(t *testing.T) {
		testMonitoringCallback(t, monitoredRouter)
	})

	// Test monitoring server
	t.Run("MonitoringServer", func(t *testing.T) {
		testMonitoringServer(t, monitoredRouter)
	})
}

// testBasicOperations tests the basic operations of the monitored router.
func testBasicOperations(t *testing.T, monitoredRouter *MonitoredRouter, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-monitor-pipe"

	// Test data - use a simple string instead of a map to avoid serialization issues
	testData := "Hello, Monitored Router!"

	// Wait group for synchronization
	done := make(chan struct{})

	// Register types for Gob
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})

	// Subscribe to the pipe
	err := monitoredRouter.Subscribe(pipeName, func(data []byte) error {
		var decoded string
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		close(done)
		return nil
	})
	require.NoError(t, err, "Failed to subscribe to pipe")

	// Dispatch a message
	err = monitoredRouter.Dispatch(pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message")

	// Wait for the message to be received
	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Check that the message was counted
	assert.Equal(t, uint64(1), monitoredRouter.Stats.GetMessagesSent(), "Should have sent 1 message")
	assert.Equal(t, uint64(1), monitoredRouter.Stats.GetMessagesReceived(), "Should have received 1 message")
	assert.Greater(t, monitoredRouter.Stats.GetBytesSent(), uint64(0), "Should have sent some bytes")
	assert.Greater(t, monitoredRouter.Stats.GetBytesReceived(), uint64(0), "Should have received some bytes")

	// Unsubscribe from the pipe
	err = monitoredRouter.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")
}

// testMonitoringFunctionality tests the monitoring functionality of the monitored router.
func testMonitoringFunctionality(t *testing.T, monitoredRouter *MonitoredRouter) {
	// Reset stats
	monitoredRouter.ResetStats()

	// Check that stats were reset
	assert.Equal(t, uint64(0), monitoredRouter.Stats.GetMessagesSent(), "Should have 0 messages sent after reset")
	assert.Equal(t, uint64(0), monitoredRouter.Stats.GetMessagesReceived(), "Should have 0 messages received after reset")
	assert.Equal(t, uint64(0), monitoredRouter.Stats.GetBytesSent(), "Should have 0 bytes sent after reset")
	assert.Equal(t, uint64(0), monitoredRouter.Stats.GetBytesReceived(), "Should have 0 bytes received after reset")

	// Test pipe name
	pipeName := "test-monitor-functionality-pipe"

	// Test data - use a simple string to avoid serialization issues
	testData := "Hello, Monitored Router Functionality!"

	// Wait group for synchronization
	done := make(chan struct{})

	// Register types for Gob
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})

	// Subscribe to the pipe
	err := monitoredRouter.Subscribe(pipeName, func(data []byte) error {
		var decoded string
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		close(done)
		return nil
	})
	require.NoError(t, err, "Failed to subscribe to pipe")

	// Dispatch a message
	err = monitoredRouter.Dispatch(pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message")

	// Wait for the message to be received
	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Check that the message was counted
	assert.Equal(t, uint64(1), monitoredRouter.Stats.GetMessagesSent(), "Should have sent 1 message")
	assert.Equal(t, uint64(1), monitoredRouter.Stats.GetMessagesReceived(), "Should have received 1 message")
	assert.Greater(t, monitoredRouter.Stats.GetBytesSent(), uint64(0), "Should have sent some bytes")
	assert.Greater(t, monitoredRouter.Stats.GetBytesReceived(), uint64(0), "Should have received some bytes")

	// Get stats
	stats := monitoredRouter.GetStats()
	assert.NotNil(t, stats, "Stats should not be nil")

	// Check uptime and idle time
	assert.Greater(t, stats.GetUptime(), time.Duration(0), "Uptime should be greater than 0")
	assert.GreaterOrEqual(t, stats.GetIdleTime(), time.Duration(0), "Idle time should be greater than or equal to 0")

	// Unsubscribe from the pipe
	err = monitoredRouter.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")
}

// testMonitoringCallback tests the monitoring callback functionality.
func testMonitoringCallback(t *testing.T, monitoredRouter *MonitoredRouter) {
	// Reset stats
	monitoredRouter.ResetStats()

	// Create a channel to receive monitoring events
	eventChan := make(chan MonitoringEvent, 10)

	// Set a custom monitoring callback
	monitoredRouter.SetMonitoringCallback(func(event MonitoringEvent) {
		eventChan <- event
	})

	// Test pipe name
	pipeName := "test-monitor-callback-pipe"

	// Test data - use a simple string to avoid serialization issues
	testData := "Hello, Monitored Router Callback!"

	// Wait group for synchronization
	done := make(chan struct{})

	// Register types for Gob
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})

	// Subscribe to the pipe
	err := monitoredRouter.Subscribe(pipeName, func(data []byte) error {
		var decoded string
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		close(done)
		return nil
	})
	require.NoError(t, err, "Failed to subscribe to pipe")

	// Dispatch a message
	err = monitoredRouter.Dispatch(pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message")

	// Wait for the message to be received
	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Check that we received monitoring events
	var dispatchEvent, receiveEvent MonitoringEvent

	// Wait for dispatch event
	select {
	case dispatchEvent = <-eventChan:
		// Event received
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for dispatch event")
	}

	// Wait for receive event
	select {
	case receiveEvent = <-eventChan:
		// Event received
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for receive event")
	}

	// Check dispatch event
	assert.Equal(t, "dispatch", dispatchEvent.Type, "Dispatch event type should be 'dispatch'")
	assert.Equal(t, pipeName, dispatchEvent.Pipe, "Dispatch event pipe should match")
	assert.Equal(t, "test-router", dispatchEvent.RouterName, "Dispatch event router name should match")
	assert.Greater(t, dispatchEvent.Size, uint64(0), "Dispatch event size should be greater than 0")

	// Check receive event
	assert.Equal(t, "receive", receiveEvent.Type, "Receive event type should be 'receive'")
	assert.Equal(t, pipeName, receiveEvent.Pipe, "Receive event pipe should match")
	assert.Equal(t, "test-router", receiveEvent.RouterName, "Receive event router name should match")
	assert.Greater(t, receiveEvent.Size, uint64(0), "Receive event size should be greater than 0")

	// Unsubscribe from the pipe
	err = monitoredRouter.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")
}

// testMonitoringServer tests the monitoring server functionality.
func testMonitoringServer(t *testing.T, monitoredRouter *MonitoredRouter) {
	// Create a monitoring server
	server := NewMonitoringServer(":8080")

	// Register the monitored router
	server.RegisterRouter("test-router", monitoredRouter)

	// Test the stats endpoint
	t.Run("StatsEndpoint", func(t *testing.T) {
		// Create a request to the stats endpoint
		req := httptest.NewRequest("GET", "/stats", nil)
		w := httptest.NewRecorder()

		// Handle the request
		server.handleStats(w, req)

		// Check the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200 OK")

		// Parse the response body
		var stats map[string]map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&stats)
		require.NoError(t, err, "Failed to decode response body")

		// Check that the stats include our router
		routerStats, ok := stats["test-router"]
		assert.True(t, ok, "Stats should include our router")

		// Check that the router stats include the expected fields
		assert.Contains(t, routerStats, "messages_sent", "Router stats should include messages_sent")
		assert.Contains(t, routerStats, "messages_received", "Router stats should include messages_received")
		assert.Contains(t, routerStats, "bytes_sent", "Router stats should include bytes_sent")
		assert.Contains(t, routerStats, "bytes_received", "Router stats should include bytes_received")
	})

	// Test the router stats endpoint
	t.Run("RouterStatsEndpoint", func(t *testing.T) {
		// Create a request to the router stats endpoint
		req := httptest.NewRequest("GET", "/stats/test-router", nil)
		w := httptest.NewRecorder()

		// Handle the request
		server.handleRouterStats(w, req)

		// Check the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200 OK")

		// Parse the response body
		var routerStats map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&routerStats)
		require.NoError(t, err, "Failed to decode response body")

		// Check that the router stats include the expected fields
		assert.Contains(t, routerStats, "messages_sent", "Router stats should include messages_sent")
		assert.Contains(t, routerStats, "messages_received", "Router stats should include messages_received")
		assert.Contains(t, routerStats, "bytes_sent", "Router stats should include bytes_sent")
		assert.Contains(t, routerStats, "bytes_received", "Router stats should include bytes_received")
	})

	// Test the reset endpoint
	t.Run("ResetEndpoint", func(t *testing.T) {
		// Create a request to the reset endpoint
		req := httptest.NewRequest("POST", "/reset", nil)
		w := httptest.NewRecorder()

		// Handle the request
		server.handleReset(w, req)

		// Check the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200 OK")

		// Check that the stats were reset
		assert.Equal(t, uint64(0), monitoredRouter.Stats.GetMessagesSent(), "Should have 0 messages sent after reset")
		assert.Equal(t, uint64(0), monitoredRouter.Stats.GetMessagesReceived(), "Should have 0 messages received after reset")
		assert.Equal(t, uint64(0), monitoredRouter.Stats.GetBytesSent(), "Should have 0 bytes sent after reset")
		assert.Equal(t, uint64(0), monitoredRouter.Stats.GetBytesReceived(), "Should have 0 bytes received after reset")
	})

	// Test the router reset endpoint
	t.Run("RouterResetEndpoint", func(t *testing.T) {
		// First, dispatch a message to have some stats
		pipeName := "test-monitor-server-pipe"
		testData := "Hello, Monitored Router Server!"

		// Wait group for synchronization
		done := make(chan struct{})

		// Register types for Gob
		gob.Register(map[string]interface{}{})
		gob.Register([]interface{}{})

		// Subscribe to the pipe
		err := monitoredRouter.Subscribe(pipeName, func(data []byte) error {
			var decoded string
			if err := core.Decode(data, &decoded); err != nil {
				t.Errorf("Failed to decode message: %v", err)
				return err
			}
			close(done)
			return nil
		})
		require.NoError(t, err, "Failed to subscribe to pipe")

		// Dispatch a message
		err = monitoredRouter.Dispatch(pipeName, testData)
		require.NoError(t, err, "Failed to dispatch message")

		// Wait for the message to be received
		select {
		case <-done:
			// Message received
		case <-time.After(5 * time.Second):
			t.Fatal("Timed out waiting for message")
		}

		// Create a request to the router reset endpoint
		req := httptest.NewRequest("POST", "/reset/test-router", nil)
		w := httptest.NewRecorder()

		// Handle the request
		server.handleRouterReset(w, req)

		// Check the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200 OK")

		// Check that the stats were reset
		assert.Equal(t, uint64(0), monitoredRouter.Stats.GetMessagesSent(), "Should have 0 messages sent after reset")
		assert.Equal(t, uint64(0), monitoredRouter.Stats.GetMessagesReceived(), "Should have 0 messages received after reset")
		assert.Equal(t, uint64(0), monitoredRouter.Stats.GetBytesSent(), "Should have 0 bytes sent after reset")
		assert.Equal(t, uint64(0), monitoredRouter.Stats.GetBytesReceived(), "Should have 0 bytes received after reset")

		// Unsubscribe from the pipe
		err = monitoredRouter.Unsubscribe(pipeName)
		require.NoError(t, err, "Failed to unsubscribe from pipe")
	})
}

// TestMessageStats tests the MessageStats struct.
func TestMessageStats(t *testing.T) {
	// Create a new MessageStats
	stats := NewMessageStats()

	// Check initial values
	assert.Equal(t, uint64(0), stats.GetMessagesSent(), "Initial messages sent should be 0")
	assert.Equal(t, uint64(0), stats.GetMessagesReceived(), "Initial messages received should be 0")
	assert.Equal(t, uint64(0), stats.GetBytesSent(), "Initial bytes sent should be 0")
	assert.Equal(t, uint64(0), stats.GetBytesReceived(), "Initial bytes received should be 0")
	assert.Equal(t, uint64(0), stats.GetErrors(), "Initial errors should be 0")

	// Test incrementing sent
	stats.IncrementSent(100)
	assert.Equal(t, uint64(1), stats.GetMessagesSent(), "Messages sent should be 1")
	assert.Equal(t, uint64(100), stats.GetBytesSent(), "Bytes sent should be 100")

	// Test incrementing received
	stats.IncrementReceived(200)
	assert.Equal(t, uint64(1), stats.GetMessagesReceived(), "Messages received should be 1")
	assert.Equal(t, uint64(200), stats.GetBytesReceived(), "Bytes received should be 200")

	// Test incrementing errors
	stats.IncrementError()
	assert.Equal(t, uint64(1), stats.GetErrors(), "Errors should be 1")

	// Test resetting
	stats.Reset()
	assert.Equal(t, uint64(0), stats.GetMessagesSent(), "Messages sent should be 0 after reset")
	assert.Equal(t, uint64(0), stats.GetMessagesReceived(), "Messages received should be 0 after reset")
	assert.Equal(t, uint64(0), stats.GetBytesSent(), "Bytes sent should be 0 after reset")
	assert.Equal(t, uint64(0), stats.GetBytesReceived(), "Bytes received should be 0 after reset")
	assert.Equal(t, uint64(0), stats.GetErrors(), "Errors should be 0 after reset")

	// Test concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				stats.IncrementSent(1)
				stats.IncrementReceived(1)
				stats.IncrementError()
			}
		}()
	}

	wg.Wait()

	// Check final values
	assert.Equal(t, uint64(numGoroutines*numOperations), stats.GetMessagesSent(), "Messages sent should match")
	assert.Equal(t, uint64(numGoroutines*numOperations), stats.GetMessagesReceived(), "Messages received should match")
	assert.Equal(t, uint64(numGoroutines*numOperations), stats.GetErrors(), "Errors should match")
	assert.Equal(t, uint64(numGoroutines*numOperations), stats.GetBytesSent(), "Bytes sent should match")
	assert.Equal(t, uint64(numGoroutines*numOperations), stats.GetBytesReceived(), "Bytes received should match")

	// Test GetStats
	statsMap := stats.GetStats()
	assert.Equal(t, uint64(numGoroutines*numOperations), statsMap["messages_sent"], "Messages sent in stats map should match")
	assert.Equal(t, uint64(numGoroutines*numOperations), statsMap["messages_received"], "Messages received in stats map should match")
	assert.Equal(t, uint64(numGoroutines*numOperations), statsMap["errors"], "Errors in stats map should match")
	assert.Equal(t, uint64(numGoroutines*numOperations), statsMap["bytes_sent"], "Bytes sent in stats map should match")
	assert.Equal(t, uint64(numGoroutines*numOperations), statsMap["bytes_received"], "Bytes received in stats map should match")
}

// TestMonitoringEvent tests the MonitoringEvent struct.
func TestMonitoringEvent(t *testing.T) {
	// Create a new MonitoringEvent
	event := MonitoringEvent{
		Type:       "test",
		Pipe:       "test-pipe",
		Timestamp:  time.Now(),
		Size:       100,
		Error:      "test-error",
		RouterName: "test-router",
	}

	// Check values
	assert.Equal(t, "test", event.Type, "Event type should match")
	assert.Equal(t, "test-pipe", event.Pipe, "Event pipe should match")
	assert.Equal(t, uint64(100), event.Size, "Event size should match")
	assert.Equal(t, "test-error", event.Error, "Event error should match")
	assert.Equal(t, "test-router", event.RouterName, "Event router name should match")
}

// TestContextAwareMonitoredRouter tests the context-aware operations of the monitored router.
func TestContextAwareMonitoredRouter(t *testing.T) {
	// Create a mock router
	router := mock.NewMockRouter()
	err := router.Connect()
	require.NoError(t, err, "Failed to connect to mock router")
	defer router.Close()

	// Create a monitored router
	monitoredRouter := NewMonitoredRouter(router, "test-ctx-router")

	// Test pipe name
	pipeName := "test-ctx-monitor-pipe"

	// Test data - use a simple string to avoid serialization issues
	testData := "Hello, Context-Aware Monitored Router!"

	// Wait group for synchronization
	done := make(chan struct{})

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register types for Gob
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})

	// Subscribe to the pipe with context
	err = monitoredRouter.SubscribeWithContext(ctx, pipeName, func(data []byte) error {
		var decoded string
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		close(done)
		return nil
	})
	require.NoError(t, err, "Failed to subscribe to pipe with context")

	// Dispatch a message with context
	err = monitoredRouter.DispatchWithContext(ctx, pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message with context")

	// Wait for the message to be received
	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Check that the message was counted
	assert.Equal(t, uint64(1), monitoredRouter.Stats.GetMessagesSent(), "Should have sent 1 message")
	assert.Equal(t, uint64(1), monitoredRouter.Stats.GetMessagesReceived(), "Should have received 1 message")
	assert.Greater(t, monitoredRouter.Stats.GetBytesSent(), uint64(0), "Should have sent some bytes")
	assert.Greater(t, monitoredRouter.Stats.GetBytesReceived(), uint64(0), "Should have received some bytes")

	// Unsubscribe from the pipe
	err = monitoredRouter.Unsubscribe(pipeName)
	require.NoError(t, err, "Failed to unsubscribe from pipe")
}
