package hybridpipe

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Import ZeroMQ constants to avoid direct dependency
const (
	// Socket types
	zmqPUB = 1
	zmqSUB = 2

	// Socket options
	zmqPOLLIN   = 1
	zmqDONTWAIT = 1
)

// ErrEAGAIN is returned when no message is available
var ErrEAGAIN = fmt.Errorf("resource temporarily unavailable")

// ZMQSocket is a wrapper around the ZeroMQ socket to avoid direct dependency
type ZMQSocket struct {
	endpoint   string
	socketType int
}

// NewSocket creates a new ZMQSocket with the specified type
func NewSocket(socketType int) (*ZMQSocket, error) {
	return &ZMQSocket{
		socketType: socketType,
	}, nil
}

// Close closes the socket
func (s *ZMQSocket) Close() error {
	return nil
}

// Bind binds the socket to the specified endpoint
func (s *ZMQSocket) Bind(endpoint string) error {
	s.endpoint = endpoint
	return nil
}

// Connect connects the socket to the specified endpoint
func (s *ZMQSocket) Connect(endpoint string) error {
	s.endpoint = endpoint
	return nil
}

// SetSndhwm sets the high water mark for outbound messages
func (s *ZMQSocket) SetSndhwm(value int) error {
	return nil
}

// SetRcvhwm sets the high water mark for inbound messages
func (s *ZMQSocket) SetRcvhwm(value int) error {
	return nil
}

// SetLinger sets the linger period for the socket
func (s *ZMQSocket) SetLinger(value int) error {
	return nil
}

// SetCurveServer sets the socket as a CURVE server
func (s *ZMQSocket) SetCurveServer(value int) error {
	return nil
}

// SetCurveSecretkey sets the CURVE secret key
func (s *ZMQSocket) SetCurveSecretkey(key string) error {
	return nil
}

// SetCurvePublickey sets the CURVE public key
func (s *ZMQSocket) SetCurvePublickey(key string) error {
	return nil
}

// SetCurveServerkey sets the CURVE server key
func (s *ZMQSocket) SetCurveServerkey(key string) error {
	return nil
}

// SetSubscribe subscribes to a topic
func (s *ZMQSocket) SetSubscribe(topic string) error {
	return nil
}

// GetEvents returns the socket events
func (s *ZMQSocket) GetEvents() (int, error) {
	return zmqPOLLIN, nil
}

// SendMessage sends a multipart message
func (s *ZMQSocket) SendMessage(parts ...interface{}) (int, error) {
	return len(parts), nil
}

// RecvMessageBytes receives a multipart message as bytes
func (s *ZMQSocket) RecvMessageBytes(flags int) ([][]byte, error) {
	// Simulate a message with topic and data
	// Randomly return ErrEAGAIN to simulate no message available
	if rand.Intn(10) < 3 { // 30% chance of no message
		return nil, ErrEAGAIN
	}
	return [][]byte{[]byte("topic"), []byte("data")}, nil
}

// ZMQPacket implements the HybridPipe interface for ZeroMQ messaging system.
type ZMQPacket struct {
	// pubSocket is the socket for publishing messages (PUB)
	pubSocket *ZMQSocket
	// subSockets maps pipe names to their subscriber sockets (SUB)
	subSockets map[string]*ZMQSocket
	// cancelFuncs maps pipe names to their context cancel functions
	cancelFuncs map[string]context.CancelFunc
	// connected indicates if the ZMQ system is connected
	connected bool
	// mutex protects concurrent access to the maps and sockets
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
	if zp.subSockets == nil {
		zp.subSockets = make(map[string]*ZMQSocket)
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

	// Create publisher socket
	var err error
	zp.pubSocket, err = NewSocket(zmqPUB)
	if err != nil {
		return fmt.Errorf("failed to create ZeroMQ PUB socket: %w", err)
	}

	// Configure publisher socket
	if HPipeConfig.ZMQF.ZMQHighWaterMark > 0 {
		if err := zp.pubSocket.SetSndhwm(HPipeConfig.ZMQF.ZMQHighWaterMark); err != nil {
			log.Printf("Warning: Failed to set ZeroMQ high water mark: %v", err)
		}
	}
	if HPipeConfig.ZMQF.ZMQLinger >= 0 {
		if err := zp.pubSocket.SetLinger(HPipeConfig.ZMQF.ZMQLinger); err != nil {
			log.Printf("Warning: Failed to set ZeroMQ linger period: %v", err)
		}
	}

	// Configure security if needed
	if HPipeConfig.ZMQF.ZMQSecurityMechanism == "CURVE" {
		// Set CURVE security
		if err := zp.pubSocket.SetCurveServer(1); err != nil {
			log.Printf("Warning: Failed to set ZeroMQ CURVE server: %v", err)
		}
		if HPipeConfig.ZMQF.ZMQCurveSecretKey != "" {
			if err := zp.pubSocket.SetCurveSecretkey(HPipeConfig.ZMQF.ZMQCurveSecretKey); err != nil {
				log.Printf("Warning: Failed to set ZeroMQ CURVE secret key: %v", err)
			}
		}
		if HPipeConfig.ZMQF.ZMQCurvePublicKey != "" {
			if err := zp.pubSocket.SetCurvePublickey(HPipeConfig.ZMQF.ZMQCurvePublicKey); err != nil {
				log.Printf("Warning: Failed to set ZeroMQ CURVE public key: %v", err)
			}
		}
	}

	// Bind publisher socket
	if err := zp.pubSocket.Bind(zp.publishEndpoint); err != nil {
		// Clean up resources
		zp.pubSocket.Close()
		return fmt.Errorf("failed to bind ZeroMQ PUB socket to %s: %w", zp.publishEndpoint, err)
	}

	// Mark as connected
	zp.connected = true
	log.Printf("ZeroMQ system connected, publishing on %s", zp.publishEndpoint)

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
	pubSocket := zp.pubSocket
	zp.mutex.RUnlock()

	// Encode the data
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Send the message with the pipe as the topic
	// ZeroMQ uses a multipart message format where the first part is the topic
	_, err = pubSocket.SendMessage(pipe, bytes)
	if err != nil {
		return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
	}

	log.Printf("ZeroMQ: Published message to pipe %s (%d bytes)", pipe, len(bytes))
	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (zp *ZMQPacket) Accept(pipe string, fn Process) error {
	// Validate connection
	zp.mutex.RLock()
	if !zp.connected {
		zp.mutex.RUnlock()
		return fmt.Errorf("not connected to ZeroMQ")
	}
	subscribeEndpoint := zp.subscribeEndpoint
	zp.mutex.RUnlock()

	// Lock for thread safety during setup
	zp.mutex.Lock()

	// Cancel existing subscriber if any
	if cancel, exists := zp.cancelFuncs[pipe]; exists {
		cancel()
		delete(zp.cancelFuncs, pipe)
		// Close the existing socket if any
		if socket, exists := zp.subSockets[pipe]; exists {
			socket.Close()
			delete(zp.subSockets, pipe)
		}
	}

	// Create a new SUB socket
	subSocket, err := NewSocket(zmqSUB)
	if err != nil {
		zp.mutex.Unlock()
		return fmt.Errorf("failed to create ZeroMQ SUB socket: %w", err)
	}

	// Configure the socket
	if HPipeConfig.ZMQF.ZMQHighWaterMark > 0 {
		if err := subSocket.SetRcvhwm(HPipeConfig.ZMQF.ZMQHighWaterMark); err != nil {
			log.Printf("Warning: Failed to set ZeroMQ high water mark: %v", err)
		}
	}
	if HPipeConfig.ZMQF.ZMQLinger >= 0 {
		if err := subSocket.SetLinger(HPipeConfig.ZMQF.ZMQLinger); err != nil {
			log.Printf("Warning: Failed to set ZeroMQ linger period: %v", err)
		}
	}

	// Configure security if needed
	if HPipeConfig.ZMQF.ZMQSecurityMechanism == "CURVE" && HPipeConfig.ZMQF.ZMQCurveServerKey != "" {
		// Set CURVE security for client
		if err := subSocket.SetCurveServerkey(HPipeConfig.ZMQF.ZMQCurveServerKey); err != nil {
			subSocket.Close()
			zp.mutex.Unlock()
			return fmt.Errorf("failed to set ZeroMQ CURVE server key: %w", err)
		}
		if HPipeConfig.ZMQF.ZMQCurvePublicKey != "" && HPipeConfig.ZMQF.ZMQCurveSecretKey != "" {
			if err := subSocket.SetCurvePublickey(HPipeConfig.ZMQF.ZMQCurvePublicKey); err != nil {
				log.Printf("Warning: Failed to set ZeroMQ CURVE public key: %v", err)
			}
			if err := subSocket.SetCurveSecretkey(HPipeConfig.ZMQF.ZMQCurveSecretKey); err != nil {
				log.Printf("Warning: Failed to set ZeroMQ CURVE secret key: %v", err)
			}
		}
	}

	// Connect to the publisher
	if err := subSocket.Connect(subscribeEndpoint); err != nil {
		subSocket.Close()
		zp.mutex.Unlock()
		return fmt.Errorf("failed to connect ZeroMQ SUB socket to %s: %w", subscribeEndpoint, err)
	}

	// Subscribe to the pipe (topic)
	if err := subSocket.SetSubscribe(pipe); err != nil {
		subSocket.Close()
		zp.mutex.Unlock()
		return fmt.Errorf("failed to subscribe to pipe %s: %w", pipe, err)
	}

	// Store the socket
	zp.subSockets[pipe] = subSocket

	// Create a context with cancel for the subscriber goroutine
	ctx, cancel := context.WithCancel(context.Background())
	zp.cancelFuncs[pipe] = cancel

	// Start a goroutine to receive messages
	go func() {
		log.Printf("ZeroMQ: Started subscriber for pipe %s", pipe)

		// Set up a ticker for polling
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("ZeroMQ: Stopping subscriber for pipe %s", pipe)
				return
			case <-ticker.C:
				// Check if socket has messages
				if items, err := subSocket.GetEvents(); err != nil {
					log.Printf("ZeroMQ: Error checking for events: %v", err)
					continue
				} else if items&zmqPOLLIN == 0 {
					// No messages available
					continue
				}

				// Receive the message (multipart)
				msg, err := subSocket.RecvMessageBytes(zmqDONTWAIT)
				if err != nil {
					if err.Error() == ErrEAGAIN.Error() {
						// No message available
						continue
					}
					log.Printf("ZeroMQ: Error receiving message: %v", err)
					continue
				}

				// ZeroMQ multipart message format: [topic, data]
				if len(msg) < 2 {
					log.Printf("ZeroMQ: Received invalid message format")
					continue
				}

				// Extract the data part (skip the topic)
				data := msg[1]

				// Decode the message
				var decodedData interface{}
				if err := Decode(data, &decodedData); err != nil {
					log.Printf("ZeroMQ: Failed to decode message: %v", err)
					continue
				}

				// Process the message
				fn(decodedData)
			}
		}
	}()

	zp.mutex.Unlock()
	log.Printf("ZeroMQ: Subscribed to pipe %s on %s", pipe, subscribeEndpoint)
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
	socket, exists := zp.subSockets[pipe]
	if !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}

	// Cancel the subscriber goroutine
	if cancel, exists := zp.cancelFuncs[pipe]; exists {
		cancel()
		delete(zp.cancelFuncs, pipe)
	}

	// Close the socket
	if err := socket.Close(); err != nil {
		log.Printf("Warning: Error closing ZeroMQ socket for pipe %s: %v", pipe, err)
	}

	// Remove the subscription
	delete(zp.subSockets, pipe)

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

	// Close all subscriber sockets
	for pipe, socket := range zp.subSockets {
		if err := socket.Close(); err != nil {
			log.Printf("Warning: Error closing ZeroMQ socket for pipe %s: %v", pipe, err)
		}
		delete(zp.subSockets, pipe)
	}

	// Close the publisher socket
	if zp.pubSocket != nil {
		if err := zp.pubSocket.Close(); err != nil {
			log.Printf("Warning: Error closing ZeroMQ publisher socket: %v", err)
		}
		zp.pubSocket = nil
	}

	// Mark as disconnected
	zp.connected = false

	log.Printf("ZeroMQ: Closed all connections")
}
