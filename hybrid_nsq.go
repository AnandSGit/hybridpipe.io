package hybridpipe

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nsqio/go-nsq"
)

// NSQPacket implements the HybridPipe interface for NSQ messaging system.
type NSQPacket struct {
	// producer is the NSQ producer client
	producer *nsq.Producer
	// consumers maps pipe names to their consumer clients
	consumers map[string]*nsq.Consumer
	// handlers maps pipe names to their message handlers
	handlers map[string]nsq.Handler
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
}

// Connect establishes a connection to the NSQ server using the configuration in HPipeConfig.
func (np *NSQPacket) Connect() error {
	// Initialize maps if needed
	np.mutex.Lock()
	if np.consumers == nil {
		np.consumers = make(map[string]*nsq.Consumer)
	}
	if np.handlers == nil {
		np.handlers = make(map[string]nsq.Handler)
	}
	np.mutex.Unlock()

	// Create NSQ producer configuration
	config := nsq.NewConfig()
	
	// Set connection timeout
	timeout := time.Duration(HPipeConfig.NSQF.NSQConnectTimeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second // Default timeout
	}
	config.DialTimeout = timeout
	
	// Set client ID if provided
	if HPipeConfig.NSQF.NSQClientID != "" {
		config.ClientID = HPipeConfig.NSQF.NSQClientID
	}
	
	// Set authentication if provided
	if HPipeConfig.NSQF.NSQAuthSecret != "" {
		config.AuthSecret = HPipeConfig.NSQF.NSQAuthSecret
	}

	// Create and connect the producer
	nsqdAddress := fmt.Sprintf("%s:%d", HPipeConfig.NSQF.NSQDAddress, HPipeConfig.NSQF.NSQDPort)
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

// Dispatch publishes a message to the specified pipe.
func (np *NSQPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if np.producer == nil {
		np.mutex.RUnlock()
		return fmt.Errorf("not connected to NSQ server")
	}
	producer := np.producer
	np.mutex.RUnlock()

	// Encode the data
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Publish the message
	err = producer.Publish(pipe, bytes)
	if err != nil {
		return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
	}

	return nil
}

// nsqMessageHandler wraps a Process function to handle NSQ messages.
type nsqMessageHandler struct {
	process Process
}

// HandleMessage implements the nsq.Handler interface.
func (h *nsqMessageHandler) HandleMessage(message *nsq.Message) error {
	// Decode the message
	var data interface{}
	if err := Decode(message.Body, &data); err != nil {
		log.Printf("Failed to decode NSQ message: %v", err)
		return nil // Don't requeue the message if we can't decode it
	}

	// Process the message
	h.process(data)
	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (np *NSQPacket) Accept(pipe string, fn Process) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if np.producer == nil {
		np.mutex.RUnlock()
		return fmt.Errorf("not connected to NSQ server")
	}
	np.mutex.RUnlock()

	// Lock for thread safety during setup
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Check if already subscribed
	if _, exists := np.consumers[pipe]; exists {
		return fmt.Errorf("already subscribed to pipe %s", pipe)
	}

	// Create NSQ consumer configuration
	config := nsq.NewConfig()
	
	// Set connection timeout
	timeout := time.Duration(HPipeConfig.NSQF.NSQConnectTimeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second // Default timeout
	}
	config.DialTimeout = timeout
	
	// Set client ID if provided
	if HPipeConfig.NSQF.NSQClientID != "" {
		config.ClientID = HPipeConfig.NSQF.NSQClientID
	}
	
	// Set authentication if provided
	if HPipeConfig.NSQF.NSQAuthSecret != "" {
		config.AuthSecret = HPipeConfig.NSQF.NSQAuthSecret
	}
	
	// Set max in flight
	if HPipeConfig.NSQF.NSQMaxInFlight > 0 {
		config.MaxInFlight = HPipeConfig.NSQF.NSQMaxInFlight
	}
	
	// Set max attempts
	if HPipeConfig.NSQF.NSQMaxAttempts > 0 {
		config.MaxAttempts = uint16(HPipeConfig.NSQF.NSQMaxAttempts)
	}
	
	// Set requeue delay
	if HPipeConfig.NSQF.NSQRequeueDelay > 0 {
		config.DefaultRequeueDelay = time.Duration(HPipeConfig.NSQF.NSQRequeueDelay) * time.Second
	}

	// Create a unique channel name based on client ID
	channelName := "hybridpipe"
	if HPipeConfig.NSQF.NSQClientID != "" {
		channelName = HPipeConfig.NSQF.NSQClientID
	}

	// Create and connect the consumer
	consumer, err := nsq.NewConsumer(pipe, channelName, config)
	if err != nil {
		return fmt.Errorf("failed to create NSQ consumer for pipe %s: %w", pipe, err)
	}

	// Create a message handler
	handler := &nsqMessageHandler{process: fn}
	consumer.AddHandler(handler)

	// Store the handler
	np.handlers[pipe] = handler

	// Connect to NSQ
	if HPipeConfig.NSQF.NSQLookupdAddresses != "" {
		// Connect via nsqlookupd
		addresses := strings.Split(HPipeConfig.NSQF.NSQLookupdAddresses, ",")
		if err := consumer.ConnectToNSQLookupds(addresses); err != nil {
			delete(np.handlers, pipe)
			return fmt.Errorf("failed to connect to NSQ lookupd for pipe %s: %w", pipe, err)
		}
	} else {
		// Connect directly to nsqd
		nsqdAddress := fmt.Sprintf("%s:%d", HPipeConfig.NSQF.NSQDAddress, HPipeConfig.NSQF.NSQDPort)
		if err := consumer.ConnectToNSQD(nsqdAddress); err != nil {
			delete(np.handlers, pipe)
			return fmt.Errorf("failed to connect to NSQ daemon for pipe %s: %w", pipe, err)
		}
	}

	// Store the consumer
	np.consumers[pipe] = consumer

	log.Printf("Subscribed to NSQ pipe %s with channel %s", pipe, channelName)
	return nil
}

// Remove unsubscribes from the specified pipe.
func (np *NSQPacket) Remove(pipe string) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if np.producer == nil {
		np.mutex.RUnlock()
		return fmt.Errorf("not connected to NSQ server")
	}
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Check if the pipe is subscribed
	consumer, exists := np.consumers[pipe]
	if !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}

	// Stop the consumer
	consumer.Stop()

	// Remove the consumer and handler
	delete(np.consumers, pipe)
	delete(np.handlers, pipe)

	log.Printf("Unsubscribed from NSQ pipe %s", pipe)
	return nil
}

// Close terminates the connection to the NSQ server.
func (np *NSQPacket) Close() {
	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Check if connected
	if np.producer == nil {
		return
	}

	// Stop all consumers
	for pipe, consumer := range np.consumers {
		consumer.Stop()
		log.Printf("Stopped NSQ consumer for pipe %s", pipe)
	}

	// Clear the consumers and handlers maps
	np.consumers = make(map[string]*nsq.Consumer)
	np.handlers = make(map[string]nsq.Handler)

	// Stop the producer
	np.producer.Stop()
	np.producer = nil

	log.Printf("Disconnected from NSQ server")
}
