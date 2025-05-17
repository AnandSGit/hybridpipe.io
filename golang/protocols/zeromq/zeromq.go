// Package zeromq provides an implementation of the HybridPipe interface for ZeroMQ messaging system.
package zeromq

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pebbe/zmq4"
	"hybridpipe.io/core"
)

// Packet implements the HybridPipe interface for ZeroMQ messaging system.
type Packet struct {
	// publisher is the ZeroMQ publisher socket
	publisher *zmq4.Socket
	// subscriber is the ZeroMQ subscriber socket
	subscriber *zmq4.Socket
	// handlers maps pipe names to their message handlers
	handlers map[string]core.Process
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// stopChan is used to signal goroutines to stop
	stopChan chan struct{}
	// running indicates if the router is running
	running bool
	// config holds the ZeroMQ configuration
	config *core.ZeroMQConfig
}

// New creates a new ZeroMQ packet with the specified configuration.
func New(config *core.ZeroMQConfig) *Packet {
	return &Packet{
		handlers: make(map[string]core.Process),
		stopChan: make(chan struct{}),
		config:   config,
	}
}

// Connect establishes a connection to the ZeroMQ system using the configuration.
func (zp *Packet) Connect() error {
	// Get configuration if not already set
	if zp.config == nil {
		config, err := core.GetProtocolConfig(core.ZEROMQ)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		zmqConfig := config.(core.ZeroMQConfig)
		zp.config = &zmqConfig
	}

	// Initialize maps if needed
	zp.mutex.Lock()
	if zp.handlers == nil {
		zp.handlers = make(map[string]core.Process)
	}
	if zp.stopChan == nil {
		zp.stopChan = make(chan struct{})
	}
	zp.mutex.Unlock()

	// Create publisher socket
	publisher, err := zmq4.NewSocket(zmq4.PUB)
	if err != nil {
		return fmt.Errorf("failed to create ZeroMQ publisher socket: %w", err)
	}

	// Set publisher options
	if zp.config.ZMQPublisherLinger >= 0 {
		if err := publisher.SetLinger(zp.config.ZMQPublisherLinger); err != nil {
			publisher.Close()
			return fmt.Errorf("failed to set publisher linger: %w", err)
		}
	}

	// Connect or bind publisher based on configuration
	if zp.config.ZMQPublisherBind {
		endpoint := fmt.Sprintf("%s://%s:%d", zp.config.ZMQPublisherProtocol, zp.config.ZMQPublisherHost, zp.config.ZMQPublisherPort)
		if err := publisher.Bind(endpoint); err != nil {
			publisher.Close()
			return fmt.Errorf("failed to bind publisher to %s: %w", endpoint, err)
		}
		log.Printf("ZeroMQ publisher bound to %s", endpoint)
	} else {
		endpoint := fmt.Sprintf("%s://%s:%d", zp.config.ZMQPublisherProtocol, zp.config.ZMQPublisherHost, zp.config.ZMQPublisherPort)
		if err := publisher.Connect(endpoint); err != nil {
			publisher.Close()
			return fmt.Errorf("failed to connect publisher to %s: %w", endpoint, err)
		}
		log.Printf("ZeroMQ publisher connected to %s", endpoint)
	}

	// Create subscriber socket
	subscriber, err := zmq4.NewSocket(zmq4.SUB)
	if err != nil {
		publisher.Close()
		return fmt.Errorf("failed to create ZeroMQ subscriber socket: %w", err)
	}

	// Set subscriber options
	if zp.config.ZMQSubscriberLinger >= 0 {
		if err := subscriber.SetLinger(zp.config.ZMQSubscriberLinger); err != nil {
			publisher.Close()
			subscriber.Close()
			return fmt.Errorf("failed to set subscriber linger: %w", err)
		}
	}

	// Connect or bind subscriber based on configuration
	if zp.config.ZMQSubscriberBind {
		endpoint := fmt.Sprintf("%s://%s:%d", zp.config.ZMQSubscriberProtocol, zp.config.ZMQSubscriberHost, zp.config.ZMQSubscriberPort)
		if err := subscriber.Bind(endpoint); err != nil {
			publisher.Close()
			subscriber.Close()
			return fmt.Errorf("failed to bind subscriber to %s: %w", endpoint, err)
		}
		log.Printf("ZeroMQ subscriber bound to %s", endpoint)
	} else {
		endpoint := fmt.Sprintf("%s://%s:%d", zp.config.ZMQSubscriberProtocol, zp.config.ZMQSubscriberHost, zp.config.ZMQSubscriberPort)
		if err := subscriber.Connect(endpoint); err != nil {
			publisher.Close()
			subscriber.Close()
			return fmt.Errorf("failed to connect subscriber to %s: %w", endpoint, err)
		}
		log.Printf("ZeroMQ subscriber connected to %s", endpoint)
	}

	// Store the sockets
	zp.mutex.Lock()
	zp.publisher = publisher
	zp.subscriber = subscriber
	zp.running = true
	zp.mutex.Unlock()

	// Start the message receiver
	go zp.receiveMessages()

	return nil
}

// Dispatch sends a message to the specified pipe.
func (zp *Packet) Dispatch(pipe string, data interface{}) error {
	// Validate connection with read lock
	zp.mutex.RLock()
	if zp.publisher == nil {
		zp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	publisher := zp.publisher
	zp.mutex.RUnlock()

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Send the message
	// ZeroMQ requires a multipart message with the topic as the first part
	_, err = publisher.SendMessage(pipe, bytes)
	if err != nil {
		return fmt.Errorf("failed to send message to pipe %s: %w", pipe, err)
	}

	return nil
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (zp *Packet) DispatchWithContext(ctx context.Context, pipe string, data interface{}) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with dispatch
	}

	// Create a channel to signal completion
	done := make(chan error, 1)

	// Dispatch the message asynchronously
	go func() {
		done <- zp.Dispatch(pipe, data)
	}()

	// Wait for either context cancellation or dispatch completion
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (zp *Packet) Subscribe(pipe string, callback core.Process) error {
	// Validate connection with read lock
	zp.mutex.RLock()
	if zp.subscriber == nil {
		zp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	subscriber := zp.subscriber
	zp.mutex.RUnlock()

	// Lock for thread safety
	zp.mutex.Lock()
	defer zp.mutex.Unlock()

	// Subscribe to the pipe
	if err := subscriber.SetSubscribe(pipe); err != nil {
		return fmt.Errorf("failed to subscribe to pipe %s: %w", pipe, err)
	}

	// Store the handler
	zp.handlers[pipe] = callback

	log.Printf("Subscribed to ZeroMQ pipe %s", pipe)
	return nil
}

// SubscribeWithContext registers a callback with context for cancellation.
func (zp *Packet) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with subscription
	}

	// Create a wrapper that checks context before processing
	wrapper := func(data []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return callback(data)
		}
	}

	// Subscribe with the wrapper
	if err := zp.Subscribe(pipe, wrapper); err != nil {
		return err
	}

	// Set up a goroutine to unsubscribe when the context is canceled
	go func() {
		<-ctx.Done()
		zp.Unsubscribe(pipe)
	}()

	return nil
}

// Unsubscribe removes a subscription from the specified pipe.
func (zp *Packet) Unsubscribe(pipe string) error {
	// Validate connection with read lock
	zp.mutex.RLock()
	if zp.subscriber == nil {
		zp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	subscriber := zp.subscriber
	zp.mutex.RUnlock()

	// Lock for thread safety
	zp.mutex.Lock()
	defer zp.mutex.Unlock()

	// Check if the pipe is subscribed
	if _, exists := zp.handlers[pipe]; !exists {
		return core.ErrPipeNotFound
	}

	// Unsubscribe from the pipe
	if err := subscriber.SetUnsubscribe(pipe); err != nil {
		return fmt.Errorf("failed to unsubscribe from pipe %s: %w", pipe, err)
	}

	// Remove the handler
	delete(zp.handlers, pipe)

	log.Printf("Unsubscribed from ZeroMQ pipe %s", pipe)
	return nil
}

// receiveMessages receives messages from the ZeroMQ subscriber socket.
func (zp *Packet) receiveMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in ZeroMQ receiver: %v", r)
		}
	}()

	// Create a poller to check for messages
	poller := zmq4.NewPoller()
	zp.mutex.RLock()
	subscriber := zp.subscriber
	poller.Add(subscriber, zmq4.POLLIN)
	zp.mutex.RUnlock()

	// Poll for messages
	for {
		// Check if router is still running
		zp.mutex.RLock()
		running := zp.running
		stopChan := zp.stopChan
		zp.mutex.RUnlock()

		if !running {
			return
		}

		// Check for stop signal
		select {
		case <-stopChan:
			return
		default:
			// Continue processing
		}

		// Poll for messages with timeout
		sockets, err := poller.Poll(time.Millisecond * 100)
		if err != nil {
			log.Printf("Error polling for ZeroMQ messages: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// Process messages
		for _, socket := range sockets {
			if socket.Socket != subscriber {
				continue
			}

			// Receive the message
			msg, err := subscriber.RecvMessageBytes(0)
			if err != nil {
				log.Printf("Error receiving ZeroMQ message: %v", err)
				continue
			}

			// ZeroMQ messages are multipart with the topic as the first part
			if len(msg) < 2 {
				log.Printf("Received invalid ZeroMQ message: too few parts")
				continue
			}

			// Extract the pipe and message data
			pipe := string(msg[0])
			data := msg[1]

			// Find the handler for the pipe
			zp.mutex.RLock()
			handler, exists := zp.handlers[pipe]
			zp.mutex.RUnlock()

			if !exists {
				log.Printf("No handler for pipe %s", pipe)
				continue
			}

			// Process the message
			if err := handler(data); err != nil {
				log.Printf("Error processing message from pipe %s: %v", pipe, err)
			}
		}
	}
}

// Close terminates the connection to the ZeroMQ system.
func (zp *Packet) Close() error {
	// Check if connected with read lock
	zp.mutex.RLock()
	if zp.publisher == nil && zp.subscriber == nil {
		zp.mutex.RUnlock()
		return nil
	}
	zp.mutex.RUnlock()

	// Signal the receiver to stop
	close(zp.stopChan)

	// Lock for thread safety
	zp.mutex.Lock()
	defer zp.mutex.Unlock()

	// Set running to false
	zp.running = false

	// Close the publisher socket
	if zp.publisher != nil {
		if err := zp.publisher.Close(); err != nil {
			log.Printf("Error closing ZeroMQ publisher socket: %v", err)
		}
		zp.publisher = nil
	}

	// Close the subscriber socket
	if zp.subscriber != nil {
		if err := zp.subscriber.Close(); err != nil {
			log.Printf("Error closing ZeroMQ subscriber socket: %v", err)
		}
		zp.subscriber = nil
	}

	// Clear the handlers
	zp.handlers = make(map[string]core.Process)

	// Create a new stop channel
	zp.stopChan = make(chan struct{})

	log.Printf("Disconnected from ZeroMQ")
	return nil
}

// Accept is an alias for Subscribe for backward compatibility.
func (zp *Packet) Accept(pipe string, fn func(interface{})) error {
	// Create a wrapper that converts the function to the expected type
	wrapper := func(data []byte) error {
		var decoded interface{}
		if err := core.Decode(data, &decoded); err != nil {
			return err
		}
		fn(decoded)
		return nil
	}

	return zp.Subscribe(pipe, wrapper)
}

// Remove is an alias for Unsubscribe for backward compatibility.
func (zp *Packet) Remove(pipe string) error {
	return zp.Unsubscribe(pipe)
}
