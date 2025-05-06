package accept

import (
	"context"
	"fmt"
	"sync"

	"hybridpipe.io/core"
)

// Acceptor is a wrapper around a HybridPipe that provides a simplified interface for accepting messages.
type Acceptor struct {
	router core.HybridPipe
	pipes  map[string]struct{}
	mutex  sync.RWMutex
}

// NewAcceptor creates a new Acceptor.
func NewAcceptor(router core.HybridPipe) *Acceptor {
	return &Acceptor{
		router: router,
		pipes:  make(map[string]struct{}),
	}
}

// Accept registers a callback for the specified pipe.
func (a *Acceptor) Accept(pipe string, callback core.Process) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Check if already accepting from this pipe
	if _, ok := a.pipes[pipe]; ok {
		return fmt.Errorf("already accepting from pipe: %s", pipe)
	}

	// Subscribe to the pipe
	err := a.router.Subscribe(pipe, callback)
	if err != nil {
		return fmt.Errorf("failed to subscribe to pipe: %w", err)
	}

	// Mark as accepting from this pipe
	a.pipes[pipe] = struct{}{}

	return nil
}

// AcceptWithContext registers a callback for the specified pipe with context.
func (a *Acceptor) AcceptWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Check if the context is done
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with accept
	}

	// If the router supports context-aware operations, use them
	if ctxRouter, ok := a.router.(core.ContextAwareHybridPipe); ok {
		a.mutex.Lock()
		defer a.mutex.Unlock()

		// Check if already accepting from this pipe
		if _, ok := a.pipes[pipe]; ok {
			return fmt.Errorf("already accepting from pipe: %s", pipe)
		}

		// Subscribe to the pipe with context
		err := ctxRouter.SubscribeWithContext(ctx, pipe, callback)
		if err != nil {
			return fmt.Errorf("failed to subscribe to pipe with context: %w", err)
		}

		// Mark as accepting from this pipe
		a.pipes[pipe] = struct{}{}

		return nil
	}

	// Fall back to regular accept
	return a.Accept(pipe, callback)
}

// Stop stops accepting messages from the specified pipe.
func (a *Acceptor) Stop(pipe string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Check if accepting from this pipe
	if _, ok := a.pipes[pipe]; !ok {
		return fmt.Errorf("not accepting from pipe: %s", pipe)
	}

	// Unsubscribe from the pipe
	err := a.router.Unsubscribe(pipe)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from pipe: %w", err)
	}

	// Remove from accepting pipes
	delete(a.pipes, pipe)

	return nil
}

// StopAll stops accepting messages from all pipes.
func (a *Acceptor) StopAll() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Unsubscribe from all pipes
	for pipe := range a.pipes {
		err := a.router.Unsubscribe(pipe)
		if err != nil {
			return fmt.Errorf("failed to unsubscribe from pipe %s: %w", pipe, err)
		}
		delete(a.pipes, pipe)
	}

	return nil
}

// IsAccepting returns true if accepting messages from the specified pipe.
func (a *Acceptor) IsAccepting(pipe string) bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	_, ok := a.pipes[pipe]
	return ok
}

// GetPipes returns a list of pipes that are being accepted from.
func (a *Acceptor) GetPipes() []string {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	pipes := make([]string, 0, len(a.pipes))
	for pipe := range a.pipes {
		pipes = append(pipes, pipe)
	}

	return pipes
}
