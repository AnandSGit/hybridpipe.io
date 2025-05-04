// Package middleware provides middleware components for the HybridPipe system.
package middleware

import (
	"context"

	"hybridpipe.io/core"
)

// Middleware is a function that wraps a HybridPipe implementation.
type Middleware func(core.HybridPipe) core.HybridPipe

// Chain applies a series of middleware to a HybridPipe implementation.
func Chain(router core.HybridPipe, middleware ...Middleware) core.HybridPipe {
	// Apply each middleware in order
	for _, m := range middleware {
		router = m(router)
	}
	return router
}

// BaseMiddleware is a base struct for middleware implementations.
type BaseMiddleware struct {
	// Router is the wrapped HybridPipe implementation.
	Router core.HybridPipe
}

// Connect establishes a connection to the messaging system.
func (m *BaseMiddleware) Connect() error {
	return m.Router.Connect()
}

// Dispatch sends a message to the specified pipe.
func (m *BaseMiddleware) Dispatch(pipe string, data any) error {
	return m.Router.Dispatch(pipe, data)
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (m *BaseMiddleware) Subscribe(pipe string, callback core.Process) error {
	return m.Router.Subscribe(pipe, callback)
}

// Unsubscribe removes a subscription from the specified pipe.
func (m *BaseMiddleware) Unsubscribe(pipe string) error {
	return m.Router.Unsubscribe(pipe)
}

// Close terminates the connection to the messaging system.
func (m *BaseMiddleware) Close() error {
	return m.Router.Close()
}

// IsConnected returns true if the connection to the messaging system is active.
func (m *BaseMiddleware) IsConnected() bool {
	if connectable, ok := m.Router.(interface{ IsConnected() bool }); ok {
		return connectable.IsConnected()
	}
	return true // Assume connected if not checkable
}

// ContextAwareBaseMiddleware is a base struct for context-aware middleware implementations.
type ContextAwareBaseMiddleware struct {
	BaseMiddleware
	ContextRouter interface {
		DispatchWithContext(ctx context.Context, pipe string, data any) error
		SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error
	}
}

// NewContextAwareBaseMiddleware creates a new ContextAwareBaseMiddleware.
func NewContextAwareBaseMiddleware(router core.HybridPipe) *ContextAwareBaseMiddleware {
	base := &ContextAwareBaseMiddleware{
		BaseMiddleware: BaseMiddleware{
			Router: router,
		},
	}

	// If the router is context-aware, store it
	if ctxRouter, ok := router.(interface {
		DispatchWithContext(ctx context.Context, pipe string, data any) error
		SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error
	}); ok {
		base.ContextRouter = ctxRouter
	}

	return base
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (m *ContextAwareBaseMiddleware) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	if m.ContextRouter != nil {
		return m.ContextRouter.DispatchWithContext(ctx, pipe, data)
	}
	// Fall back to regular dispatch if the wrapped router is not context-aware
	return m.Router.Dispatch(pipe, data)
}

// SubscribeWithContext registers a callback with context for cancellation.
func (m *ContextAwareBaseMiddleware) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	if m.ContextRouter != nil {
		return m.ContextRouter.SubscribeWithContext(ctx, pipe, callback)
	}
	// Fall back to regular subscribe if the wrapped router is not context-aware
	return m.Router.Subscribe(pipe, callback)
}
