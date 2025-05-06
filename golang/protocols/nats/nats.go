// Package nats provides an implementation of the HybridPipe interface for NATS messaging system.
package nats

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	nats "github.com/nats-io/nats.go"
	"hybridpipe.io/core"
)

// Packet implements the HybridPipe interface for NATS messaging system.
type Packet struct {
	// HandleConn is the connection to the NATS server
	HandleConn *nats.Conn
	// PipeHandle maps pipe names to their subscriptions
	PipeHandle map[string]*nats.Subscription
	// mutex protects concurrent access to the connection and subscriptions
	mutex sync.RWMutex
	// config holds the NATS configuration
	config *core.NATSConfig
}

// New creates a new NATS packet with the specified configuration.
func New(config *core.NATSConfig) *Packet {
	return &Packet{
		PipeHandle: make(map[string]*nats.Subscription),
		config:     config,
	}
}

// Connect establishes a connection to the NATS server using the configuration.
func (np *Packet) Connect() error {
	// Get configuration if not already set
	if np.config == nil {
		config, err := core.GetProtocolConfig(core.NATS)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		natsConfig := config.(core.NATSConfig)
		np.config = &natsConfig
	}

	// Build the NATS server URL with port
	serverURL := np.config.NServer
	if np.config.NLport > 0 {
		serverURL = fmt.Sprintf("%s:%d", np.config.NServer, np.config.NLport)
	}

	// Set connection options
	options := []nats.Option{
		nats.Name("HybridPipe NATS Client"),
		nats.ReconnectWait(time.Duration(np.config.NReconnectWait) * time.Second),
		nats.MaxReconnects(np.config.NMaxReconnects),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("NATS connection closed")
		}),
	}

	// Add TLS options if certificate files are provided
	if np.config.NATSCertFile != "" && np.config.NATSKeyFile != "" && np.config.NATSCAFile != "" {
		options = append(options,
			nats.Secure(),
			nats.ClientCert(np.config.NATSCertFile, np.config.NATSKeyFile),
			nats.RootCAs(np.config.NATSCAFile),
		)
	}

	// Connect to NATS server
	nc, err := nats.Connect(serverURL, options...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS server %s: %w", serverURL, err)
	}

	// Initialize the connection and subscription map with write lock
	np.mutex.Lock()
	np.HandleConn = nc
	if np.PipeHandle == nil {
		np.PipeHandle = make(map[string]*nats.Subscription)
	}
	np.mutex.Unlock()

	log.Printf("Connected to NATS server at %s", serverURL)
	return nil
}

// Dispatch sends a message to the specified pipe.
func (np *Packet) Dispatch(pipe string, data any) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if np.HandleConn == nil {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	conn := np.HandleConn
	np.mutex.RUnlock()

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Publish the message
	if err := conn.Publish(pipe, bytes); err != nil {
		return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
	}

	return nil
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (np *Packet) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with dispatch
	}

	// Validate connection with read lock
	np.mutex.RLock()
	if np.HandleConn == nil {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	conn := np.HandleConn
	np.mutex.RUnlock()

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Create a channel to signal completion
	done := make(chan error, 1)

	// Publish the message asynchronously
	go func() {
		err := conn.Publish(pipe, bytes)
		done <- err
	}()

	// Wait for either context cancellation or publish completion
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
		}
		return nil
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
	if np.HandleConn == nil {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	conn := np.HandleConn
	np.mutex.RUnlock()

	// Create a message handler
	handler := func(msg *nats.Msg) {
		if err := callback(msg.Data); err != nil {
			log.Printf("Error processing message from pipe %s: %v", pipe, err)
		}
	}

	// Subscribe to the pipe
	sub, err := conn.Subscribe(pipe, handler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to pipe %s: %w", pipe, err)
	}

	// Store the subscription
	np.mutex.Lock()
	np.PipeHandle[pipe] = sub
	np.mutex.Unlock()

	log.Printf("Subscribed to NATS pipe %s", pipe)
	return nil
}

// SubscribeWithContext registers a callback with context for cancellation.
func (np *Packet) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with subscription
	}

	// Validate connection with read lock
	np.mutex.RLock()
	if np.HandleConn == nil {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	conn := np.HandleConn
	np.mutex.RUnlock()

	// Create a message handler that checks context before processing
	handler := func(msg *nats.Msg) {
		select {
		case <-ctx.Done():
			return
		default:
			if err := callback(msg.Data); err != nil {
				log.Printf("Error processing message from pipe %s: %v", pipe, err)
			}
		}
	}

	// Subscribe to the pipe
	sub, err := conn.Subscribe(pipe, handler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to pipe %s: %w", pipe, err)
	}

	// Store the subscription
	np.mutex.Lock()
	np.PipeHandle[pipe] = sub
	np.mutex.Unlock()

	// Set up a goroutine to unsubscribe when the context is canceled
	go func() {
		<-ctx.Done()
		np.Unsubscribe(pipe)
	}()

	log.Printf("Subscribed to NATS pipe %s with context", pipe)
	return nil
}

// Remove unsubscribes from the specified pipe.
func (np *Packet) Remove(pipe string) error {
	return np.Unsubscribe(pipe)
}

// Unsubscribe removes a subscription from the specified pipe.
func (np *Packet) Unsubscribe(pipe string) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if np.HandleConn == nil {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Check if the pipe is subscribed
	sub, exists := np.PipeHandle[pipe]
	if !exists {
		return core.ErrPipeNotFound
	}

	// Unsubscribe from the pipe
	if err := sub.Unsubscribe(); err != nil {
		return fmt.Errorf("failed to unsubscribe from pipe %s: %w", pipe, err)
	}

	// Remove the subscription
	delete(np.PipeHandle, pipe)

	log.Printf("Unsubscribed from NATS pipe %s", pipe)
	return nil
}

// Disconnect closes the connection to the NATS server.
func (np *Packet) Disconnect() error {
	// Check if connected with read lock
	np.mutex.RLock()
	if np.HandleConn == nil {
		np.mutex.RUnlock()
		return nil
	}
	conn := np.HandleConn
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Unsubscribe from all pipes
	for pipe, sub := range np.PipeHandle {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("Error unsubscribing from pipe %s: %v", pipe, err)
		}
	}
	np.PipeHandle = make(map[string]*nats.Subscription)

	// Close the connection
	conn.Close()
	np.HandleConn = nil

	log.Printf("Disconnected from NATS server")
	return nil
}

// Close terminates the connection to the NATS server.
func (np *Packet) Close() error {
	return np.Disconnect()
}

// IsConnected returns true if the connection to the NATS server is active.
func (np *Packet) IsConnected() bool {
	np.mutex.RLock()
	defer np.mutex.RUnlock()
	return np.HandleConn != nil && np.HandleConn.Status() == nats.CONNECTED
}
