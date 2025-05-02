package hybridpipe

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// NetChanPacket implements the HybridPipe interface for NetChan messaging system.
type NetChanPacket struct {
	// channels maps pipe names to their channels
	channels map[string]chan interface{}
	// handlers maps pipe names to their message handlers
	handlers map[string]Process
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
}

// Connect establishes a connection to the NetChan system.
// For NetChan, this just initializes the internal structures.
func (np *NetChanPacket) Connect() error {
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Initialize the maps if they don't exist
	if np.channels == nil {
		np.channels = make(map[string]chan interface{})
	}
	if np.handlers == nil {
		np.handlers = make(map[string]Process)
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
func (np *NetChanPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if !np.connected {
		np.mutex.RUnlock()
		return fmt.Errorf("not connected to NetChan system")
	}
	np.mutex.RUnlock()

	// Get or create the channel for this pipe
	ch, err := np.getOrCreateChannel(pipe)
	if err != nil {
		if np.stats != nil {
			np.stats.IncrementError()
		}
		return fmt.Errorf("failed to get or create channel for pipe %s: %w", pipe, err)
	}

	// Get timeout from config
	timeout := HPipeConfig.NetChanF.NetChanTimeout
	if timeout <= 0 {
		timeout = 5000 // Default timeout: 5 seconds
	}

	// Send the message with a timeout
	select {
	case ch <- data:
		// Update statistics
		if np.stats != nil {
			// Estimate message size (this is approximate)
			var size uint64 = 64 // Base size
			np.stats.IncrementSent(size)
		}
		return nil
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		if np.stats != nil {
			np.stats.IncrementError()
		}
		return fmt.Errorf("timeout sending message to pipe %s", pipe)
	}
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (np *NetChanPacket) Accept(pipe string, fn Process) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if !np.connected {
		np.mutex.RUnlock()
		return fmt.Errorf("not connected to NetChan system")
	}
	np.mutex.RUnlock()

	// Get or create the channel for this pipe
	ch, err := np.getOrCreateChannel(pipe)
	if err != nil {
		if np.stats != nil {
			np.stats.IncrementError()
		}
		return fmt.Errorf("failed to get or create channel for pipe %s: %w", pipe, err)
	}

	// Store the handler with write lock
	np.mutex.Lock()
	np.handlers[pipe] = fn
	stopChan := make(chan struct{})
	np.stopChans[pipe] = stopChan
	np.mutex.Unlock()

	// Start a goroutine to receive messages
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case <-np.ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					// Channel closed
					log.Printf("Channel for pipe %s closed", pipe)
					return
				}

				// Update statistics
				if np.stats != nil {
					// Estimate message size (this is approximate)
					var size uint64 = 64 // Base size
					np.stats.IncrementReceived(size)
				}

				// Get the handler with read lock
				np.mutex.RLock()
				handler, exists := np.handlers[pipe]
				np.mutex.RUnlock()

				if exists {
					// Process the message with panic recovery
					func() {
						defer func() {
							if r := recover(); r != nil {
								log.Printf("Recovered from panic in handler for pipe %s: %v", pipe, r)
								if np.stats != nil {
									np.stats.IncrementError()
								}
							}
						}()
						handler(msg)
					}()
				}
			}
		}
	}()

	log.Printf("Subscribed to pipe %s", pipe)
	return nil
}

// Remove unsubscribes from the specified pipe.
func (np *NetChanPacket) Remove(pipe string) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if !np.connected {
		np.mutex.RUnlock()
		return fmt.Errorf("not connected to NetChan system")
	}
	np.mutex.RUnlock()

	// Check if the pipe is subscribed with read lock
	np.mutex.RLock()
	_, exists := np.channels[pipe]
	stopChan, stopExists := np.stopChans[pipe]
	np.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}

	// Signal the accept goroutine to stop
	if stopExists {
		close(stopChan)
	}

	// Remove the channel and handler with write lock
	np.mutex.Lock()
	delete(np.channels, pipe)
	delete(np.handlers, pipe)
	delete(np.stopChans, pipe)
	np.mutex.Unlock()

	log.Printf("Unsubscribed from pipe %s", pipe)
	return nil
}

// Close terminates all connections to the NetChan system.
func (np *NetChanPacket) Close() {
	// Check if connected with read lock
	np.mutex.RLock()
	if !np.connected {
		np.mutex.RUnlock()
		return
	}
	np.mutex.RUnlock()

	// Cancel the context to signal all goroutines to stop
	if np.cancel != nil {
		np.cancel()
	}

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Signal all accept goroutines to stop
	for pipe, stopChan := range np.stopChans {
		close(stopChan)
		log.Printf("Signaled stop for pipe %s", pipe)
	}

	// Close all channels
	for pipe := range np.channels {
		// We can't directly close the channel as it might be used by other goroutines
		// Instead, we'll just remove it from the map
		log.Printf("Removing channel for pipe %s", pipe)
	}

	// Clear the maps
	np.channels = make(map[string]chan interface{})
	np.handlers = make(map[string]Process)
	np.stopChans = make(map[string]chan struct{})

	// Reset connection state
	np.connected = false

	// Create a new context for future use
	np.ctx, np.cancel = context.WithCancel(context.Background())

	log.Printf("Closed NetChan connections")
}

// getOrCreateChannel gets or creates a channel for the specified pipe.
func (np *NetChanPacket) getOrCreateChannel(pipe string) (chan interface{}, error) {
	// Check if the channel already exists with read lock
	np.mutex.RLock()
	ch, exists := np.channels[pipe]
	np.mutex.RUnlock()

	if exists {
		return ch, nil
	}

	// Create a new channel with buffer size from config
	bufferSize := HPipeConfig.NetChanF.NetChanBufferSize
	if bufferSize <= 0 {
		bufferSize = 10 // Default buffer size
	}
	ch = make(chan interface{}, bufferSize)

	// Store the channel with write lock
	np.mutex.Lock()
	np.channels[pipe] = ch
	np.mutex.Unlock()

	return ch, nil
}

// GetStats returns the message statistics.
func (np *NetChanPacket) GetStats() *MessageStats {
	return np.stats
}

// ResetStats resets the message statistics.
func (np *NetChanPacket) ResetStats() {
	if np.stats != nil {
		np.stats.Reset()
	}
}

// GetChannelCount returns the number of active channels.
func (np *NetChanPacket) GetChannelCount() int {
	np.mutex.RLock()
	defer np.mutex.RUnlock()
	return len(np.channels)
}

// GetSubscriptionCount returns the number of active subscriptions.
func (np *NetChanPacket) GetSubscriptionCount() int {
	np.mutex.RLock()
	defer np.mutex.RUnlock()
	return len(np.handlers)
}

// IsConnected returns whether the NetChan system is connected.
func (np *NetChanPacket) IsConnected() bool {
	np.mutex.RLock()
	defer np.mutex.RUnlock()
	return np.connected
}
