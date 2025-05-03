// Package core provides the fundamental interfaces and functionality for HybridPipe.
package core

import (
	"context"
	"sync"
)

// Process defines the callback function that should be called when a client receives a message.
type Process func(data []byte) error

// HybridPipe defines the core interface for all messaging implementations.
// Any messaging system implementation must satisfy this interface.
type HybridPipe interface {
	// Connect establishes a connection to the messaging system.
	Connect() error

	// Disconnect closes the connection to the messaging system.
	Disconnect() error

	// Dispatch sends a message to the specified pipe.
	Dispatch(pipe string, data any) error

	// DispatchWithContext sends a message with context for cancellation and timeouts.
	DispatchWithContext(ctx context.Context, pipe string, data any) error

	// Accept subscribes to messages from the specified pipe and processes them with the provided function.
	// This is an alias for Subscribe for backward compatibility.
	Accept(pipe string, fn func(any)) error

	// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
	Subscribe(pipe string, callback Process) error

	// SubscribeWithContext registers a callback with context for cancellation.
	SubscribeWithContext(ctx context.Context, pipe string, callback Process) error

	// Remove unsubscribes from the specified pipe.
	// This is an alias for Unsubscribe for backward compatibility.
	Remove(pipe string) error

	// Unsubscribe removes a subscription from the specified pipe.
	Unsubscribe(pipe string) error

	// Close terminates the connection to the messaging system.
	// This is an alias for Disconnect for backward compatibility.
	Close() error

	// IsConnected returns true if the connection to the messaging system is active.
	IsConnected() bool
}

// RouterFactory is a function that creates a new instance of a HybridPipe implementation.
type RouterFactory func() HybridPipe

// registeredFactories maps broker types to their factory functions.
var registeredFactories = make(map[int]RouterFactory)
var factoryMutex sync.RWMutex

// RegisterRouterFactory registers a factory function for a specific broker type.
func RegisterRouterFactory(brokerType int, factory RouterFactory) {
	factoryMutex.Lock()
	defer factoryMutex.Unlock()
	registeredFactories[brokerType] = factory
}
