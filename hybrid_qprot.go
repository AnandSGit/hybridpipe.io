package hybridpipe

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"pack.ag/amqp"
)

// QpidPacket implements the HybridPipe interface for Apache Qpid messaging system.
type QpidPacket struct {
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
}

// createQpidTLSConfig creates a TLS configuration for Qpid connections.
func createQpidTLSConfig(clientCert, clientKey, caCert string) (*tls.Config, error) {
	// Create a basic TLS configuration
	tlsConfig := tls.Config{
		MinVersion: tls.VersionTLS12, // Enforce minimum TLS version for security
	}

	// Skip TLS setup if certificates are not provided
	if clientCert == "" || clientKey == "" || caCert == "" {
		log.Printf("Warning: TLS certificates not provided, using insecure connection")
		return nil, nil
	}

	// Load client certificate
	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	// Load CA certificate
	caCertData, err := os.ReadFile(caCert)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	// Create certificate pool and add CA certificate
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCertData) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}
	tlsConfig.RootCAs = caCertPool

	return &tlsConfig, nil
}

// Connect establishes a connection to the Qpid server using the configuration in HPipeConfig.
func (qp *QpidPacket) Connect() error {
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
	var tlsConfig *tls.Config
	var err error
	if HPipeConfig.QpidF.QpidCertFile != "" && HPipeConfig.QpidF.QpidKeyFile != "" && HPipeConfig.QpidF.QpidCAFile != "" {
		tlsConfig, err = createQpidTLSConfig(
			HPipeConfig.QpidF.QpidCertFile,
			HPipeConfig.QpidF.QpidKeyFile,
			HPipeConfig.QpidF.QpidCAFile,
		)
		if err != nil {
			return fmt.Errorf("failed to create TLS configuration: %w", err)
		}
	}

	// Build the Qpid server URL
	serverURL := fmt.Sprintf("amqp://%s:%d", HPipeConfig.QpidF.QpidServer, HPipeConfig.QpidF.QpidPort)

	// Set connection options
	opts := []amqp.ConnOption{
		amqp.ConnIdleTimeout(time.Duration(HPipeConfig.QpidF.QpidTimeout) * time.Second),
	}

	// Add TLS if configured
	if tlsConfig != nil {
		// Note: pack.ag/amqp doesn't have a direct TLS option, so we'd need to use a custom dialer
		// This is a placeholder for when we implement actual TLS support
		log.Printf("TLS support is configured but not fully implemented for Qpid")
	}

	// Add SASL authentication if username and password are provided
	if HPipeConfig.QpidF.QpidUsername != "" && HPipeConfig.QpidF.QpidPassword != "" {
		opts = append(opts, amqp.ConnSASLPlain(HPipeConfig.QpidF.QpidUsername, HPipeConfig.QpidF.QpidPassword))
	}

	// Connect to the Qpid server
	client, err := amqp.Dial(serverURL, opts...)
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
	qp.client = client
	qp.session = session

	log.Printf("Connected to Qpid server at %s", serverURL)
	return nil
}

// Dispatch publishes a message to the specified pipe.
func (qp *QpidPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection
	qp.mutex.RLock()
	if qp.client == nil || qp.session == nil {
		qp.mutex.RUnlock()
		return fmt.Errorf("not connected to Qpid server")
	}
	qp.mutex.RUnlock()

	// Get or create a sender for the pipe
	qp.mutex.Lock()
	sender, exists := qp.senders[pipe]
	if !exists {
		var err error
		sender, err = qp.session.NewSender(
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
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(HPipeConfig.QpidF.QpidTimeout)*time.Second)
	defer cancel()

	// Create and send the message
	msg := amqp.NewMessage(bytes)
	err = sender.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send message to pipe %s: %w", pipe, err)
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (qp *QpidPacket) Accept(pipe string, fn Process) error {
	// Validate connection
	qp.mutex.RLock()
	if qp.client == nil || qp.session == nil {
		qp.mutex.RUnlock()
		return fmt.Errorf("not connected to Qpid server")
	}
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

	// Create a new receiver for the pipe
	receiver, err := qp.session.NewReceiver(
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
	go qp.readMessages(ctx, pipe, receiver, fn)

	log.Printf("Subscribed to Qpid pipe %s", pipe)
	return nil
}

// readMessages reads messages from Qpid in a loop until the context is canceled.
func (qp *QpidPacket) readMessages(ctx context.Context, pipe string, receiver *amqp.Receiver, fn Process) {
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
			msg, err := receiver.Receive(receiveCtx)
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
				// If the error is fatal, wait before retrying
				if err != context.Canceled {
					time.Sleep(1 * time.Second)
				}
				continue
			}

			// Accept the message
			msg.Accept()

			// Decode the message
			var data interface{}
			if err := Decode(msg.GetData(), &data); err != nil {
				log.Printf("Failed to decode message from pipe %s: %v", pipe, err)
				continue
			}

			// Process the message
			fn(data)
		}
	}
}

// Remove unsubscribes from the specified pipe.
func (qp *QpidPacket) Remove(pipe string) error {
	// Validate connection
	qp.mutex.RLock()
	if qp.client == nil || qp.session == nil {
		qp.mutex.RUnlock()
		return fmt.Errorf("not connected to Qpid server")
	}
	qp.mutex.RUnlock()

	// Lock for thread safety
	qp.mutex.Lock()
	defer qp.mutex.Unlock()

	// Check if the pipe is subscribed
	receiver, exists := qp.receivers[pipe]
	if !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
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

// Close terminates the connection to the Qpid server.
func (qp *QpidPacket) Close() {
	// Check if connected
	qp.mutex.Lock()
	defer qp.mutex.Unlock()

	if qp.client == nil {
		return
	}

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

	// Close the client
	if err := qp.client.Close(); err != nil {
		log.Printf("Error closing Qpid connection: %v", err)
	}
	qp.client = nil

	log.Printf("Closed Qpid connection")
}
