package hybridpipe

import (
	"context"
	"fmt"
	"sync"
)

// NetChanAdapter provides a Go channel-like interface for HybridPipe's NetChan implementation.
// It allows using native Go channel syntax with HybridPipe.
type NetChanAdapter struct {
	router     HybridPipe
	sendChans  map[string]chan interface{}
	recvChans  map[string]chan interface{}
	handlers   map[string]context.CancelFunc
	bufferSize int
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewNetChanAdapter creates a new adapter for using Go channel syntax with HybridPipe.
func NewNetChanAdapter(bufferSize int) (*NetChanAdapter, error) {
	// Create a new NetChanPacket
	router := &NetChanPacket{}

	// Connect to the NetChan system
	if err := router.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to NetChan system: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &NetChanAdapter{
		router:     router,
		sendChans:  make(map[string]chan interface{}),
		recvChans:  make(map[string]chan interface{}),
		handlers:   make(map[string]context.CancelFunc),
		bufferSize: bufferSize,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// GetSendChan returns a channel that can be used to send values to the specified pipe.
func (a *NetChanAdapter) GetSendChan(pipe string) (chan<- interface{}, error) {
	a.mutex.RLock()
	ch, exists := a.sendChans[pipe]
	a.mutex.RUnlock()

	if exists {
		return ch, nil
	}

	// Create a new channel
	ch = make(chan interface{}, a.bufferSize)

	// Create a context for this pipe
	pipeCtx, pipeCancel := context.WithCancel(a.ctx)

	// Start a goroutine to forward messages from the channel to the pipe
	go func() {
		defer pipeCancel()

		for {
			select {
			case <-pipeCtx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}

				// Forward to the pipe
				if err := a.router.Dispatch(pipe, msg); err != nil {
					// Log error but continue
					fmt.Printf("Error dispatching to pipe %s: %v\n", pipe, err)
				}
			}
		}
	}()

	// Store the channel and cancel function
	a.mutex.Lock()
	a.sendChans[pipe] = ch
	a.handlers[pipe] = pipeCancel
	a.mutex.Unlock()

	return ch, nil
}

// GetRecvChan returns a channel that can be used to receive values from the specified pipe.
func (a *NetChanAdapter) GetRecvChan(pipe string) (<-chan interface{}, error) {
	a.mutex.RLock()
	ch, exists := a.recvChans[pipe]
	a.mutex.RUnlock()

	if exists {
		return ch, nil
	}

	// Create a new channel
	ch = make(chan interface{}, a.bufferSize)

	// Create a context for this pipe
	pipeCtx, pipeCancel := context.WithCancel(a.ctx)

	// Subscribe to the pipe
	err := a.router.Accept(pipe, func(data interface{}) {
		// Check if the context is done
		select {
		case <-pipeCtx.Done():
			return
		default:
			// Forward to the channel
			select {
			case <-pipeCtx.Done():
				return
			case ch <- data:
				// Successfully forwarded
			}
		}
	})

	if err != nil {
		pipeCancel()
		return nil, fmt.Errorf("failed to accept from pipe %s: %w", pipe, err)
	}

	// Store the channel and cancel function
	a.mutex.Lock()
	a.recvChans[pipe] = ch
	if _, exists := a.handlers[pipe]; !exists {
		a.handlers[pipe] = pipeCancel
	}
	a.mutex.Unlock()

	return ch, nil
}

// RemovePipe removes a pipe from the adapter.
func (a *NetChanAdapter) RemovePipe(pipe string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Get the cancel function
	cancel, exists := a.handlers[pipe]
	if !exists {
		return fmt.Errorf("pipe %s not found", pipe)
	}

	// Cancel the context
	cancel()

	// Remove the pipe from the router
	if err := a.router.Remove(pipe); err != nil {
		return fmt.Errorf("failed to remove pipe %s: %w", pipe, err)
	}

	// Close the send channel if it exists
	if sendCh, exists := a.sendChans[pipe]; exists {
		close(sendCh)
		delete(a.sendChans, pipe)
	}

	// Delete the receive channel if it exists
	if _, exists := a.recvChans[pipe]; exists {
		delete(a.recvChans, pipe)
	}

	// Delete the handler
	delete(a.handlers, pipe)

	return nil
}

// Close closes the adapter and all associated channels.
func (a *NetChanAdapter) Close() error {
	// Signal all goroutines to stop
	a.cancel()

	// Lock for thread safety
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Close all send channels
	for pipe, ch := range a.sendChans {
		close(ch)
		delete(a.sendChans, pipe)
	}

	// Delete all receive channels
	for pipe := range a.recvChans {
		delete(a.recvChans, pipe)
	}

	// Delete all handlers
	for pipe := range a.handlers {
		delete(a.handlers, pipe)
	}

	// Close the router
	a.router.Close()

	return nil
}

// GetRouter returns the underlying HybridPipe router.
func (a *NetChanAdapter) GetRouter() HybridPipe {
	return a.router
}
