package hybridpipe

import (
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
		return fmt.Errorf("failed to get or create channel for pipe %s: %w", pipe, err)
	}

	// Send the message with a timeout
	select {
	case ch <- data:
		return nil
	case <-time.After(time.Duration(HPipeConfig.NetChanF.NetChanTimeout) * time.Millisecond):
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
			case msg, ok := <-ch:
				if !ok {
					// Channel closed
					log.Printf("Channel for pipe %s closed", pipe)
					return
				}

				// Get the handler with read lock
				np.mutex.RLock()
				handler, exists := np.handlers[pipe]
				np.mutex.RUnlock()

				if exists {
					// Process the message
					handler(msg)
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

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Signal all accept goroutines to stop
	for pipe, stopChan := range np.stopChans {
		close(stopChan)
		log.Printf("Signaled stop for pipe %s", pipe)
	}

	// Clear the maps
	np.channels = make(map[string]chan interface{})
	np.handlers = make(map[string]Process)
	np.stopChans = make(map[string]chan struct{})

	np.connected = false
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
