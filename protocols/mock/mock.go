// Package mock provides an in-memory implementation of the HybridPipe interface for testing.
package mock

import (
	"context"
	"sync"

	"hybridpipe.io/core"
)

func init() {
	// Register the mock router factory
	core.RegisterRouterFactory(core.MOCK, func() core.HybridPipe {
		return NewMockRouter()
	})
}

// MockRouter implements the HybridPipe interface for testing.
type MockRouter struct {
	connected     bool
	subscriptions map[string][]core.Process
	messages      map[string][][]byte
	mutex         sync.RWMutex
}

// NewMockRouter creates a new MockRouter.
func NewMockRouter() *MockRouter {
	return &MockRouter{
		connected:     false,
		subscriptions: make(map[string][]core.Process),
		messages:      make(map[string][][]byte),
	}
}

// Connect establishes a connection to the messaging system.
func (r *MockRouter) Connect() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.connected = true
	return nil
}

// Disconnect closes the connection to the messaging system.
func (r *MockRouter) Disconnect() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.connected = false
	return nil
}

// Dispatch sends a message to the specified pipe.
func (r *MockRouter) Dispatch(pipe string, data any) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return core.ErrNotConnected
	}

	// Convert data to bytes
	var bytes []byte
	var ok bool
	if bytes, ok = data.([]byte); !ok {
		var err error
		bytes, err = core.Encode(data)
		if err != nil {
			return err
		}
	}

	// Store the message
	if _, exists := r.messages[pipe]; !exists {
		r.messages[pipe] = make([][]byte, 0)
	}
	r.messages[pipe] = append(r.messages[pipe], bytes)

	// Notify subscribers
	if subs, exists := r.subscriptions[pipe]; exists {
		for _, sub := range subs {
			go sub(bytes)
		}
	}

	return nil
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (r *MockRouter) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return r.Dispatch(pipe, data)
	}
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (r *MockRouter) Accept(pipe string, fn func(any)) error {
	// Check if connected without holding the lock for the entire operation
	r.mutex.RLock()
	if !r.connected {
		r.mutex.RUnlock()
		return core.ErrNotConnected
	}
	r.mutex.RUnlock()

	// Validate the callback
	if fn == nil {
		return core.ErrInvalidCallback
	}

	// Create a wrapper that converts the Process function to the expected type
	wrapper := func(data []byte) error {
		fn(data)
		return nil
	}

	// Call Subscribe without holding the lock
	return r.Subscribe(pipe, wrapper)
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (r *MockRouter) Subscribe(pipe string, callback core.Process) error {
	// Validate the callback
	if callback == nil {
		return core.ErrInvalidCallback
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return core.ErrNotConnected
	}

	// Add the subscription
	if _, exists := r.subscriptions[pipe]; !exists {
		r.subscriptions[pipe] = make([]core.Process, 0)
	}
	r.subscriptions[pipe] = append(r.subscriptions[pipe], callback)

	// Process any existing messages
	if messages, exists := r.messages[pipe]; exists {
		for _, msg := range messages {
			go callback(msg)
		}
	}

	return nil
}

// SubscribeWithContext registers a callback with context for cancellation.
func (r *MockRouter) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return r.Subscribe(pipe, callback)
	}
}

// Remove unsubscribes from the specified pipe.
func (r *MockRouter) Remove(pipe string) error {
	return r.Unsubscribe(pipe)
}

// Unsubscribe removes a subscription from the specified pipe.
func (r *MockRouter) Unsubscribe(pipe string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return core.ErrNotConnected
	}

	delete(r.subscriptions, pipe)
	return nil
}

// Close terminates the connection to the messaging system.
func (r *MockRouter) Close() error {
	return r.Disconnect()
}

// IsConnected returns true if the connection to the messaging system is active.
func (r *MockRouter) IsConnected() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.connected
}
