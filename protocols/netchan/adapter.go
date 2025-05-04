// Package netchan provides a NetChan implementation for the HybridPipe system.
package netchan

import (
	"context"
	"fmt"
	"log"
	"sync"

	"hybridpipe.io/core"
)

// NetChanAdapter adapts the NetChan library to the HybridPipe interface.
type NetChanAdapter struct {
	// router is the underlying NetChan router.
	router *Packet
	// sendChans is a map of send channels.
	sendChans map[string]chan interface{}
	// recvChans is a map of receive channels.
	recvChans map[string]chan interface{}
	// handlers is a map of message handlers.
	handlers map[string]context.CancelFunc
	// ctx is the context for the adapter.
	ctx context.Context
	// cancel is the cancel function for the context.
	cancel context.CancelFunc
	// mutex protects the maps.
	mutex sync.RWMutex
}

// NewNetChanAdapter creates a new NetChanAdapter.
func NewNetChanAdapter() (*NetChanAdapter, error) {
	// Create a new NetChan router
	config := &core.NetChanConfig{
		ChannelBufferSize: 10,
		SendTimeout:       1000,
	}
	router := New(config)

	// Connect to the NetChan system
	if err := router.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to NetChan system: %w", err)
	}

	// Create a context with cancel function
	ctx, cancel := context.WithCancel(context.Background())

	return &NetChanAdapter{
		router:    router,
		sendChans: make(map[string]chan interface{}),
		recvChans: make(map[string]chan interface{}),
		handlers:  make(map[string]context.CancelFunc),
		ctx:       ctx,
		cancel:    cancel,
		mutex:     sync.RWMutex{},
	}, nil
}

// Connect establishes a connection to the NetChan system.
func (a *NetChanAdapter) Connect() error {
	// Already connected
	return nil
}

// Dispatch sends a message to the specified pipe.
func (a *NetChanAdapter) Dispatch(pipe string, data any) error {
	// Get or create the send channel
	sendChan, err := a.getOrCreateSendChan(pipe)
	if err != nil {
		return fmt.Errorf("failed to get or create send channel: %w", err)
	}

	// Send the message
	select {
	case sendChan <- data:
		return nil
	case <-a.ctx.Done():
		return fmt.Errorf("context canceled")
	}
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (a *NetChanAdapter) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	// Get or create the send channel
	sendChan, err := a.getOrCreateSendChan(pipe)
	if err != nil {
		return fmt.Errorf("failed to get or create send channel: %w", err)
	}

	// Send the message
	select {
	case sendChan <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-a.ctx.Done():
		return fmt.Errorf("adapter context canceled")
	}
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (a *NetChanAdapter) Subscribe(pipe string, callback core.Process) error {
	// Lock for thread safety
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Check if already subscribed
	if _, exists := a.handlers[pipe]; exists {
		return fmt.Errorf("already subscribed to pipe %s", pipe)
	}

	// Get or create the receive channel
	recvChan, err := a.getOrCreateRecvChanLocked(pipe)
	if err != nil {
		return fmt.Errorf("failed to get or create receive channel: %w", err)
	}

	// Create a context with cancel function for the handler
	ctx, cancel := context.WithCancel(a.ctx)
	a.handlers[pipe] = cancel

	// Start a goroutine to handle messages
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in NetChan handler for pipe %s: %v", pipe, r)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				log.Printf("Stopping NetChan handler for pipe %s", pipe)
				return
			case msg, ok := <-recvChan:
				if !ok {
					log.Printf("NetChan receive channel closed for pipe %s", pipe)
					return
				}

				// Encode the message
				encoded, err := core.Encode(msg)
				if err != nil {
					log.Printf("Failed to encode message from pipe %s: %v", pipe, err)
					continue
				}

				// Call the callback
				if err := callback(encoded); err != nil {
					log.Printf("Error processing message from pipe %s: %v", pipe, err)
				}
			}
		}
	}()

	log.Printf("Subscribed to NetChan pipe %s", pipe)
	return nil
}

// SubscribeWithContext registers a callback with context for cancellation.
func (a *NetChanAdapter) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Lock for thread safety
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Check if already subscribed
	if _, exists := a.handlers[pipe]; exists {
		return fmt.Errorf("already subscribed to pipe %s", pipe)
	}

	// Get or create the receive channel
	recvChan, err := a.getOrCreateRecvChanLocked(pipe)
	if err != nil {
		return fmt.Errorf("failed to get or create receive channel: %w", err)
	}

	// Create a context with cancel function for the handler
	handlerCtx, cancel := context.WithCancel(ctx)
	a.handlers[pipe] = cancel

	// Start a goroutine to handle messages
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in NetChan handler for pipe %s: %v", pipe, r)
			}
		}()

		for {
			select {
			case <-handlerCtx.Done():
				log.Printf("Stopping NetChan handler for pipe %s", pipe)
				return
			case <-a.ctx.Done():
				log.Printf("Stopping NetChan handler for pipe %s (adapter context canceled)", pipe)
				return
			case msg, ok := <-recvChan:
				if !ok {
					log.Printf("NetChan receive channel closed for pipe %s", pipe)
					return
				}

				// Encode the message
				encoded, err := core.Encode(msg)
				if err != nil {
					log.Printf("Failed to encode message from pipe %s: %v", pipe, err)
					continue
				}

				// Call the callback
				if err := callback(encoded); err != nil {
					log.Printf("Error processing message from pipe %s: %v", pipe, err)
				}
			}
		}
	}()

	log.Printf("Subscribed to NetChan pipe %s with context", pipe)
	return nil
}

// Unsubscribe removes a subscription from the specified pipe.
func (a *NetChanAdapter) Unsubscribe(pipe string) error {
	// Lock for thread safety
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Check if the pipe is subscribed
	cancel, exists := a.handlers[pipe]
	if !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}

	// Cancel the handler
	cancel()
	delete(a.handlers, pipe)

	log.Printf("Unsubscribed from NetChan pipe %s", pipe)
	return nil
}

// Close terminates the connection to the NetChan system.
func (a *NetChanAdapter) Close() error {
	// Lock for thread safety
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Cancel all handlers
	a.cancel()

	// Close the router
	a.router.Close()

	log.Printf("Closed NetChan adapter")
	return nil
}

// getOrCreateSendChan gets or creates a send channel for the specified pipe.
func (a *NetChanAdapter) getOrCreateSendChan(pipe string) (chan interface{}, error) {
	// Lock for thread safety
	a.mutex.Lock()
	defer a.mutex.Unlock()

	return a.getOrCreateSendChanLocked(pipe)
}

// getOrCreateSendChanLocked gets or creates a send channel for the specified pipe (with lock already held).
func (a *NetChanAdapter) getOrCreateSendChanLocked(pipe string) (chan interface{}, error) {
	// Check if the channel already exists
	if ch, exists := a.sendChans[pipe]; exists {
		return ch, nil
	}

	// Create a new channel
	ch := make(chan interface{}, 10)
	a.sendChans[pipe] = ch

	// Start a goroutine to forward messages to the router
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in NetChan sender for pipe %s: %v", pipe, r)
			}
		}()

		for {
			select {
			case <-a.ctx.Done():
				log.Printf("Stopping NetChan sender for pipe %s", pipe)
				return
			case msg, ok := <-ch:
				if !ok {
					log.Printf("NetChan send channel closed for pipe %s", pipe)
					return
				}

				// Dispatch the message to the router
				if err := a.router.Dispatch(pipe, msg); err != nil {
					log.Printf("Error dispatching message to pipe %s: %v", pipe, err)
				}
			}
		}
	}()

	return ch, nil
}

// getOrCreateRecvChan gets or creates a receive channel for the specified pipe.
func (a *NetChanAdapter) getOrCreateRecvChan(pipe string) (chan interface{}, error) {
	// Lock for thread safety
	a.mutex.Lock()
	defer a.mutex.Unlock()

	return a.getOrCreateRecvChanLocked(pipe)
}

// getOrCreateRecvChanLocked gets or creates a receive channel for the specified pipe (with lock already held).
func (a *NetChanAdapter) getOrCreateRecvChanLocked(pipe string) (chan interface{}, error) {
	// Check if the channel already exists
	if ch, exists := a.recvChans[pipe]; exists {
		return ch, nil
	}

	// Create a new channel
	ch := make(chan interface{}, 10)
	a.recvChans[pipe] = ch

	// Subscribe to the router
	err := a.router.Subscribe(pipe, func(data []byte) error {
		// Decode the message
		var msg interface{}
		if err := core.Decode(data, &msg); err != nil {
			return fmt.Errorf("failed to decode message: %w", err)
		}

		// Send the message to the channel
		select {
		case ch <- msg:
			return nil
		case <-a.ctx.Done():
			return fmt.Errorf("context canceled")
		}
	})

	if err != nil {
		delete(a.recvChans, pipe)
		return nil, fmt.Errorf("failed to subscribe to router: %w", err)
	}

	return ch, nil
}

// IsConnected returns true if the connection to the NetChan system is active.
func (a *NetChanAdapter) IsConnected() bool {
	return a.router.IsConnected()
}
