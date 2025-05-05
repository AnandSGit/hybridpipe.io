package nsq

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
)

// TestNSQRouterInterface tests that the NSQ Packet implements the HybridPipe interface.
func TestNSQRouterInterface(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ Packet
	var router core.HybridPipe = &Packet{}
	assert.NotNil(t, router, "NSQ Packet should implement HybridPipe interface")
}

// setupNSQConfig sets up the NSQ configuration for testing.
func setupNSQConfig() *core.NSQConfig {
	// Get NSQ server address from environment variable or use default
	nsqdAddress := os.Getenv("NSQ_NSQD_ADDRESS")
	if nsqdAddress == "" {
		nsqdAddress = "localhost"
	}

	lookupdAddress := os.Getenv("NSQ_LOOKUPD_ADDRESS")
	if lookupdAddress == "" {
		lookupdAddress = "localhost:4161"
	}

	// Create a test configuration
	return &core.NSQConfig{
		NSQDAddress:         nsqdAddress,
		NSQDPort:            4150,
		NSQLookupdAddresses: lookupdAddress,
		NSQClientID:         "hybridpipe-test-" + time.Now().Format("20060102150405"),
		NSQMaxInFlight:      100,
		NSQRequeueDelay:     5,
		NSQConnectTimeout:   10,
	}
}

// createNSQRouter creates a new NSQ router with the test configuration.
func createNSQRouter(t *testing.T) *Packet {
	// Create a new NSQ Packet
	router := &Packet{
		config: setupNSQConfig(),
	}

	return router
}

// TestNSQRouterBasicOperations tests the basic operations of the NSQ router.
func TestNSQRouterBasicOperations(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ router
	router := createNSQRouter(t)

	// Run the basic operations test
	testutils.TestProtocolBasicOperations(t, router)
}

// TestNSQRouterContextAwareOperations tests the context-aware operations of the NSQ router.
func TestNSQRouterContextAwareOperations(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ router
	router := createNSQRouter(t)

	// Run the context-aware operations test
	testutils.TestProtocolContextAwareOperations(t, router)
}

// TestNSQRouterErrorHandling tests the error handling of the NSQ router.
func TestNSQRouterErrorHandling(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ router
	router := createNSQRouter(t)

	// Run the error handling test
	testutils.TestProtocolErrorHandling(t, router)
}

// TestNSQRouterConcurrentOperations tests the concurrent operations of the NSQ router.
func TestNSQRouterConcurrentOperations(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ router
	router := createNSQRouter(t)

	// Run the concurrent operations test
	testutils.TestProtocolConcurrentOperations(t, router)
}

// TestNSQRouterConnect tests the Connect method of the NSQ router.
func TestNSQRouterConnect(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ router
	router := createNSQRouter(t)

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

// TestNSQRouterDispatch tests the Dispatch method of the NSQ router.
func TestNSQRouterDispatch(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ router
	router := createNSQRouter(t)

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

// TestNSQRouterSubscribe tests the Subscribe method of the NSQ router.
func TestNSQRouterSubscribe(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ router
	router := createNSQRouter(t)

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

// TestNSQRouterUnsubscribe tests the Unsubscribe method of the NSQ router.
func TestNSQRouterUnsubscribe(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ router
	router := createNSQRouter(t)

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

// TestNSQRouterAccept tests the Accept method of the NSQ router.
func TestNSQRouterAccept(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ router
	router := createNSQRouter(t)

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

// TestNSQRouterEndToEnd tests the end-to-end functionality of the NSQ router.
func TestNSQRouterEndToEnd(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Create a new NSQ router
	router := createNSQRouter(t)

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

// TestNSQRouterRegistration tests the registration of the NSQ router with the core registry.
func TestNSQRouterRegistration(t *testing.T) {
	// Skip in short mode as this test requires an NSQ server
	if testing.Short() {
		t.Skip("Skipping NSQ tests in short mode")
	}

	// Deploy a router of type NSQ
	router, err := core.DeployRouter(core.NSQ)
	require.NoError(t, err, "DeployRouter should not return an error")
	require.NotNil(t, router, "Deployed router should not be nil")

	// Check that the router is an NSQ Packet
	_, ok := router.(*Packet)
	assert.True(t, ok, "Deployed router should be an NSQ Packet")

	// Close the router
	err = router.Close()
	require.NoError(t, err, "Close should not return an error")
}
