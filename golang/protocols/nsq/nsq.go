// Package nsq provides an implementation of the HybridPipe interface for NSQ messaging system.
package nsq

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nsqio/go-nsq"
	"hybridpipe.io/core"
)

// nsqMessageHandler implements the nsq.Handler interface.
type nsqMessageHandler struct {
	process core.Process
}

// HandleMessage implements the nsq.Handler interface.
func (h *nsqMessageHandler) HandleMessage(msg *nsq.Message) error {
	// Mark the message as processed
	msg.Finish()

	// Process the message
	return h.process(msg.Body)
}

// Packet implements the HybridPipe interface for NSQ messaging system.
type Packet struct {
	// producer is the NSQ producer client
	producer *nsq.Producer
	// consumers maps pipe names to their consumer clients
	consumers map[string]*nsq.Consumer
	// handlers maps pipe names to their message handlers
	handlers map[string]nsq.Handler
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// config holds the NSQ configuration
	config *core.NSQConfig
}

// New creates a new NSQ packet with the specified configuration.
func New(config *core.NSQConfig) *Packet {
	return &Packet{
		consumers: make(map[string]*nsq.Consumer),
		handlers:  make(map[string]nsq.Handler),
		config:    config,
	}
}

// Connect establishes a connection to the NSQ server using the configuration.
func (np *Packet) Connect() error {
	// Get configuration if not already set
	if np.config == nil {
		config, err := core.GetProtocolConfig(core.NSQ)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		nsqConfig := config.(core.NSQConfig)
		np.config = &nsqConfig
	}

	// Initialize maps if needed
	np.mutex.Lock()
	if np.consumers == nil {
		np.consumers = make(map[string]*nsq.Consumer)
	}
	if np.handlers == nil {
		np.handlers = make(map[string]nsq.Handler)
	}
	np.mutex.Unlock()

	// Create NSQ config
	config := nsq.NewConfig()

	// Set connection timeout
	if np.config.NSQConnectTimeout > 0 {
		config.DialTimeout = time.Duration(np.config.NSQConnectTimeout) * time.Second
	}

	// Set max in flight
	if np.config.NSQMaxInFlight > 0 {
		config.MaxInFlight = np.config.NSQMaxInFlight
	}

	// Set authentication if provided
	if np.config.NSQAuthSecret != "" {
		config.AuthSecret = np.config.NSQAuthSecret
	}

	// Create and connect the producer
	nsqdAddress := fmt.Sprintf("%s:%d", np.config.NSQDAddress, np.config.NSQDPort)
	producer, err := nsq.NewProducer(nsqdAddress, config)
	if err != nil {
		return fmt.Errorf("failed to create NSQ producer: %w", err)
	}

	// Ping the NSQ server to verify connection
	if err := producer.Ping(); err != nil {
		producer.Stop()
		return fmt.Errorf("failed to connect to NSQ server: %w", err)
	}

	// Store the producer
	np.mutex.Lock()
	np.producer = producer
	np.mutex.Unlock()

	log.Printf("Connected to NSQ server at %s", nsqdAddress)
	return nil
}

// Dispatch sends a message to the specified pipe.
func (np *Packet) Dispatch(pipe string, data any) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if np.producer == nil {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	producer := np.producer
	np.mutex.RUnlock()

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Publish the message
	if err := producer.Publish(pipe, bytes); err != nil {
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
	if np.producer == nil {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	producer := np.producer
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
		err := producer.Publish(pipe, bytes)
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
	if np.producer == nil {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Close existing consumer if any
	if consumer, exists := np.consumers[pipe]; exists {
		consumer.Stop()
		delete(np.consumers, pipe)
	}

	// Create NSQ config
	config := nsq.NewConfig()

	// Set connection timeout
	if np.config.NSQConnectTimeout > 0 {
		config.DialTimeout = time.Duration(np.config.NSQConnectTimeout) * time.Second
	}

	// Set max in flight
	if np.config.NSQMaxInFlight > 0 {
		config.MaxInFlight = np.config.NSQMaxInFlight
	}

	// Set requeue delay
	if np.config.NSQRequeueDelay > 0 {
		config.DefaultRequeueDelay = time.Duration(np.config.NSQRequeueDelay) * time.Second
	}

	// Create a unique channel name based on client ID
	channelName := "hybridpipe"
	if np.config.NSQClientID != "" {
		channelName = np.config.NSQClientID
	}

	// Create and connect the consumer
	consumer, err := nsq.NewConsumer(pipe, channelName, config)
	if err != nil {
		return fmt.Errorf("failed to create NSQ consumer for pipe %s: %w", pipe, err)
	}

	// Create a message handler
	handler := &nsqMessageHandler{process: callback}
	consumer.AddHandler(handler)

	// Store the handler
	np.handlers[pipe] = handler

	// Connect to NSQ lookupd or nsqd
	if np.config.NSQLookupdAddresses != "" {
		// Connect to NSQ lookupd
		addresses := strings.Split(np.config.NSQLookupdAddresses, ",")
		for i, addr := range addresses {
			addresses[i] = strings.TrimSpace(addr)
		}
		err = consumer.ConnectToNSQLookupds(addresses)
	} else {
		// Connect to NSQ daemon
		nsqdAddress := fmt.Sprintf("%s:%d", np.config.NSQDAddress, np.config.NSQDPort)
		err = consumer.ConnectToNSQD(nsqdAddress)
	}

	if err != nil {
		consumer.Stop()
		delete(np.handlers, pipe)
		return fmt.Errorf("failed to connect NSQ consumer for pipe %s: %w", pipe, err)
	}

	// Store the consumer
	np.consumers[pipe] = consumer

	log.Printf("Subscribed to NSQ pipe %s", pipe)
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

	// Subscribe normally
	if err := np.Subscribe(pipe, callback); err != nil {
		return err
	}

	// Set up a goroutine to unsubscribe when the context is canceled
	go func() {
		<-ctx.Done()
		np.Unsubscribe(pipe)
	}()

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
	if np.producer == nil {
		np.mutex.RUnlock()
		return core.ErrNotConnected
	}
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Check if the pipe is subscribed
	consumer, exists := np.consumers[pipe]
	if !exists {
		return core.ErrPipeNotFound
	}

	// Stop the consumer
	consumer.Stop()

	// Remove the consumer and handler
	delete(np.consumers, pipe)
	delete(np.handlers, pipe)

	log.Printf("Unsubscribed from NSQ pipe %s", pipe)
	return nil
}

// Disconnect closes the connection to the NSQ server.
func (np *Packet) Disconnect() error {
	// Check if connected with read lock
	np.mutex.RLock()
	if np.producer == nil {
		np.mutex.RUnlock()
		return nil
	}
	producer := np.producer
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Stop all consumers
	for pipe, consumer := range np.consumers {
		consumer.Stop()
		log.Printf("Stopped consumer for pipe %s", pipe)
	}
	np.consumers = make(map[string]*nsq.Consumer)
	np.handlers = make(map[string]nsq.Handler)

	// Stop the producer
	producer.Stop()
	np.producer = nil

	log.Printf("Disconnected from NSQ server")
	return nil
}

// Close terminates the connection to the NSQ server.
func (np *Packet) Close() error {
	return np.Disconnect()
}

// IsConnected returns true if the connection to the NSQ server is active.
func (np *Packet) IsConnected() bool {
	np.mutex.RLock()
	defer np.mutex.RUnlock()
	return np.producer != nil
}
