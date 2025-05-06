// Package netchan provides an implementation of the HybridPipe interface for NetChan messaging system.
package netchan

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"hybridpipe.io/core"
)

// MessageStats tracks message statistics for the NetChan system.
type MessageStats struct {
	SentCount     int64
	ReceivedCount int64
	ErrorCount    int64
	mutex         sync.RWMutex
}

// NewMessageStats creates a new MessageStats instance.
func NewMessageStats() *MessageStats {
	return &MessageStats{}
}

// IncrementSent increments the sent message count.
func (ms *MessageStats) IncrementSent() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.SentCount++
}

// IncrementReceived increments the received message count.
func (ms *MessageStats) IncrementReceived() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.ReceivedCount++
}

// IncrementErrors increments the error count.
func (ms *MessageStats) IncrementErrors() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.ErrorCount++
}

// GetStats returns the current message statistics.
func (ms *MessageStats) GetStats() (int64, int64, int64) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	return ms.SentCount, ms.ReceivedCount, ms.ErrorCount
}

// Packet implements the HybridPipe interface for NetChan messaging system.
type Packet struct {
	// channels maps pipe names to their channels
	channels map[string]chan any
	// handlers maps pipe names to their message handlers
	handlers map[string]core.Process
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// connected indicates if the NetChan is connected
	connected bool
	// stopChans maps pipe names to their stop channels
	stopChans map[string]chan struct{}
	// ctx is the context for the NetChan system
	ctx context.Context
	// cancel is the cancel function for the context
	cancel context.CancelFunc
	// stats tracks message statistics
	stats *MessageStats
	// config holds the NetChan configuration
	config *core.NetChanConfig
}

// New creates a new NetChan packet with the specified configuration.
func New(config *core.NetChanConfig) *Packet {
	ctx, cancel := context.WithCancel(context.Background())
	return &Packet{
		channels:  make(map[string]chan any),
		handlers:  make(map[string]core.Process),
		stopChans: make(map[string]chan struct{}),
		ctx:       ctx,
		cancel:    cancel,
		stats:     NewMessageStats(),
		config:    config,
	}
}

// Connect establishes a connection to the NetChan system.
// For NetChan, this just initializes the internal structures.
func (np *Packet) Connect() error {
	// Get configuration if not already set
	if np.config == nil {
		config, err := core.GetProtocolConfig(core.NETCHAN)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		netChanConfig := config.(core.NetChanConfig)
		np.config = &netChanConfig
	}

	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Initialize the maps if they don't exist
	if np.channels == nil {
		np.channels = make(map[string]chan any)
	}
	if np.handlers == nil {
		np.handlers = make(map[string]core.Process)
	}
	if np.stopChans == nil {
		np.stopChans = make(map[string]chan struct{})
	}

	// Create a context for the NetChan system
	if np.ctx == nil {
		np.ctx, np.cancel = context.WithCancel(context.Background())
	}

	// Initialize message statistics
	if np.stats == nil {
		np.stats = NewMessageStats()
	}

	np.connected = true
	log.Printf("NetChan system initialized")
	return nil
}

// Dispatch sends a message to the specified pipe.
func (np *Packet) Dispatch(pipe string, data any) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if !np.connected {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	np.mutex.RUnlock()

	// Get or create channel for the pipe
	np.mutex.Lock()
	ch, exists := np.channels[pipe]
	if !exists {
		bufferSize := np.config.ChannelBufferSize
		if bufferSize <= 0 {
			bufferSize = 10 // Default buffer size
		}
		ch = make(chan any, bufferSize)
		np.channels[pipe] = ch
	}
	np.mutex.Unlock()

	// Send the message with a timeout
	timeout := time.Duration(np.config.SendTimeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 1000 * time.Millisecond // Default timeout
	}

	select {
	case ch <- data:
		np.stats.IncrementSent()
		return nil
	case <-time.After(timeout):
		np.stats.IncrementErrors()
		return fmt.Errorf("timeout sending message to pipe %s", pipe)
	case <-np.ctx.Done():
		return fmt.Errorf("NetChan system is shutting down")
	}
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (np *Packet) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if !np.connected {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	np.mutex.RUnlock()

	// Get or create channel for the pipe
	np.mutex.Lock()
	ch, exists := np.channels[pipe]
	if !exists {
		bufferSize := np.config.ChannelBufferSize
		if bufferSize <= 0 {
			bufferSize = 10 // Default buffer size
		}
		ch = make(chan any, bufferSize)
		np.channels[pipe] = ch
	}
	np.mutex.Unlock()

	// Send the message with context
	select {
	case ch <- data:
		np.stats.IncrementSent()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-np.ctx.Done():
		return fmt.Errorf("NetChan system is shutting down")
	}
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (np *Packet) Accept(pipe string, fn func(any)) error {
	// Create a wrapper that converts the function to the expected type
	wrapper := func(data []byte) error {
		var decoded any
		if err := core.Decode(data, &decoded); err != nil {
			return err
		}
		fn(decoded)
		return nil
	}

	return np.Subscribe(pipe, wrapper)
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (np *Packet) Subscribe(pipe string, callback core.Process) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if !np.connected {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Stop existing handler if any
	if stopCh, exists := np.stopChans[pipe]; exists {
		close(stopCh)
		delete(np.stopChans, pipe)
	}

	// Get or create channel for the pipe
	ch, exists := np.channels[pipe]
	if !exists {
		bufferSize := np.config.ChannelBufferSize
		if bufferSize <= 0 {
			bufferSize = 10 // Default buffer size
		}
		ch = make(chan any, bufferSize)
		np.channels[pipe] = ch
	}

	// Store the handler
	np.handlers[pipe] = callback

	// Create a stop channel for the handler goroutine
	stopCh := make(chan struct{})
	np.stopChans[pipe] = stopCh

	// Start a goroutine to handle messages
	go np.readMessages(pipe, ch, callback, stopCh)

	log.Printf("Subscribed to NetChan pipe %s", pipe)
	return nil
}

// SubscribeWithContext registers a callback with context for cancellation.
func (np *Packet) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if !np.connected {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Stop existing handler if any
	if stopCh, exists := np.stopChans[pipe]; exists {
		close(stopCh)
		delete(np.stopChans, pipe)
	}

	// Get or create channel for the pipe
	ch, exists := np.channels[pipe]
	if !exists {
		bufferSize := np.config.ChannelBufferSize
		if bufferSize <= 0 {
			bufferSize = 10 // Default buffer size
		}
		ch = make(chan any, bufferSize)
		np.channels[pipe] = ch
	}

	// Store the handler
	np.handlers[pipe] = callback

	// Create a stop channel for the handler goroutine
	stopCh := make(chan struct{})
	np.stopChans[pipe] = stopCh

	// Start a goroutine to handle messages
	go np.readMessagesWithContext(ctx, pipe, ch, callback, stopCh)

	// Set up a goroutine to unsubscribe when the context is canceled
	go func() {
		select {
		case <-ctx.Done():
			np.Unsubscribe(pipe)
		case <-stopCh:
			// Stop channel was closed, nothing to do
		}
	}()

	log.Printf("Subscribed to NetChan pipe %s with context", pipe)
	return nil
}

// readMessages reads messages from a channel in a loop until the stop channel is closed.
func (np *Packet) readMessages(pipe string, ch <-chan any, callback core.Process, stopCh <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in NetChan reader for pipe %s: %v", pipe, r)
			np.stats.IncrementErrors()
		}
	}()

	for {
		select {
		case <-stopCh:
			log.Printf("Stopping NetChan reader for pipe %s", pipe)
			return
		case <-np.ctx.Done():
			log.Printf("NetChan system is shutting down")
			return
		case data := <-ch:
			np.stats.IncrementReceived()

			// Encode the data to bytes for the callback
			bytes, err := core.Encode(data)
			if err != nil {
				log.Printf("Failed to encode message from pipe %s: %v", pipe, err)
				np.stats.IncrementErrors()
				continue
			}

			// Process the message
			if err := callback(bytes); err != nil {
				log.Printf("Error processing message from pipe %s: %v", pipe, err)
				np.stats.IncrementErrors()
			}
		}
	}
}

// readMessagesWithContext reads messages from a channel in a loop until the context is canceled or the stop channel is closed.
func (np *Packet) readMessagesWithContext(ctx context.Context, pipe string, ch <-chan any, callback core.Process, stopCh <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in NetChan reader for pipe %s: %v", pipe, r)
			np.stats.IncrementErrors()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context canceled for NetChan reader for pipe %s", pipe)
			return
		case <-stopCh:
			log.Printf("Stopping NetChan reader for pipe %s", pipe)
			return
		case <-np.ctx.Done():
			log.Printf("NetChan system is shutting down")
			return
		case data := <-ch:
			np.stats.IncrementReceived()

			// Check context before processing
			if ctx.Err() != nil {
				return
			}

			// Encode the data to bytes for the callback
			bytes, err := core.Encode(data)
			if err != nil {
				log.Printf("Failed to encode message from pipe %s: %v", pipe, err)
				np.stats.IncrementErrors()
				continue
			}

			// Process the message
			if err := callback(bytes); err != nil {
				log.Printf("Error processing message from pipe %s: %v", pipe, err)
				np.stats.IncrementErrors()
			}
		}
	}
}

// Remove unsubscribes from the specified pipe.
func (np *Packet) Remove(pipe string) error {
	return np.Unsubscribe(pipe)
}

// Unsubscribe removes a subscription from the specified pipe.
func (np *Packet) Unsubscribe(pipe string) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if !np.connected {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Check if the pipe is subscribed
	if _, exists := np.handlers[pipe]; !exists {
		return core.ErrPipeNotFound
	}

	// Stop the handler goroutine
	if stopCh, exists := np.stopChans[pipe]; exists {
		close(stopCh)
		delete(np.stopChans, pipe)
	}

	// Remove the handler
	delete(np.handlers, pipe)

	log.Printf("Unsubscribed from NetChan pipe %s", pipe)
	return nil
}

// Disconnect closes the connection to the NetChan system.
func (np *Packet) Disconnect() error {
	// Check if connected with read lock
	np.mutex.RLock()
	if !np.connected {
		np.mutex.RUnlock()
		return nil
	}
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Cancel the context to signal shutdown
	if np.cancel != nil {
		np.cancel()
	}

	// Stop all handler goroutines
	for pipe, stopCh := range np.stopChans {
		close(stopCh)
		delete(np.stopChans, pipe)
		log.Printf("Stopped handler for pipe %s", pipe)
	}

	// Clear the maps
	np.channels = make(map[string]chan any)
	np.handlers = make(map[string]core.Process)
	np.connected = false

	log.Printf("Disconnected from NetChan system")
	return nil
}

// Close terminates the connection to the NetChan system.
func (np *Packet) Close() error {
	return np.Disconnect()
}

// IsConnected returns true if the connection to the NetChan system is active.
func (np *Packet) IsConnected() bool {
	np.mutex.RLock()
	defer np.mutex.RUnlock()
	return np.connected
}

// GetStats returns the current message statistics.
func (np *Packet) GetStats() (int64, int64, int64) {
	return np.stats.GetStats()
}
