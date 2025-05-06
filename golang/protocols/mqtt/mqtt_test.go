package mqtt

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hybridpipe.io/core"
	"hybridpipe.io/protocols/testutils"
)

// TestMQTTRouterInterface tests that the MQTT Packet implements the HybridPipe interface.
func TestMQTTRouterInterface(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT Packet
	var router core.HybridPipe = &Packet{}
	assert.NotNil(t, router, "MQTT Packet should implement HybridPipe interface")
}

// setupMQTTConfig sets up the MQTT configuration for testing.
func setupMQTTConfig() *core.MQTTConfig {
	// Get MQTT broker address from environment variable or use default
	brokerAddress := os.Getenv("MQTT_BROKER_ADDRESS")
	if brokerAddress == "" {
		brokerAddress = "tcp://localhost:1883"
	}

	// Create a test configuration
	return &core.MQTTConfig{
		MQTTBroker:         brokerAddress,
		MQTTClientID:       "hybridpipe-test-" + time.Now().Format("20060102150405"),
		MQTTQoS:            1,
		MQTTRetain:         false,
		MQTTCleanSession:   true,
		MQTTPublishTimeout: 10,
		MQTTConnectTimeout: 10,
	}
}

// createMQTTRouter creates a new MQTT router with the test configuration.
func createMQTTRouter(t *testing.T) *Packet {
	// Create a new MQTT Packet
	router := &Packet{
		subscriptions: make(map[string]mqtt.Token),
		handlers:      make(map[string]mqtt.MessageHandler),
		config:        setupMQTTConfig(),
	}

	return router
}

// TestMQTTRouterBasicOperations tests the basic operations of the MQTT router.
func TestMQTTRouterBasicOperations(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT router
	router := createMQTTRouter(t)

	// Run the basic operations test
	testutils.TestProtocolBasicOperations(t, router)
}

// TestMQTTRouterContextAwareOperations tests the context-aware operations of the MQTT router.
func TestMQTTRouterContextAwareOperations(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT router
	router := createMQTTRouter(t)

	// Run the context-aware operations test
	testutils.TestProtocolContextAwareOperations(t, router)
}

// TestMQTTRouterErrorHandling tests the error handling of the MQTT router.
func TestMQTTRouterErrorHandling(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT router
	router := createMQTTRouter(t)

	// Run the error handling test
	testutils.TestProtocolErrorHandling(t, router)
}

// TestMQTTRouterConcurrentOperations tests the concurrent operations of the MQTT router.
func TestMQTTRouterConcurrentOperations(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT router
	router := createMQTTRouter(t)

	// Run the concurrent operations test
	testutils.TestProtocolConcurrentOperations(t, router)
}

// TestMQTTRouterConnect tests the Connect method of the MQTT router.
func TestMQTTRouterConnect(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT router
	router := createMQTTRouter(t)

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

// TestMQTTRouterDispatch tests the Dispatch method of the MQTT router.
func TestMQTTRouterDispatch(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT router
	router := createMQTTRouter(t)

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

// TestMQTTRouterSubscribe tests the Subscribe method of the MQTT router.
func TestMQTTRouterSubscribe(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT router
	router := createMQTTRouter(t)

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

// TestMQTTRouterUnsubscribe tests the Unsubscribe method of the MQTT router.
func TestMQTTRouterUnsubscribe(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT router
	router := createMQTTRouter(t)

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

// TestMQTTRouterAccept tests the Accept method of the MQTT router.
func TestMQTTRouterAccept(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT router
	router := createMQTTRouter(t)

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

// TestMQTTRouterEndToEnd tests the end-to-end functionality of the MQTT router.
func TestMQTTRouterEndToEnd(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Create a new MQTT router
	router := createMQTTRouter(t)

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

// TestMQTTRouterRegistration tests the registration of the MQTT router with the core registry.
func TestMQTTRouterRegistration(t *testing.T) {
	// Skip in short mode as this test requires an MQTT broker
	if testing.Short() {
		t.Skip("Skipping MQTT tests in short mode")
	}

	// Deploy a router of type MQTT
	router, err := core.DeployRouter(core.MQTT)
	require.NoError(t, err, "DeployRouter should not return an error")
	require.NotNil(t, router, "Deployed router should not be nil")

	// Check that the router is a MQTT Packet
	_, ok := router.(*Packet)
	assert.True(t, ok, "Deployed router should be a MQTT Packet")

	// Close the router
	err = router.Close()
	require.NoError(t, err, "Close should not return an error")
}
