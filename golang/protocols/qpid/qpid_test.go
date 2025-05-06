package qpid

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hybridpipe.io/core"
	"hybridpipe.io/protocols/testutils"
	"pack.ag/amqp"
)

// TestQpidRouterInterface tests that the Qpid Packet implements the HybridPipe interface.
func TestQpidRouterInterface(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid Packet
	var router core.HybridPipe = &Packet{}
	assert.NotNil(t, router, "Qpid Packet should implement HybridPipe interface")
}

// setupQpidConfig sets up the Qpid configuration for testing.
func setupQpidConfig() *core.QpidConfig {
	// Get Qpid server address from environment variable or use default
	serverAddress := os.Getenv("QPID_SERVER_ADDRESS")
	if serverAddress == "" {
		serverAddress = "amqp://guest:guest@localhost:5672"
	}

	// Create a test configuration
	return &core.QpidConfig{
		QpidServer: serverAddress,
	}
}

// createQpidRouter creates a new Qpid router with the test configuration.
func createQpidRouter(t *testing.T) *Packet {
	// Create a new Qpid Packet
	router := &Packet{
		config:  setupQpidConfig(),
		senders: make(map[string]*amqp.Sender),
	}

	return router
}

// TestQpidRouterBasicOperations tests the basic operations of the Qpid router.
func TestQpidRouterBasicOperations(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid router
	router := createQpidRouter(t)

	// Run the basic operations test
	testutils.TestProtocolBasicOperations(t, router)
}

// TestQpidRouterContextAwareOperations tests the context-aware operations of the Qpid router.
func TestQpidRouterContextAwareOperations(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid router
	router := createQpidRouter(t)

	// Run the context-aware operations test
	testutils.TestProtocolContextAwareOperations(t, router)
}

// TestQpidRouterErrorHandling tests the error handling of the Qpid router.
func TestQpidRouterErrorHandling(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid router
	router := createQpidRouter(t)

	// Run the error handling test
	testutils.TestProtocolErrorHandling(t, router)
}

// TestQpidRouterConcurrentOperations tests the concurrent operations of the Qpid router.
func TestQpidRouterConcurrentOperations(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid router
	router := createQpidRouter(t)

	// Run the concurrent operations test
	testutils.TestProtocolConcurrentOperations(t, router)
}

// TestQpidRouterConnect tests the Connect method of the Qpid router.
func TestQpidRouterConnect(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid router
	router := createQpidRouter(t)

	// Test initial state
	assert.False(t, router.IsConnected(), "Router should not be connected initially")

	// Test Connect
	err := router.Connect()
	require.NoError(t, err, "Connect should not return an error")
	assert.True(t, router.IsConnected(), "Router should be connected after Connect()")

	// Test Connect when already connected
	err = router.Connect()
	assert.NoError(t, err, "Connect when already connected should not return an error")
	assert.True(t, router.IsConnected(), "Router should still be connected")

	// Test Disconnect
	err = router.Disconnect()
	require.NoError(t, err, "Disconnect should not return an error")
	assert.False(t, router.IsConnected(), "Router should not be connected after Disconnect()")

	// Test Close (alias for Disconnect)
	err = router.Connect()
	require.NoError(t, err, "Connect should not return an error")
	err = router.Close()
	require.NoError(t, err, "Close should not return an error")
	assert.False(t, router.IsConnected(), "Router should not be connected after Close()")
}

// TestQpidRouterDispatch tests the Dispatch method of the Qpid router.
func TestQpidRouterDispatch(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid router
	router := createQpidRouter(t)

	// Test Dispatch before Connect
	err := router.Dispatch("test-pipe", "test-data")
	assert.Error(t, err, "Dispatch before Connect should return an error")

	// Connect
	err = router.Connect()
	require.NoError(t, err, "Connect should not return an error")
	defer router.Close()

	// Test Dispatch with string data
	err = router.Dispatch("test-pipe", "test-data")
	assert.NoError(t, err, "Dispatch with string data should not return an error")

	// Test Dispatch with struct data
	type TestData struct {
		Message string
		Value   int
	}
	err = router.Dispatch("test-pipe", TestData{Message: "Hello", Value: 42})
	assert.NoError(t, err, "Dispatch with struct data should not return an error")

	// Test Dispatch with nil data
	err = router.Dispatch("test-pipe", nil)
	assert.NoError(t, err, "Dispatch with nil data should not return an error")

	// Test Dispatch with context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = router.DispatchWithContext(ctx, "test-pipe", "test-data")
	assert.NoError(t, err, "DispatchWithContext should not return an error")

	// Test Dispatch with canceled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context immediately
	err = router.DispatchWithContext(canceledCtx, "test-pipe", "test-data")
	assert.Error(t, err, "DispatchWithContext with canceled context should return an error")
}

// TestQpidRouterSubscribe tests the Subscribe method of the Qpid router.
func TestQpidRouterSubscribe(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid router
	router := createQpidRouter(t)

	// Test Subscribe before Connect
	err := router.Subscribe("test-pipe", func(data []byte) error { return nil })
	assert.Error(t, err, "Subscribe before Connect should return an error")

	// Connect
	err = router.Connect()
	require.NoError(t, err, "Connect should not return an error")
	defer router.Close()

	// Test Subscribe with valid callback
	err = router.Subscribe("test-pipe", func(data []byte) error { return nil })
	assert.NoError(t, err, "Subscribe with valid callback should not return an error")

	// Test Subscribe with nil callback
	err = router.Subscribe("test-pipe-nil", nil)
	assert.Error(t, err, "Subscribe with nil callback should return an error")

	// Test Subscribe with context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = router.SubscribeWithContext(ctx, "test-pipe-ctx", func(data []byte) error { return nil })
	assert.NoError(t, err, "SubscribeWithContext should not return an error")

	// Test Subscribe with canceled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context immediately
	err = router.SubscribeWithContext(canceledCtx, "test-pipe-canceled", func(data []byte) error { return nil })
	assert.Error(t, err, "SubscribeWithContext with canceled context should return an error")
}

// TestQpidRouterUnsubscribe tests the Unsubscribe method of the Qpid router.
func TestQpidRouterUnsubscribe(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid router
	router := createQpidRouter(t)

	// Test Unsubscribe before Connect
	err := router.Unsubscribe("test-pipe")
	assert.Error(t, err, "Unsubscribe before Connect should return an error")

	// Connect
	err = router.Connect()
	require.NoError(t, err, "Connect should not return an error")
	defer router.Close()

	// Test Unsubscribe from non-existent pipe
	err = router.Unsubscribe("non-existent-pipe")
	assert.Error(t, err, "Unsubscribe from non-existent pipe should return an error")

	// Subscribe to a pipe
	err = router.Subscribe("test-pipe", func(data []byte) error { return nil })
	require.NoError(t, err, "Subscribe should not return an error")

	// Test Unsubscribe from existing pipe
	err = router.Unsubscribe("test-pipe")
	assert.NoError(t, err, "Unsubscribe from existing pipe should not return an error")

	// Test Remove (alias for Unsubscribe)
	err = router.Subscribe("test-pipe-remove", func(data []byte) error { return nil })
	require.NoError(t, err, "Subscribe should not return an error")
	err = router.Remove("test-pipe-remove")
	assert.NoError(t, err, "Remove should not return an error")
}

// TestQpidRouterAccept tests the Accept method of the Qpid router.
func TestQpidRouterAccept(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid router
	router := createQpidRouter(t)

	// Test Accept before Connect
	err := router.Accept("test-pipe", func(data any) {})
	assert.Error(t, err, "Accept before Connect should return an error")

	// Connect
	err = router.Connect()
	require.NoError(t, err, "Connect should not return an error")
	defer router.Close()

	// Test Accept with valid callback
	err = router.Accept("test-pipe", func(data any) {})
	assert.NoError(t, err, "Accept with valid callback should not return an error")

	// Test Accept with nil callback
	err = router.Accept("test-pipe-nil", nil)
	assert.Error(t, err, "Accept with nil callback should return an error")
}

// TestQpidRouterEndToEnd tests the end-to-end functionality of the Qpid router.
func TestQpidRouterEndToEnd(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Create a new Qpid router
	router := createQpidRouter(t)

	// Connect
	err := router.Connect()
	require.NoError(t, err, "Connect should not return an error")
	defer router.Close()

	// Test pipe name
	pipeName := "test-pipe-end-to-end"

	// Test data
	testData := testutils.NewTestMessage("End-to-end test message")

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
	require.NoError(t, err, "Subscribe should not return an error")

	// Dispatch a message
	err = router.Dispatch(pipeName, testData)
	require.NoError(t, err, "Dispatch should not return an error")

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

	// Verify that the message was received
	assert.NotNil(t, receivedData, "Received data should not be nil")

	// Unsubscribe from the pipe
	err = router.Unsubscribe(pipeName)
	require.NoError(t, err, "Unsubscribe should not return an error")
}

// TestQpidRouterRegistration tests the registration of the Qpid router with the core registry.
func TestQpidRouterRegistration(t *testing.T) {
	// Skip in short mode as this test requires a Qpid server
	if testing.Short() {
		t.Skip("Skipping Qpid tests in short mode")
	}

	// Deploy a router of type QPID
	router, err := core.DeployRouter(core.QPID)
	require.NoError(t, err, "DeployRouter should not return an error")
	require.NotNil(t, router, "Deployed router should not be nil")

	// Check that the router is a Qpid Packet
	_, ok := router.(*Packet)
	assert.True(t, ok, "Deployed router should be a Qpid Packet")

	// Close the router
	err = router.Close()
	require.NoError(t, err, "Close should not return an error")
}
