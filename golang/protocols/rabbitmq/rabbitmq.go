// Package rabbitmq provides an implementation of the HybridPipe interface for RabbitMQ messaging system.
package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"sync"

	amqp "github.com/streadway/amqp"
	"hybridpipe.io/core"
)

// Packet implements the HybridPipe interface for RabbitMQ messaging system.
type Packet struct {
	// HandleConn is the connection to the RabbitMQ server
	HandleConn *amqp.Connection
	// RChannel is the AMQP channel for communication
	RChannel *amqp.Channel
	// consumers maps pipe names to their consumer cancel channels
	consumers map[string]chan bool
	// mutex protects concurrent access to the consumers map
	mutex sync.RWMutex
	// config holds the RabbitMQ configuration
	config *core.RabbitMQConfig
}

// New creates a new RabbitMQ packet with the specified configuration.
func New(config *core.RabbitMQConfig) *Packet {
	return &Packet{
		consumers: make(map[string]chan bool),
		config:    config,
	}
}

// Connect establishes a connection to the RabbitMQ server using the configuration.
func (rp *Packet) Connect() error {
	// Get configuration if not already set
	if rp.config == nil {
		config, err := core.GetProtocolConfig(core.RABBITMQ)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		rabbitConfig := config.(core.RabbitMQConfig)
		rp.config = &rabbitConfig
	}

	// Initialize the consumers map if needed
	rp.mutex.Lock()
	if rp.consumers == nil {
		rp.consumers = make(map[string]chan bool)
	}
	rp.mutex.Unlock()

	// Build the connection string
	connStr := rp.config.RServerPort
	if connStr == "" {
		connStr = "amqp://guest:guest@localhost:5672/"
	}

	// Connect to RabbitMQ server
	conn, err := amqp.Dial(connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ server: %w", err)
	}

	// Create a channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	// Store the connection and channel
	rp.mutex.Lock()
	rp.HandleConn = conn
	rp.RChannel = ch
	rp.mutex.Unlock()

	log.Printf("Connected to RabbitMQ server at %s", connStr)
	return nil
}

// Dispatch sends a message to the specified pipe.
func (rp *Packet) Dispatch(pipe string, data any) error {
	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.HandleConn == nil || rp.RChannel == nil {
		rp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	ch := rp.RChannel
	rp.mutex.RUnlock()

	// Declare the queue
	_, err := ch.QueueDeclare(
		pipe,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", pipe, err)
	}

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Publish the message
	err = ch.Publish(
		"",    // exchange
		pipe,  // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/octet-stream",
			Body:        bytes,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
	}

	return nil
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (rp *Packet) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with dispatch
	}

	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.HandleConn == nil || rp.RChannel == nil {
		rp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	ch := rp.RChannel
	rp.mutex.RUnlock()

	// Declare the queue
	_, err := ch.QueueDeclare(
		pipe,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", pipe, err)
	}

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Create a channel to signal completion
	done := make(chan error, 1)

	// Publish the message asynchronously
	go func() {
		err := ch.Publish(
			"",    // exchange
			pipe,  // routing key
			false, // mandatory
			false, // immediate
			amqp.Publishing{
				ContentType: "application/octet-stream",
				Body:        bytes,
			},
		)
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
func (rp *Packet) Accept(pipe string, fn func(any)) error {
	// Create a wrapper that converts the function to the expected type
	wrapper := func(data []byte) error {
		var decoded any
		if err := core.Decode(data, &decoded); err != nil {
			return err
		}
		fn(decoded)
		return nil
	}

	return rp.Subscribe(pipe, wrapper)
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (rp *Packet) Subscribe(pipe string, callback core.Process) error {
	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.HandleConn == nil || rp.RChannel == nil {
		rp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	ch := rp.RChannel
	rp.mutex.RUnlock()

	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Cancel existing consumer if any
	if cancelCh, exists := rp.consumers[pipe]; exists {
		cancelCh <- true
		close(cancelCh)
		delete(rp.consumers, pipe)
	}

	// Declare the queue
	_, err := ch.QueueDeclare(
		pipe,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", pipe, err)
	}

	// Create a consumer
	msgs, err := ch.Consume(
		pipe,  // queue
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer for pipe %s: %w", pipe, err)
	}

	// Create a cancel channel
	cancelCh := make(chan bool)
	rp.consumers[pipe] = cancelCh

	// Start a goroutine to handle messages
	go func() {
		for {
			select {
			case <-cancelCh:
				log.Printf("Stopping RabbitMQ consumer for pipe %s", pipe)
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Printf("RabbitMQ channel closed for pipe %s", pipe)
					return
				}

				// Process the message
				if err := callback(msg.Body); err != nil {
					log.Printf("Error processing message from pipe %s: %v", pipe, err)
				}
			}
		}
	}()

	log.Printf("Subscribed to RabbitMQ pipe %s", pipe)
	return nil
}

// SubscribeWithContext registers a callback with context for cancellation.
func (rp *Packet) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with subscription
	}

	// Subscribe normally
	if err := rp.Subscribe(pipe, callback); err != nil {
		return err
	}

	// Set up a goroutine to unsubscribe when the context is canceled
	go func() {
		<-ctx.Done()
		rp.Unsubscribe(pipe)
	}()

	return nil
}

// Remove unsubscribes from the specified pipe.
func (rp *Packet) Remove(pipe string) error {
	return rp.Unsubscribe(pipe)
}

// Unsubscribe removes a subscription from the specified pipe.
func (rp *Packet) Unsubscribe(pipe string) error {
	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.HandleConn == nil || rp.RChannel == nil {
		rp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	rp.mutex.RUnlock()

	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Check if the pipe is subscribed
	cancelCh, exists := rp.consumers[pipe]
	if !exists {
		return core.ErrPipeNotFound
	}

	// Cancel the consumer
	cancelCh <- true
	close(cancelCh)
	delete(rp.consumers, pipe)

	log.Printf("Unsubscribed from RabbitMQ pipe %s", pipe)
	return nil
}

// Disconnect closes the connection to the RabbitMQ server.
func (rp *Packet) Disconnect() error {
	// Check if connected with read lock
	rp.mutex.RLock()
	if rp.HandleConn == nil {
		rp.mutex.RUnlock()
		return nil
	}
	conn := rp.HandleConn
	ch := rp.RChannel
	rp.mutex.RUnlock()

	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Cancel all consumers
	for pipe, cancelCh := range rp.consumers {
		cancelCh <- true
		close(cancelCh)
		log.Printf("Canceled consumer for pipe %s", pipe)
	}
	rp.consumers = make(map[string]chan bool)

	// Close the channel
	if ch != nil {
		if err := ch.Close(); err != nil {
			log.Printf("Error closing RabbitMQ channel: %v", err)
		}
		rp.RChannel = nil
	}

	// Close the connection
	if err := conn.Close(); err != nil {
		log.Printf("Error closing RabbitMQ connection: %v", err)
		return err
	}
	rp.HandleConn = nil

	log.Printf("Disconnected from RabbitMQ server")
	return nil
}

// Close terminates the connection to the RabbitMQ server.
func (rp *Packet) Close() error {
	return rp.Disconnect()
}

// IsConnected returns true if the connection to the RabbitMQ server is active.
func (rp *Packet) IsConnected() bool {
	rp.mutex.RLock()
	defer rp.mutex.RUnlock()
	return rp.HandleConn != nil && rp.RChannel != nil
}
