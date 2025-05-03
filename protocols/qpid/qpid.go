// Package qpid provides an implementation of the HybridPipe interface for Apache Qpid messaging system.
package qpid

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"hybridpipe.io/core"
	"pack.ag/amqp"
)

// Packet implements the HybridPipe interface for Apache Qpid messaging system.
type Packet struct {
	// client is the connection to the Qpid server
	client *amqp.Client
	// session is the AMQP session
	session *amqp.Session
	// senders maps pipe names to their senders
	senders map[string]*amqp.Sender
	// receivers maps pipe names to their receivers
	receivers map[string]*amqp.Receiver
	// cancelFuncs maps pipe names to their context cancel functions
	cancelFuncs map[string]context.CancelFunc
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// config holds the Qpid configuration
	config *core.QpidConfig
}

// New creates a new Qpid packet with the specified configuration.
func New(config *core.QpidConfig) *Packet {
	return &Packet{
		senders:     make(map[string]*amqp.Sender),
		receivers:   make(map[string]*amqp.Receiver),
		cancelFuncs: make(map[string]context.CancelFunc),
		config:      config,
	}
}

// createTLSConfig creates a TLS configuration from certificate files.
func createTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" || caFile == "" {
		return nil, nil
	}

	// Load client cert
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA cert
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	return tlsConfig, nil
}

// Connect establishes a connection to the Qpid server using the configuration.
func (qp *Packet) Connect() error {
	// Get configuration if not already set
	if qp.config == nil {
		config, err := core.GetProtocolConfig(core.QPID)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		qpidConfig := config.(core.QpidConfig)
		qp.config = &qpidConfig
	}

	// Initialize maps if needed
	qp.mutex.Lock()
	if qp.senders == nil {
		qp.senders = make(map[string]*amqp.Sender)
	}
	if qp.receivers == nil {
		qp.receivers = make(map[string]*amqp.Receiver)
	}
	if qp.cancelFuncs == nil {
		qp.cancelFuncs = make(map[string]context.CancelFunc)
	}
	qp.mutex.Unlock()

	// Create TLS configuration if certificates are provided
	tlsConfig, err := createTLSConfig(
		qp.config.QpidCertFile,
		qp.config.QpidKeyFile,
		qp.config.QpidCAFile,
	)
	if err != nil {
		return fmt.Errorf("failed to create TLS configuration: %w", err)
	}

	// Set connection options
	options := []amqp.ConnOption{
		amqp.ConnIdleTimeout(30 * time.Second),
		amqp.ConnMaxFrameSize(4294967295),
	}

	// Add TLS options if TLS is enabled
	if qp.config.QpidTLSEnabled && tlsConfig != nil {
		// Note: The actual implementation depends on the amqp package version
		// For pack.ag/amqp, use:
		// options = append(options, amqp.ConnTLS(tlsConfig))
		// For Azure/go-amqp, use:
		options = append(options, amqp.ConnSASLAnonymous(), amqp.ConnTLS(true))
	}

	// Connect to the server
	serverURL := fmt.Sprintf("amqp://%s:%d", qp.config.QpidServer, qp.config.QpidPort)
	client, err := amqp.Dial(serverURL, options...)
	if err != nil {
		return fmt.Errorf("failed to connect to Qpid server %s: %w", serverURL, err)
	}

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to create AMQP session: %w", err)
	}

	// Store the client and session
	qp.mutex.Lock()
	qp.client = client
	qp.session = session
	qp.mutex.Unlock()

	log.Printf("Connected to Qpid server at %s", serverURL)
	return nil
}

// Dispatch sends a message to the specified pipe.
func (qp *Packet) Dispatch(pipe string, data any) error {
	// Validate connection with read lock
	qp.mutex.RLock()
	if qp.client == nil || qp.session == nil {
		qp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	session := qp.session
	qp.mutex.RUnlock()

	// Get or create sender for the pipe
	qp.mutex.Lock()
	sender, exists := qp.senders[pipe]
	if !exists {
		var err error
		sender, err = session.NewSender(
			amqp.LinkTargetAddress(pipe),
		)
		if err != nil {
			qp.mutex.Unlock()
			return fmt.Errorf("failed to create sender for pipe %s: %w", pipe, err)
		}
		qp.senders[pipe] = sender
	}
	qp.mutex.Unlock()

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send the message
	err = sender.Send(ctx, amqp.NewMessage(bytes))
	if err != nil {
		return fmt.Errorf("failed to send message to pipe %s: %w", pipe, err)
	}

	return nil
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (qp *Packet) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	// Validate connection with read lock
	qp.mutex.RLock()
	if qp.client == nil || qp.session == nil {
		qp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	session := qp.session
	qp.mutex.RUnlock()

	// Get or create sender for the pipe
	qp.mutex.Lock()
	sender, exists := qp.senders[pipe]
	if !exists {
		var err error
		sender, err = session.NewSender(
			amqp.LinkTargetAddress(pipe),
		)
		if err != nil {
			qp.mutex.Unlock()
			return fmt.Errorf("failed to create sender for pipe %s: %w", pipe, err)
		}
		qp.senders[pipe] = sender
	}
	qp.mutex.Unlock()

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Send the message with the provided context
	err = sender.Send(ctx, amqp.NewMessage(bytes))
	if err != nil {
		return fmt.Errorf("failed to send message to pipe %s: %w", pipe, err)
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (qp *Packet) Accept(pipe string, fn func(any)) error {
	// Create a wrapper that converts the function to the expected type
	wrapper := func(data []byte) error {
		var decoded any
		if err := core.Decode(data, &decoded); err != nil {
			return err
		}
		fn(decoded)
		return nil
	}

	return qp.Subscribe(pipe, wrapper)
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (qp *Packet) Subscribe(pipe string, callback core.Process) error {
	// Validate connection with read lock
	qp.mutex.RLock()
	if qp.client == nil || qp.session == nil {
		qp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	session := qp.session
	qp.mutex.RUnlock()

	// Lock for thread safety
	qp.mutex.Lock()
	defer qp.mutex.Unlock()

	// Cancel existing receiver if any
	if cancel, exists := qp.cancelFuncs[pipe]; exists {
		cancel()
		delete(qp.cancelFuncs, pipe)
	}

	// Close existing receiver if any
	if receiver, exists := qp.receivers[pipe]; exists {
		receiver.Close(context.Background())
		delete(qp.receivers, pipe)
	}

	// Create a receiver for the pipe
	receiver, err := session.NewReceiver(
		amqp.LinkSourceAddress(pipe),
		amqp.LinkCredit(10),
	)
	if err != nil {
		return fmt.Errorf("failed to create receiver for pipe %s: %w", pipe, err)
	}

	// Store the receiver
	qp.receivers[pipe] = receiver

	// Create a context with cancel function for the receiver goroutine
	ctx, cancel := context.WithCancel(context.Background())
	qp.cancelFuncs[pipe] = cancel

	// Start a goroutine to handle messages
	go qp.readMessages(ctx, pipe, receiver, callback)

	log.Printf("Subscribed to Qpid pipe %s", pipe)
	return nil
}

// SubscribeWithContext registers a callback with context for cancellation.
func (qp *Packet) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Validate connection with read lock
	qp.mutex.RLock()
	if qp.client == nil || qp.session == nil {
		qp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	session := qp.session
	qp.mutex.RUnlock()

	// Lock for thread safety
	qp.mutex.Lock()
	defer qp.mutex.Unlock()

	// Cancel existing receiver if any
	if cancel, exists := qp.cancelFuncs[pipe]; exists {
		cancel()
		delete(qp.cancelFuncs, pipe)
	}

	// Close existing receiver if any
	if receiver, exists := qp.receivers[pipe]; exists {
		receiver.Close(context.Background())
		delete(qp.receivers, pipe)
	}

	// Create a receiver for the pipe
	receiver, err := session.NewReceiver(
		amqp.LinkSourceAddress(pipe),
		amqp.LinkCredit(10),
	)
	if err != nil {
		return fmt.Errorf("failed to create receiver for pipe %s: %w", pipe, err)
	}

	// Store the receiver
	qp.receivers[pipe] = receiver

	// Create a context with cancel function for the receiver goroutine
	receiverCtx, cancel := context.WithCancel(ctx)
	qp.cancelFuncs[pipe] = cancel

	// Start a goroutine to handle messages
	go qp.readMessages(receiverCtx, pipe, receiver, callback)

	log.Printf("Subscribed to Qpid pipe %s with context", pipe)
	return nil
}

// readMessages reads messages from Qpid in a loop until the context is canceled.
func (qp *Packet) readMessages(ctx context.Context, pipe string, receiver *amqp.Receiver, callback core.Process) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in Qpid reader for pipe %s: %v", pipe, r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping Qpid reader for pipe %s", pipe)
			return
		default:
			// Create a context with timeout for receiving messages
			receiveCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
			message, err := receiver.Receive(receiveCtx)
			cancel()

			// Check for context cancellation
			if ctx.Err() != nil {
				return
			}

			// Handle timeout and continue
			if err == context.DeadlineExceeded {
				continue
			}

			// Handle other errors
			if err != nil {
				log.Printf("Error receiving message from pipe %s: %v", pipe, err)
				// If the error is fatal, break the loop
				if err != context.Canceled {
					time.Sleep(1 * time.Second) // Wait before retrying
				}
				continue
			}

			// Accept the message
			// Note: The actual implementation depends on the amqp package version
			// For pack.ag/amqp, use:
			// message.Accept(context.Background())
			// For Azure/go-amqp, use:
			message.Accept()

			// Process the message
			if err := callback(message.GetData()); err != nil {
				log.Printf("Error processing message from pipe %s: %v", pipe, err)
			}
		}
	}
}

// Remove unsubscribes from the specified pipe.
func (qp *Packet) Remove(pipe string) error {
	return qp.Unsubscribe(pipe)
}

// Unsubscribe removes a subscription from the specified pipe.
func (qp *Packet) Unsubscribe(pipe string) error {
	// Validate connection with read lock
	qp.mutex.RLock()
	if qp.client == nil || qp.session == nil {
		qp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	qp.mutex.RUnlock()

	// Lock for thread safety
	qp.mutex.Lock()
	defer qp.mutex.Unlock()

	// Check if the pipe is subscribed
	receiver, exists := qp.receivers[pipe]
	if !exists {
		return core.ErrPipeNotFound
	}

	// Cancel the receiver goroutine
	if cancel, exists := qp.cancelFuncs[pipe]; exists {
		cancel()
		delete(qp.cancelFuncs, pipe)
	}

	// Close the receiver
	if err := receiver.Close(context.Background()); err != nil {
		log.Printf("Error closing receiver for pipe %s: %v", pipe, err)
	}
	delete(qp.receivers, pipe)

	log.Printf("Unsubscribed from Qpid pipe %s", pipe)
	return nil
}

// Disconnect closes the connection to the Qpid server.
func (qp *Packet) Disconnect() error {
	// Check if connected with read lock
	qp.mutex.RLock()
	if qp.client == nil {
		qp.mutex.RUnlock()
		return nil
	}
	client := qp.client
	qp.mutex.RUnlock()

	// Lock for thread safety
	qp.mutex.Lock()
	defer qp.mutex.Unlock()

	// Cancel all receiver goroutines
	for pipe, cancel := range qp.cancelFuncs {
		cancel()
		delete(qp.cancelFuncs, pipe)
		log.Printf("Canceled receiver for pipe %s", pipe)
	}

	// Close all receivers
	for pipe, receiver := range qp.receivers {
		if err := receiver.Close(context.Background()); err != nil {
			log.Printf("Error closing receiver for pipe %s: %v", pipe, err)
		}
	}
	qp.receivers = make(map[string]*amqp.Receiver)

	// Close all senders
	for pipe, sender := range qp.senders {
		if err := sender.Close(context.Background()); err != nil {
			log.Printf("Error closing sender for pipe %s: %v", pipe, err)
		}
	}
	qp.senders = make(map[string]*amqp.Sender)

	// Close the session
	if qp.session != nil {
		if err := qp.session.Close(context.Background()); err != nil {
			log.Printf("Error closing Qpid session: %v", err)
		}
		qp.session = nil
	}

	// Close the connection
	if err := client.Close(); err != nil {
		log.Printf("Error closing Qpid connection: %v", err)
		return err
	}
	qp.client = nil

	log.Printf("Disconnected from Qpid server")
	return nil
}

// Close terminates the connection to the Qpid server.
func (qp *Packet) Close() error {
	return qp.Disconnect()
}

// IsConnected returns true if the connection to the Qpid server is active.
func (qp *Packet) IsConnected() bool {
	qp.mutex.RLock()
	defer qp.mutex.RUnlock()
	return qp.client != nil && qp.session != nil
}
