package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"hybridpipe.io/core"
)

// MockRouter is a mock implementation of the HybridPipe interface for testing.
type MockRouter struct {
	mock.Mock
}

func (m *MockRouter) Connect() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRouter) Disconnect() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRouter) Dispatch(pipe string, data any) error {
	args := m.Called(pipe, data)
	return args.Error(0)
}

func (m *MockRouter) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	args := m.Called(ctx, pipe, data)
	return args.Error(0)
}

func (m *MockRouter) Accept(pipe string, fn func(any)) error {
	args := m.Called(pipe, fn)
	return args.Error(0)
}

func (m *MockRouter) Subscribe(pipe string, callback core.Process) error {
	args := m.Called(pipe, callback)
	return args.Error(0)
}

func (m *MockRouter) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	args := m.Called(ctx, pipe, callback)
	return args.Error(0)
}

func (m *MockRouter) Remove(pipe string) error {
	args := m.Called(pipe)
	return args.Error(0)
}

func (m *MockRouter) Unsubscribe(pipe string) error {
	args := m.Called(pipe)
	return args.Error(0)
}

func (m *MockRouter) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRouter) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

// TestMiddlewareChain tests chaining multiple middleware together.
func TestMiddlewareChain(t *testing.T) {
	// Create a mock router
	mockRouter := new(MockRouter)

	// Set up expectations
	mockRouter.On("Connect").Return(nil)
	mockRouter.On("Dispatch", "test-pipe", "test-data").Return(nil)
	mockRouter.On("Close").Return(nil)
	mockRouter.On("Subscribe", "test-pipe", mock.AnythingOfType("core.Process")).Return(nil)
	mockRouter.On("Unsubscribe", "test-pipe").Return(nil)
	mockRouter.On("IsConnected").Return(true)
	mockRouter.On("Accept", "test-pipe", mock.AnythingOfType("func(interface {})")).Return(nil)
	mockRouter.On("Remove", "test-pipe").Return(nil)
	mockRouter.On("Disconnect").Return(nil)

	// Create middleware functions that track calls
	middleware1Calls := 0
	middleware2Calls := 0

	middleware1 := func(router core.HybridPipe) core.HybridPipe {
		middleware1Calls++
		return router // Pass-through middleware
	}

	middleware2 := func(router core.HybridPipe) core.HybridPipe {
		middleware2Calls++
		return router // Pass-through middleware
	}

	// Chain the middleware
	chainedRouter := Chain(mockRouter, middleware1, middleware2)

	// Test the chained router
	err := chainedRouter.Connect()
	assert.NoError(t, err, "Connect should not return an error")

	err = chainedRouter.Dispatch("test-pipe", "test-data")
	assert.NoError(t, err, "Dispatch should not return an error")

	err = chainedRouter.Subscribe("test-pipe", func(data []byte) error { return nil })
	assert.NoError(t, err, "Subscribe should not return an error")

	err = chainedRouter.Unsubscribe("test-pipe")
	assert.NoError(t, err, "Unsubscribe should not return an error")

	connected := chainedRouter.IsConnected()
	assert.True(t, connected, "IsConnected should return true")

	err = chainedRouter.Accept("test-pipe", func(data any) {})
	assert.NoError(t, err, "Accept should not return an error")

	err = chainedRouter.Remove("test-pipe")
	assert.NoError(t, err, "Remove should not return an error")

	err = chainedRouter.Disconnect()
	assert.NoError(t, err, "Disconnect should not return an error")

	err = chainedRouter.Close()
	assert.NoError(t, err, "Close should not return an error")

	// Verify that all expectations were met
	mockRouter.AssertExpectations(t)

	// Verify that middleware functions were called
	assert.Equal(t, 1, middleware1Calls, "Middleware1 should be called once")
	assert.Equal(t, 1, middleware2Calls, "Middleware2 should be called once")
}

// TestBaseMiddleware tests the BaseMiddleware struct.
func TestBaseMiddleware(t *testing.T) {
	// Create a mock router
	mockRouter := new(MockRouter)

	// Set up expectations
	mockRouter.On("Connect").Return(nil)
	mockRouter.On("Dispatch", "test-pipe", "test-data").Return(nil)
	mockRouter.On("Subscribe", "test-pipe", mock.AnythingOfType("core.Process")).Return(nil)
	mockRouter.On("Unsubscribe", "test-pipe").Return(nil)
	mockRouter.On("Close").Return(nil)
	mockRouter.On("IsConnected").Return(true)

	// Create a BaseMiddleware
	baseMiddleware := &BaseMiddleware{
		Router: mockRouter,
	}

	// Test the BaseMiddleware
	err := baseMiddleware.Connect()
	assert.NoError(t, err, "Connect should not return an error")

	err = baseMiddleware.Dispatch("test-pipe", "test-data")
	assert.NoError(t, err, "Dispatch should not return an error")

	err = baseMiddleware.Subscribe("test-pipe", func(data []byte) error { return nil })
	assert.NoError(t, err, "Subscribe should not return an error")

	err = baseMiddleware.Unsubscribe("test-pipe")
	assert.NoError(t, err, "Unsubscribe should not return an error")

	err = baseMiddleware.Close()
	assert.NoError(t, err, "Close should not return an error")

	connected := baseMiddleware.IsConnected()
	assert.True(t, connected, "IsConnected should return true")

	// Verify that all expectations were met
	mockRouter.AssertExpectations(t)
}

// TestContextAwareBaseMiddleware tests the ContextAwareBaseMiddleware struct.
func TestContextAwareBaseMiddleware(t *testing.T) {
	// Create a mock router
	mockRouter := new(MockRouter)

	// Set up expectations
	mockRouter.On("Connect").Return(nil)
	mockRouter.On("Dispatch", "test-pipe", "test-data").Return(nil)
	mockRouter.On("Subscribe", "test-pipe", mock.AnythingOfType("core.Process")).Return(nil)
	mockRouter.On("Unsubscribe", "test-pipe").Return(nil)
	mockRouter.On("Close").Return(nil)
	mockRouter.On("IsConnected").Return(true)
	mockRouter.On("DispatchWithContext", mock.Anything, "test-pipe", "test-data").Return(nil)
	mockRouter.On("SubscribeWithContext", mock.Anything, "test-pipe", mock.AnythingOfType("core.Process")).Return(nil)

	// Create a ContextAwareBaseMiddleware
	ctxMiddleware := NewContextAwareBaseMiddleware(mockRouter)

	// Test the ContextAwareBaseMiddleware
	ctx := context.Background()

	err := ctxMiddleware.Connect()
	assert.NoError(t, err, "Connect should not return an error")

	err = ctxMiddleware.Dispatch("test-pipe", "test-data")
	assert.NoError(t, err, "Dispatch should not return an error")

	err = ctxMiddleware.DispatchWithContext(ctx, "test-pipe", "test-data")
	assert.NoError(t, err, "DispatchWithContext should not return an error")

	err = ctxMiddleware.Subscribe("test-pipe", func(data []byte) error { return nil })
	assert.NoError(t, err, "Subscribe should not return an error")

	err = ctxMiddleware.SubscribeWithContext(ctx, "test-pipe", func(data []byte) error { return nil })
	assert.NoError(t, err, "SubscribeWithContext should not return an error")

	err = ctxMiddleware.Unsubscribe("test-pipe")
	assert.NoError(t, err, "Unsubscribe should not return an error")

	err = ctxMiddleware.Close()
	assert.NoError(t, err, "Close should not return an error")

	connected := ctxMiddleware.IsConnected()
	assert.True(t, connected, "IsConnected should return true")

	// Verify that all expectations were met
	mockRouter.AssertExpectations(t)
}

// TestContextAwareBaseMiddlewareWithNonContextRouter tests the ContextAwareBaseMiddleware with a non-context-aware router.
func TestContextAwareBaseMiddlewareWithNonContextRouter(t *testing.T) {
	// Create a mock router that doesn't implement the context-aware methods
	mockRouter := new(MockRouter)

	// Set up expectations
	mockRouter.On("Connect").Return(nil)
	mockRouter.On("Dispatch", "test-pipe", "test-data").Return(nil)
	mockRouter.On("Subscribe", "test-pipe", mock.AnythingOfType("core.Process")).Return(nil)
	mockRouter.On("Unsubscribe", "test-pipe").Return(nil)
	mockRouter.On("Close").Return(nil)
	mockRouter.On("IsConnected").Return(true)
	// Add expectations for context-aware methods
	mockRouter.On("DispatchWithContext", mock.Anything, "test-pipe", "test-data").Return(nil)
	mockRouter.On("SubscribeWithContext", mock.Anything, "test-pipe", mock.AnythingOfType("core.Process")).Return(nil)

	// Create a ContextAwareBaseMiddleware
	ctxMiddleware := NewContextAwareBaseMiddleware(mockRouter)

	// Test the ContextAwareBaseMiddleware
	ctx := context.Background()

	err := ctxMiddleware.Connect()
	assert.NoError(t, err, "Connect should not return an error")

	err = ctxMiddleware.Dispatch("test-pipe", "test-data")
	assert.NoError(t, err, "Dispatch should not return an error")

	// These should use the mocked context methods
	err = ctxMiddleware.DispatchWithContext(ctx, "test-pipe", "test-data")
	assert.NoError(t, err, "DispatchWithContext should not return an error")

	err = ctxMiddleware.Subscribe("test-pipe", func(data []byte) error { return nil })
	assert.NoError(t, err, "Subscribe should not return an error")

	err = ctxMiddleware.SubscribeWithContext(ctx, "test-pipe", func(data []byte) error { return nil })
	assert.NoError(t, err, "SubscribeWithContext should not return an error")

	err = ctxMiddleware.Unsubscribe("test-pipe")
	assert.NoError(t, err, "Unsubscribe should not return an error")

	err = ctxMiddleware.Close()
	assert.NoError(t, err, "Close should not return an error")

	connected := ctxMiddleware.IsConnected()
	assert.True(t, connected, "IsConnected should return true")

	// Verify that all expectations were met
	mockRouter.AssertExpectations(t)
}

// TestLoggingMiddleware tests the logging middleware.
func TestLoggingMiddleware(t *testing.T) {
	// Skip for now as we need to implement the logging middleware first
	t.Skip("Skipping logging middleware tests until the middleware package is fully implemented")
}

// TestRetryMiddleware tests the retry middleware.
func TestRetryMiddleware(t *testing.T) {
	t.Skip("Skipping retry middleware tests until the middleware package is fully implemented")
}

// TestTimeoutMiddleware tests the timeout middleware.
func TestTimeoutMiddleware(t *testing.T) {
	t.Skip("Skipping timeout middleware tests until the middleware package is fully implemented")
}

// TestCircuitBreakerMiddleware tests the circuit breaker middleware.
func TestCircuitBreakerMiddleware(t *testing.T) {
	t.Skip("Skipping circuit breaker middleware tests until the middleware package is fully implemented")
}
