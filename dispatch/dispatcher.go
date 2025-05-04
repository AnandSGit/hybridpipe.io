package dispatch

import (
	"context"
	"fmt"

	"hybridpipe.io/core"
)

// Dispatcher is a wrapper around a HybridPipe that provides a simplified interface for dispatching messages.
type Dispatcher struct {
	router core.HybridPipe
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(router core.HybridPipe) *Dispatcher {
	return &Dispatcher{
		router: router,
	}
}

// Dispatch sends a message to the specified pipe.
func (d *Dispatcher) Dispatch(pipe string, data interface{}) error {
	return d.router.Dispatch(pipe, data)
}

// DispatchWithContext sends a message to the specified pipe with context.
func (d *Dispatcher) DispatchWithContext(ctx context.Context, pipe string, data interface{}) error {
	// Check if the context is done
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with dispatch
	}

	// If the router supports context-aware operations, use them
	if ctxRouter, ok := d.router.(core.ContextAwareHybridPipe); ok {
		return ctxRouter.DispatchWithContext(ctx, pipe, data)
	}

	// Fall back to regular dispatch
	return d.Dispatch(pipe, data)
}

// DispatchBatch sends multiple messages to the specified pipe.
func (d *Dispatcher) DispatchBatch(pipe string, data []interface{}) error {
	for _, item := range data {
		if err := d.Dispatch(pipe, item); err != nil {
			return fmt.Errorf("failed to dispatch batch item: %w", err)
		}
	}
	return nil
}

// DispatchBatchWithContext sends multiple messages to the specified pipe with context.
func (d *Dispatcher) DispatchBatchWithContext(ctx context.Context, pipe string, data []interface{}) error {
	for _, item := range data {
		// Check if the context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue with dispatch
		}

		if err := d.DispatchWithContext(ctx, pipe, item); err != nil {
			return fmt.Errorf("failed to dispatch batch item with context: %w", err)
		}
	}
	return nil
}

// GetRouter returns the underlying router.
func (d *Dispatcher) GetRouter() core.HybridPipe {
	return d.router
}
