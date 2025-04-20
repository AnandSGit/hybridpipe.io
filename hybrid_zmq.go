package hybridpipe

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// ZMQPacket implements the HybridPipe interface for ZeroMQ messaging system.
type ZMQPacket struct {
	// channels maps pipe names to their channels
	channels map[string]chan []byte
	// handlers maps pipe names to their message handlers
	handlers map[string]Process
	// cancelFuncs maps pipe names to their context cancel functions
	cancelFuncs map[string]context.CancelFunc
	// connected indicates if the ZMQ system is connected
	connected bool
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// endpoints stores the configured endpoints
	endpoints []string
	// publishEndpoint is the endpoint for publishing messages
	publishEndpoint string
	// subscribeEndpoint is the endpoint for subscribing to messages
	subscribeEndpoint string
}

// Connect establishes a connection to the ZeroMQ system.
func (zp *ZMQPacket) Connect() error {
	// Initialize maps if needed
	zp.mutex.Lock()
	defer zp.mutex.Unlock()

	// Check if already connected
	if zp.connected {
		return nil
	}

	// Initialize maps
	if zp.channels == nil {
		zp.channels = make(map[string]chan []byte)
	}
	if zp.handlers == nil {
		zp.handlers = make(map[string]Process)
	}
	if zp.cancelFuncs == nil {
		zp.cancelFuncs = make(map[string]context.CancelFunc)
	}

	// Parse configuration
	if HPipeConfig.ZMQF.ZMQEndpoints != "" {
		zp.endpoints = strings.Split(HPipeConfig.ZMQF.ZMQEndpoints, ",")
	}
	zp.publishEndpoint = HPipeConfig.ZMQF.ZMQPublishEndpoint
	if zp.publishEndpoint == "" {
		zp.publishEndpoint = "tcp://*:5555" // Default publish endpoint
	}
	zp.subscribeEndpoint = HPipeConfig.ZMQF.ZMQSubscribeEndpoint
	if zp.subscribeEndpoint == "" {
		zp.subscribeEndpoint = "tcp://localhost:5555" // Default subscribe endpoint
	}

	// Mark as connected
	zp.connected = true
	log.Printf("ZeroMQ system initialized (simulated)")
	return nil
}

// Dispatch publishes a message to the specified pipe.
func (zp *ZMQPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection
	zp.mutex.RLock()
	if !zp.connected {
		zp.mutex.RUnlock()
		return fmt.Errorf("not connected to ZeroMQ")
	}
	zp.mutex.RUnlock()

	// Encode the data
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Get or create the channel for this pipe
	ch, err := zp.getOrCreateChannel(pipe)
	if err != nil {
		return fmt.Errorf("failed to get or create channel for pipe %s: %w", pipe, err)
	}

	// Send the message with a timeout
	select {
	case ch <- bytes:
		log.Printf("ZeroMQ: Published message to pipe %s (%d bytes)", pipe, len(bytes))
		return nil
	case <-time.After(time.Duration(HPipeConfig.ZMQF.ZMQConnectTimeout) * time.Second):
		return fmt.Errorf("timeout sending message to pipe %s", pipe)
	}
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (zp *ZMQPacket) Accept(pipe string, fn Process) error {
	// Validate connection
	zp.mutex.RLock()
	if !zp.connected {
		zp.mutex.RUnlock()
		return fmt.Errorf("not connected to ZeroMQ")
	}
	zp.mutex.RUnlock()

	// Get or create the channel for this pipe
	ch, err := zp.getOrCreateChannel(pipe)
	if err != nil {
		return fmt.Errorf("failed to get or create channel for pipe %s: %w", pipe, err)
	}

	// Lock for thread safety during setup
	zp.mutex.Lock()

	// Cancel existing subscriber if any
	if cancel, exists := zp.cancelFuncs[pipe]; exists {
		cancel()
		delete(zp.cancelFuncs, pipe)
	}

	// Store the handler
	zp.handlers[pipe] = fn

	// Create a context with cancel for the subscriber goroutine
	ctx, cancel := context.WithCancel(context.Background())
	zp.cancelFuncs[pipe] = cancel

	// Start a goroutine to receive messages
	go func() {
		log.Printf("ZeroMQ: Started subscriber for pipe %s", pipe)

		for {
			select {
			case <-ctx.Done():
				log.Printf("ZeroMQ: Stopping subscriber for pipe %s", pipe)
				return
			case bytes, ok := <-ch:
				if !ok {
					// Channel closed
					log.Printf("ZeroMQ: Channel for pipe %s closed", pipe)
					return
				}

				// Decode the message
				var decodedData interface{}
				if err := Decode(bytes, &decodedData); err != nil {
					log.Printf("ZeroMQ: Failed to decode message: %v", err)
					continue
				}

				// Process the message
				fn(decodedData)
			}
		}
	}()

	zp.mutex.Unlock()
	log.Printf("ZeroMQ: Subscribed to pipe %s", pipe)
	return nil
}

// Remove unsubscribes from the specified pipe.
func (zp *ZMQPacket) Remove(pipe string) error {
	// Validate connection
	zp.mutex.RLock()
	if !zp.connected {
		zp.mutex.RUnlock()
		return fmt.Errorf("not connected to ZeroMQ")
	}
	zp.mutex.RUnlock()

	// Lock for thread safety
	zp.mutex.Lock()
	defer zp.mutex.Unlock()

	// Check if the pipe is subscribed
	_, exists := zp.channels[pipe]
	if !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}

	// Cancel the subscriber goroutine
	if cancel, exists := zp.cancelFuncs[pipe]; exists {
		cancel()
		delete(zp.cancelFuncs, pipe)
	}

	// Remove the subscription and handler
	delete(zp.channels, pipe)
	delete(zp.handlers, pipe)

	log.Printf("ZeroMQ: Unsubscribed from pipe %s", pipe)
	return nil
}

// Close terminates all connections to the ZeroMQ system.
func (zp *ZMQPacket) Close() {
	// Lock for thread safety
	zp.mutex.Lock()
	defer zp.mutex.Unlock()

	// Check if connected
	if !zp.connected {
		return
	}

	// Cancel all subscriber goroutines
	for pipe, cancel := range zp.cancelFuncs {
		cancel()
		delete(zp.cancelFuncs, pipe)
		log.Printf("ZeroMQ: Canceled subscriber for pipe %s", pipe)
	}

	// Clear maps
	zp.channels = make(map[string]chan []byte)
	zp.handlers = make(map[string]Process)
	zp.cancelFuncs = make(map[string]context.CancelFunc)

	// Mark as disconnected
	zp.connected = false

	log.Printf("ZeroMQ: Closed all connections")
}

// getOrCreateChannel gets or creates a channel for the specified pipe.
func (zp *ZMQPacket) getOrCreateChannel(pipe string) (chan []byte, error) {
	// Check if the channel already exists with read lock
	zp.mutex.RLock()
	ch, exists := zp.channels[pipe]
	zp.mutex.RUnlock()

	if exists {
		return ch, nil
	}

	// Create a new channel with buffer size from config
	bufferSize := HPipeConfig.ZMQF.ZMQHighWaterMark
	if bufferSize <= 0 {
		bufferSize = 1000 // Default buffer size
	}
	ch = make(chan []byte, bufferSize)

	// Store the channel with write lock
	zp.mutex.Lock()
	zp.channels[pipe] = ch
	zp.mutex.Unlock()

	return ch, nil
}
