package hybridpipe

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	nats "github.com/nats-io/nats.go"
)

// NatsPacket implements the HybridPipe interface for NATS messaging system.
type NatsPacket struct {
	// HandleConn is the connection to the NATS server
	HandleConn *nats.Conn
	// PipeHandle maps pipe names to their subscriptions
	PipeHandle map[string]*nats.Subscription
	// mutex protects concurrent access to the connection and subscriptions
	mutex sync.RWMutex
}

// Connect establishes a connection to the NATS server using the configuration in HPipeConfig.
func (np *NatsPacket) Connect() error {
	// Build the NATS server URL with port
	serverURL := HPipeConfig.NServer
	if HPipeConfig.NLport > 0 {
		serverURL = fmt.Sprintf("%s:%d", HPipeConfig.NServer, HPipeConfig.NLport)
	}

	// Set up connection options
	options := []nats.Option{
		nats.Name(getClientName()),
		nats.ReconnectWait(time.Duration(HPipeConfig.NReconnectWait) * time.Second),
		nats.MaxReconnects(HPipeConfig.NMaxAttempt),
		nats.Timeout(time.Duration(HPipeConfig.NTimeout) * time.Second),
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
	if HPipeConfig.NATSCertFile != "" && HPipeConfig.NATSKeyFile != "" && HPipeConfig.NATSCAFile != "" {
		options = append(options,
			nats.Secure(),
			nats.ClientCert(HPipeConfig.NATSCertFile, HPipeConfig.NATSKeyFile),
			nats.RootCAs(HPipeConfig.NATSCAFile),
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

// Dispatch publishes a message to the specified pipe.
func (np *NatsPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if np.HandleConn == nil {
		np.mutex.RUnlock()
		return fmt.Errorf("not connected to NATS server")
	}
	np.mutex.RUnlock()

	// Encode the data
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Publish the message
	if err = np.HandleConn.Publish(pipe, bytes); err != nil {
		return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
	}

	// Flush to ensure the message is sent
	if err = np.HandleConn.Flush(); err != nil {
		return fmt.Errorf("failed to flush NATS connection: %w", err)
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (np *NatsPacket) Accept(pipe string, fn Process) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if np.HandleConn == nil {
		np.mutex.RUnlock()
		return fmt.Errorf("not connected to NATS server")
	}
	np.mutex.RUnlock()

	// Create a queue group name based on the process name
	queueGroup := getClientName()

	// Subscribe to the pipe with a queue group
	sub, err := np.HandleConn.QueueSubscribe(pipe, queueGroup, func(msg *nats.Msg) {
		// Decode the message
		var data interface{}
		if err := Decode(msg.Data, &data); err != nil {
			log.Printf("Failed to decode message from pipe %s: %v", pipe, err)
			return
		}

		// Process the message
		fn(data)
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to pipe %s: %w", pipe, err)
	}

	// Store the subscription with write lock
	np.mutex.Lock()
	np.PipeHandle[pipe] = sub
	np.mutex.Unlock()

	log.Printf("Subscribed to pipe %s with queue group %s", pipe, queueGroup)
	return nil
}

// Remove unsubscribes from the specified pipe.
func (np *NatsPacket) Remove(pipe string) error {
	// Validate connection with read lock
	np.mutex.RLock()
	if np.HandleConn == nil {
		np.mutex.RUnlock()
		return fmt.Errorf("not connected to NATS server")
	}
	np.mutex.RUnlock()

	// Check if the pipe is subscribed with read lock
	np.mutex.RLock()
	sub, exists := np.PipeHandle[pipe]
	if !exists {
		np.mutex.RUnlock()
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}
	np.mutex.RUnlock()

	// Unsubscribe from the pipe
	if err := sub.Unsubscribe(); err != nil {
		return fmt.Errorf("failed to unsubscribe from pipe %s: %w", pipe, err)
	}

	// Remove the subscription from the map with write lock
	np.mutex.Lock()
	delete(np.PipeHandle, pipe)
	np.mutex.Unlock()

	log.Printf("Unsubscribed from pipe %s", pipe)
	return nil
}

// Close terminates the connection to the NATS server.
func (np *NatsPacket) Close() {
	// Check if connected with read lock
	np.mutex.RLock()
	if np.HandleConn == nil {
		np.mutex.RUnlock()
		return
	}
	np.mutex.RUnlock()

	// Lock for thread safety
	np.mutex.Lock()
	defer np.mutex.Unlock()

	// Close all subscriptions
	for pipe, sub := range np.PipeHandle {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("Failed to unsubscribe from pipe %s: %v", pipe, err)
		}
	}

	// Clear the subscription map
	np.PipeHandle = make(map[string]*nats.Subscription)

	// Close the connection
	np.HandleConn.Close()
	log.Printf("Closed NATS connection")
}

// getClientName returns a unique client name for NATS connection.
func getClientName() string {
	// Use the process name as the base
	name := "hybridpipe"
	if len(os.Args) > 0 {
		name = os.Args[0]
	}

	// Add process ID for uniqueness
	return name + "-" + strconv.Itoa(os.Getpid())
}
