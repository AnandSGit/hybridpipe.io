package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRouter is a mock implementation of the HybridPipe interface for testing.
type MockRouter struct {
	connected bool
}

func (m *MockRouter) Connect() error {
	m.connected = true
	return nil
}

func (m *MockRouter) Disconnect() error {
	m.connected = false
	return nil
}

func (m *MockRouter) Dispatch(pipe string, data any) error {
	return nil
}

func (m *MockRouter) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	return nil
}

func (m *MockRouter) Accept(pipe string, fn func(any)) error {
	return nil
}

func (m *MockRouter) Subscribe(pipe string, callback Process) error {
	return nil
}

func (m *MockRouter) SubscribeWithContext(ctx context.Context, pipe string, callback Process) error {
	return nil
}

func (m *MockRouter) Remove(pipe string) error {
	return nil
}

func (m *MockRouter) Unsubscribe(pipe string) error {
	return nil
}

func (m *MockRouter) Close() error {
	return m.Disconnect()
}

func (m *MockRouter) IsConnected() bool {
	return m.connected
}

// MockRouterFactory is a factory function for creating MockRouter instances.
func MockRouterFactory() HybridPipe {
	return &MockRouter{}
}

func TestRegisterRouterFactory(t *testing.T) {
	// Register a mock router factory
	const testBrokerType = 9999
	RegisterRouterFactory(testBrokerType, MockRouterFactory)

	// Verify that the factory was registered
	factoryMutex.RLock()
	factory, exists := registeredFactories[testBrokerType]
	factoryMutex.RUnlock()

	assert.True(t, exists, "Factory should be registered")
	assert.NotNil(t, factory, "Factory should not be nil")

	// Create a router using the factory
	router := factory()
	assert.NotNil(t, router, "Router should not be nil")
	assert.IsType(t, &MockRouter{}, router, "Router should be of type *MockRouter")
}

func TestDeployRouter(t *testing.T) {
	// Register a mock router factory
	const testBrokerType = 9998
	RegisterRouterFactory(testBrokerType, MockRouterFactory)

	// Deploy a router
	router, err := DeployRouter(testBrokerType)
	require.NoError(t, err, "DeployRouter should not return an error")
	require.NotNil(t, router, "Router should not be nil")
	assert.IsType(t, &MockRouter{}, router, "Router should be of type *MockRouter")

	// Verify that the router is connected
	assert.True(t, router.IsConnected(), "Router should be connected")

	// Test deploying with an unregistered broker type
	_, err = DeployRouter(9997)
	assert.Error(t, err, "DeployRouter should return an error for unregistered broker type")
}

// We'll skip this test as DeployRouterWithConfig is not implemented in the current version
func TestDeployRouterWithConfigSkipped(t *testing.T) {
	t.Skip("Skipping DeployRouterWithConfig test as it's not implemented in the current version")
}
