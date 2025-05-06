package core

import (
	"context"
)

// ContextAwareHybridPipe extends the HybridPipe interface with context-aware operations.
type ContextAwareHybridPipe interface {
	HybridPipe

	// DispatchWithContext sends a message with context for cancellation and timeouts.
	DispatchWithContext(ctx context.Context, pipe string, data any) error

	// SubscribeWithContext registers a callback with context for cancellation.
	SubscribeWithContext(ctx context.Context, pipe string, callback Process) error
}
